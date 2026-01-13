package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"gaa/file-organizer/src/config"
)

// FileWatcher monitora uma pasta e detecta novos arquivos
type FileWatcher struct {
	config  *config.Monitor
	logger  *slog.Logger
	watcher *fsnotify.Watcher
	delay   time.Duration
	doneCh  chan struct{}
}

// NewFileWatcher cria uma nova instância do file watcher
func NewFileWatcher(monitor *config.Monitor, delay time.Duration, logger *slog.Logger) (*FileWatcher, error) {
	// Criar watcher do fsnotify
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	fw := &FileWatcher{
		config:  monitor,
		logger:  logger,
		watcher: fsWatcher,
		delay:   delay,
		doneCh:  make(chan struct{}),
	}

	// Registrar o source_path
	if err := fw.addPath(monitor.SourcePath, monitor.Recursive); err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("failed to watch path: %w", err)
	}

	return fw, nil
}

// addPath adiciona um path ao watcher, recursivamente se necessário
func (fw *FileWatcher) addPath(path string, recursive bool) error {
	// Adicionar o path principal
	if err := fw.watcher.Add(path); err != nil {
		return err
	}

	fw.logger.Debug("Watching path", "path", path)

	// Se recursivo, adicionar todas as subpastas
	if recursive {
		err := filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				fw.logger.Warn("Error walking path", "path", walkPath, "error", err)
				return nil // Continuar mesmo com erro
			}

			// Adicionar apenas diretórios (exceto ocultos)
			if info.IsDir() && !strings.HasPrefix(filepath.Base(walkPath), ".") {
				if walkPath != path { // Não adicionar o path principal novamente
					if err := fw.watcher.Add(walkPath); err != nil {
						fw.logger.Warn("Failed to watch subdirectory", "path", walkPath, "error", err)
					} else {
						fw.logger.Debug("Watching subdirectory", "path", walkPath)
					}
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory tree: %w", err)
		}
	}

	return nil
}

// Start inicia o monitoramento de arquivos
func (fw *FileWatcher) Start() error {
	fw.logger.Info("Starting file watcher",
		"monitor", fw.config.Name,
		"path", fw.config.SourcePath,
		"recursive", fw.config.Recursive,
	)

	// Goroutine para processar eventos
	go fw.watchLoop()

	return nil
}

// watchLoop é a goroutine principal que escuta eventos do fsnotify
func (fw *FileWatcher) watchLoop() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return // Canal fechado
			}

			// Processar evento
			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return // Canal fechado
			}
			fw.logger.Error("Watcher error", "error", err)

		case <-fw.doneCh:
			fw.logger.Debug("Watcher stopping")
			return
		}
	}
}

// handleEvent processa um evento do fsnotify
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Filtro 1: Ignorar eventos Chmod
	if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		return
	}

	// Filtro 2: Aceitar apenas Create e Write
	if event.Op&fsnotify.Create != fsnotify.Create && event.Op&fsnotify.Write != fsnotify.Write {
		return
	}

	// Obter info do arquivo
	fileInfo, err := os.Stat(event.Name)
	if err != nil {
		if !os.IsNotExist(err) {
			fw.logger.Warn("Failed to stat file", "file", event.Name, "error", err)
		}
		return
	}

	// Filtro 3: Ignorar diretórios (processar apenas arquivos)
	if fileInfo.IsDir() {
		// Se for recursivo e for um novo diretório, adicionar ao watcher
		if fw.config.Recursive && event.Op&fsnotify.Create == fsnotify.Create {
			if err := fw.watcher.Add(event.Name); err != nil {
				fw.logger.Warn("Failed to watch new subdirectory", "path", event.Name, "error", err)
			} else {
				fw.logger.Debug("Now watching new subdirectory", "path", event.Name)
			}
		}
		return
	}

	// Filtro 4: Ignorar arquivos ocultos (começam com ".")
	filename := filepath.Base(event.Name)
	if strings.HasPrefix(filename, ".") {
		fw.logger.Debug("Ignoring hidden file", "file", filename)
		return
	}

	// Filtro 5: Ignorar arquivos temporários
	if fw.isTempFile(filename) {
		fw.logger.Debug("Ignoring temporary file", "file", filename)
		return
	}

	// Verificar se arquivo está pronto para ser processado
	fw.logger.Debug("File event detected", "file", event.Name, "op", event.Op.String())

	if fw.IsFileReady(event.Name) {
		fw.logger.Info("File ready for processing", "file", filename)
		// TODO: Fase 3 - Processar arquivo (MatchRule + MoveFile)
		// Por enquanto apenas loga
	} else {
		fw.logger.Warn("File not ready or locked", "file", filename)
	}
}

// isTempFile verifica se o arquivo é temporário
func (fw *FileWatcher) isTempFile(filename string) bool {
	tempExtensions := []string{
		".tmp",
		".temp",
		".crdownload", // Chrome downloads
		".part",       // Firefox downloads
		".download",
		".partial",
	}

	lowerFilename := strings.ToLower(filename)
	for _, ext := range tempExtensions {
		if strings.HasSuffix(lowerFilename, ext) {
			return true
		}
	}

	return false
}

// IsFileReady verifica se um arquivo está pronto para ser processado
// Implementa retry logic para lidar com arquivos sendo escritos
func (fw *FileWatcher) IsFileReady(path string) bool {
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		// Tentar abrir o arquivo em modo read-only
		file, err := os.OpenFile(path, os.O_RDONLY, 0)
		if err == nil {
			file.Close()

			// Verificar se o arquivo tem tamanho > 0
			fileInfo, err := os.Stat(path)
			if err != nil {
				fw.logger.Debug("File disappeared during check", "file", path)
				return false
			}

			// Aceitar arquivos com tamanho 0 (arquivos vazios são válidos)
			// mas logar para debug
			if fileInfo.Size() == 0 {
				fw.logger.Debug("File has zero size", "file", path)
			}

			return true // Arquivo está pronto
		}

		// Se for erro de permissão ou "not exist", não tentar novamente
		if os.IsNotExist(err) || os.IsPermission(err) {
			fw.logger.Debug("File not accessible", "file", path, "error", err)
			return false
		}

		// Arquivo pode estar sendo escrito, aguardar
		if i < maxRetries-1 {
			fw.logger.Debug("File busy, retrying...",
				"file", path,
				"attempt", i+1,
				"max_retries", maxRetries,
			)
			time.Sleep(fw.delay)
		}
	}

	return false // Arquivo travado ou corrompido após todas as tentativas
}

// Stop para o watcher gracefully
func (fw *FileWatcher) Stop() {
	fw.logger.Info("Stopping file watcher", "monitor", fw.config.Name)

	// Sinalizar para a goroutine parar
	close(fw.doneCh)

	// Fechar o watcher do fsnotify
	if err := fw.watcher.Close(); err != nil {
		fw.logger.Error("Error closing watcher", "error", err)
	}

	fw.logger.Debug("File watcher stopped", "monitor", fw.config.Name)
}
