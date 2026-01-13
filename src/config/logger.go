package config

import (
	"io"
	"log/slog"
	"os"
)

// InitLogger inicializa o logger com o nível especificado
// O logger escreve tanto para stdout quanto para o arquivo logs/organizer.log
func InitLogger(level string) *slog.Logger {
	// Mapear string para slog.Level
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo // Default para info
	}

	// Criar diretório de logs se não existir
	if err := os.MkdirAll("logs", 0755); err != nil {
		slog.Warn("Failed to create logs directory", "error", err)
	}

	// Abrir arquivo de log
	logFile, err := os.OpenFile("logs/organizer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Warn("Failed to open log file, logging only to stdout", "error", err)
		// Se falhar, logar apenas para stdout
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
		return slog.New(handler)
	}

	// Criar MultiWriter para escrever tanto em stdout quanto no arquivo
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Criar handler com o nível apropriado
	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: logLevel,
		// Adicionar timestamp e source info para melhor debugging
		AddSource: false, // Pode ativar se quiser ver arquivo:linha
	})

	return slog.New(handler)
}
