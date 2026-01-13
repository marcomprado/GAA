package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config representa a configuração completa do daemon
type Config struct {
	Settings Settings  `yaml:"settings"`
	Monitors []Monitor `yaml:"monitors"`
}

// Settings contém configurações globais do serviço
type Settings struct {
	LogLevel        string `yaml:"log_level"`
	DelayBeforeMove string `yaml:"delay_before_move"` // Ex: "2s", "500ms"
	MaxWorkers      int    `yaml:"max_workers"`
}

// Monitor representa uma pasta a ser monitorada
type Monitor struct {
	Name       string `yaml:"name"`
	SourcePath string `yaml:"source_path"`
	Recursive  bool   `yaml:"recursive"`
	Rules      []Rule `yaml:"rules"`
}

// Rule representa uma regra de organização de arquivos
type Rule struct {
	Name             string   `yaml:"name"`
	Extensions       []string `yaml:"extensions,omitempty"`          // Opcional: lista de extensões (ex: [".pdf", ".docx"])
	NameContains     []string `yaml:"name_contains,omitempty"`       // Opcional: arquivo deve conter uma dessas strings no nome
	NameStartsWith   []string `yaml:"name_starts_with,omitempty"`    // Opcional: arquivo deve começar com uma dessas strings
	Destination      string   `yaml:"destination"`
	ConflictStrategy string   `yaml:"conflict_strategy"` // "rename", "overwrite"
}

// LoadConfig carrega e parseia o arquivo de configuração YAML
func LoadConfig(path string) (*Config, error) {
	// Abrir arquivo
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Parsear YAML
	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Validate verifica se a configuração é válida
func (c *Config) Validate() error {
	// Validar log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Settings.LogLevel] {
		return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", c.Settings.LogLevel)
	}

	// Validar delay_before_move
	if _, err := c.ParseDelayDuration(); err != nil {
		return fmt.Errorf("invalid delay_before_move: %w", err)
	}

	// Validar max_workers
	if c.Settings.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be greater than 0, got: %d", c.Settings.MaxWorkers)
	}

	// Validar cada monitor
	if len(c.Monitors) == 0 {
		return fmt.Errorf("no monitors configured")
	}

	for i, monitor := range c.Monitors {
		if monitor.Name == "" {
			return fmt.Errorf("monitor #%d has no name", i+1)
		}

		// Verificar se source_path existe
		if _, err := os.Stat(monitor.SourcePath); os.IsNotExist(err) {
			return fmt.Errorf("monitor '%s': source_path does not exist: %s", monitor.Name, monitor.SourcePath)
		}

		// Validar regras
		if len(monitor.Rules) == 0 {
			return fmt.Errorf("monitor '%s' has no rules", monitor.Name)
		}

		for j, rule := range monitor.Rules {
			if rule.Name == "" {
				return fmt.Errorf("monitor '%s', rule #%d has no name", monitor.Name, j+1)
			}

			// Pelo menos um critério de matching deve estar definido
			if len(rule.Extensions) == 0 && len(rule.NameContains) == 0 && len(rule.NameStartsWith) == 0 {
				return fmt.Errorf("monitor '%s', rule '%s': must define at least one matching criterion (extensions, name_contains, or name_starts_with)", monitor.Name, rule.Name)
			}

			if rule.Destination == "" {
				return fmt.Errorf("monitor '%s', rule '%s' has no destination", monitor.Name, rule.Name)
			}

			// Validar conflict_strategy
			validStrategies := map[string]bool{
				"rename":    true,
				"overwrite": true,
				"skip":      true,
			}
			if !validStrategies[rule.ConflictStrategy] {
				return fmt.Errorf("monitor '%s', rule '%s': invalid conflict_strategy: %s (must be rename, overwrite, or skip)",
					monitor.Name, rule.Name, rule.ConflictStrategy)
			}

			// Criar diretório de destino se não existir
			if err := os.MkdirAll(rule.Destination, 0755); err != nil {
				return fmt.Errorf("monitor '%s', rule '%s': failed to create destination directory: %w",
					monitor.Name, rule.Name, err)
			}
		}
	}

	return nil
}

// ParseDelayDuration converte a string delay_before_move em time.Duration
func (c *Config) ParseDelayDuration() (time.Duration, error) {
	duration, err := time.ParseDuration(c.Settings.DelayBeforeMove)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format '%s': %w (example: '2s', '500ms')",
			c.Settings.DelayBeforeMove, err)
	}

	if duration < 0 {
		return 0, fmt.Errorf("delay_before_move cannot be negative: %s", c.Settings.DelayBeforeMove)
	}

	return duration, nil
}
