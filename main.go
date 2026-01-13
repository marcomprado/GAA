package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gaa/file-organizer/src/config"
	"gaa/file-organizer/src/watcher"
)

func main() {
	// Parse CLI flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Carregar configuração
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validar configuração
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// Inicializar logger
	logger := config.InitLogger(cfg.Settings.LogLevel)
	logger.Info("File Organizer Daemon started",
		"version", "1.0.0",
		"monitors", len(cfg.Monitors),
		"max_workers", cfg.Settings.MaxWorkers,
	)

	// Obter delay da configuração
	delay, err := cfg.ParseDelayDuration()
	if err != nil {
		log.Fatalf("Failed to parse delay_before_move: %v", err)
	}

	// Mostrar configuração carregada
	for _, monitor := range cfg.Monitors {
		logger.Info("Monitor configured",
			"name", monitor.Name,
			"source", monitor.SourcePath,
			"recursive", monitor.Recursive,
			"rules", len(monitor.Rules),
		)
	}

	// Inicializar watchers
	watchers := make([]*watcher.FileWatcher, 0, len(cfg.Monitors))
	for _, monitor := range cfg.Monitors {
		w, err := watcher.NewFileWatcher(&monitor, delay, logger)
		if err != nil {
			logger.Error("Failed to create watcher", "monitor", monitor.Name, "error", err)
			continue
		}
		watchers = append(watchers, w)

		if err := w.Start(); err != nil {
			logger.Error("Failed to start watcher", "monitor", monitor.Name, "error", err)
			continue
		}

		logger.Info("Watcher started successfully",
			"monitor", monitor.Name,
			"path", monitor.SourcePath,
			"recursive", monitor.Recursive,
		)
	}

	// Verificar se pelo menos um watcher foi iniciado
	if len(watchers) == 0 {
		log.Fatalf("No watchers could be started")
	}

	// Graceful shutdown (interceptar Ctrl+C e SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("Daemon is running. Press Ctrl+C to stop.")

	// Aguardar sinal de shutdown
	sig := <-sigChan
	logger.Info("Shutting down gracefully...", "signal", sig.String())

	// Parar todos os watchers
	for _, w := range watchers {
		w.Stop()
	}

	logger.Info("Daemon stopped")
}
