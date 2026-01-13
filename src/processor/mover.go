package processor

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MoveFile move um arquivo do source para o destination directory
// aplica a estratégia de conflito especificada se o arquivo já existir
func MoveFile(sourcePath, destDir, conflictStrategy string, logger *slog.Logger) error {
	filename := filepath.Base(sourcePath)

	// Verificar se arquivo fonte ainda existe
	sourceInfo, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		logger.Warn("Source file no longer exists, skipping", "file", sourcePath)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	if sourceInfo.IsDir() {
		logger.Warn("Source is a directory, not a file, skipping", "path", sourcePath)
		return nil
	}

	logger.Debug("Starting file move",
		"source", sourcePath,
		"dest_dir", destDir,
		"file_size", sourceInfo.Size())

	// Construir caminho de destino
	destPath := filepath.Join(destDir, filename)

	// Criar diretório de destino se não existir (antes de qualquer operação)
	logger.Debug("Ensuring destination directory exists", "path", destDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	logger.Debug("Destination directory ready", "path", destDir)

	// Verificar se arquivo de destino já existe
	if _, statErr := os.Stat(destPath); statErr == nil {
		// Arquivo já existe - aplicar estratégia de conflito
		logger.Debug("Destination file already exists, applying conflict strategy",
			"file", filename,
			"strategy", conflictStrategy)
		destPath, err = handleConflict(destPath, conflictStrategy, logger)
		if err != nil {
			return err
		}
	}

	// Tentar mover o arquivo
	logger.Debug("Attempting to move file", "from", sourcePath, "to", destPath)
	err = os.Rename(sourcePath, destPath)
	if err != nil {
		// Se falhar (provavelmente volumes diferentes), fazer copy + delete
		if strings.Contains(err.Error(), "cross-device") || strings.Contains(err.Error(), "invalid cross-device link") {
			logger.Debug("Cross-device move detected, using copy+delete", "file", filename)
			if err := copyFile(sourcePath, destPath); err != nil {
				return fmt.Errorf("failed to copy file: %w", err)
			}

			// Remover arquivo original apenas após cópia bem-sucedida
			if err := os.Remove(sourcePath); err != nil {
				logger.Warn("Failed to remove source file after copy", "file", sourcePath, "error", err)
			}
		} else {
			return fmt.Errorf("failed to move file: %w", err)
		}
	}

	logger.Info("File moved successfully",
		"file", filename,
		"destination", filepath.Base(destPath),
	)

	return nil
}

// handleConflict aplica a estratégia de conflito e retorna o novo destPath
func handleConflict(destPath, strategy string, logger *slog.Logger) (string, error) {
	filename := filepath.Base(destPath)

	switch strategy {
	case "overwrite":
		// Para overwrite, simplesmente retornar o mesmo destPath
		// os.Rename sobrescreve automaticamente no Unix/macOS
		// Para cross-device, a lógica de backup está no MoveFile
		logger.Debug("Existing file will be overwritten", "file", filename)
		return destPath, nil

	case "rename":
		// Gerar nome único
		newDestPath := generateUniqueName(destPath)
		logger.Debug("File renamed to avoid conflict",
			"original", filename,
			"new", filepath.Base(newDestPath),
		)
		return newDestPath, nil

	default:
		return "", fmt.Errorf("unknown conflict strategy: %s (use 'rename' or 'overwrite')", strategy)
	}
}

// generateUniqueName gera um nome único para o arquivo adicionando um contador
// Exemplo: document.pdf -> document_1.pdf -> document_2.pdf
func generateUniqueName(destPath string) string {
	dir := filepath.Dir(destPath)
	ext := filepath.Ext(destPath)
	nameWithoutExt := strings.TrimSuffix(filepath.Base(destPath), ext)

	counter := 1
	for {
		newName := fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		newPath := filepath.Join(dir, newName)

		// Verificar se esse nome já existe
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}

		counter++

		// Segurança: limitar tentativas
		if counter > 1000 {
			// Usar timestamp como fallback
			timestamp := time.Now().Format("20060102_150405")
			newName := fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext)
			return filepath.Join(dir, newName)
		}
	}
}

// copyFile copia um arquivo do source para destination
func copyFile(sourcePath, destPath string) error {
	// Abrir arquivo fonte
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Criar arquivo destino
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copiar conteúdo
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		// Se a cópia falhar, tentar remover arquivo de destino parcial
		destFile.Close()
		os.Remove(destPath)
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Sincronizar para garantir que dados foram escritos no disco
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	// Copiar permissões do arquivo original
	sourceInfo, err := sourceFile.Stat()
	if err == nil {
		if err := os.Chmod(destPath, sourceInfo.Mode()); err != nil {
			// Não é crítico, apenas logar
			// logger não está disponível aqui, então ignoramos
		}
	}

	return nil
}
