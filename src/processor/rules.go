package processor

import (
	"path/filepath"
	"strings"

	"gaa/file-organizer/src/config"
)

// MatchRule encontra a primeira regra que corresponde ao arquivo
// Retorna nil se nenhuma regra corresponder
func MatchRule(filePath string, rules []config.Rule) *config.Rule {
	// Extrair o nome do arquivo e extensão
	filename := filepath.Base(filePath)
	ext := strings.ToLower(filepath.Ext(filename))

	// Nome sem extensão (para matching de nome)
	nameWithoutExt := strings.ToLower(strings.TrimSuffix(filename, ext))

	// Iterar sobre as regras (primeira que corresponder é retornada)
	for i := range rules {
		rule := &rules[i]

		// Verificar extensão (se definida)
		// Se Extensions está vazio, considera match automático
		// Se Extensions não está vazio, verifica se a extensão do arquivo está na lista
		extensionMatch := len(rule.Extensions) == 0 || matchesExtension(ext, rule.Extensions)
		if !extensionMatch {
			continue // Extensão não corresponde, próxima regra
		}

		// Verificar name_contains (se definido) - OR logic
		// Se NameContains está vazio, considera match automático
		// Se NameContains não está vazio, verifica se o nome contém alguma das strings
		containsMatch := len(rule.NameContains) == 0 || matchesContains(nameWithoutExt, rule.NameContains)
		if !containsMatch {
			continue // Nome não contém nenhuma das strings, próxima regra
		}

		// Verificar name_contains_all (se definido) - AND logic
		// Se NameContainsAll está vazio, considera match automático
		// Se NameContainsAll não está vazio, verifica se o nome contém TODAS as strings
		containsAllMatch := len(rule.NameContainsAll) == 0 || matchesContainsAll(nameWithoutExt, rule.NameContainsAll)
		if !containsAllMatch {
			continue // Nome não contém todas as strings, próxima regra
		}

		// Verificar name_starts_with (se definido)
		// Se NameStartsWith está vazio, considera match automático
		// Se NameStartsWith não está vazio, verifica se o nome começa com alguma das strings
		startsWithMatch := len(rule.NameStartsWith) == 0 || matchesStartsWith(nameWithoutExt, rule.NameStartsWith)
		if !startsWithMatch {
			continue // Nome não começa com nenhuma das strings, próxima regra
		}

		// Todos os critérios definidos passaram - esta regra corresponde!
		return rule
	}

	// Nenhuma regra correspondeu
	return nil
}

// matchesExtension verifica se a extensão do arquivo está na lista de extensões da regra
func matchesExtension(ext string, extensions []string) bool {
	for _, ruleExt := range extensions {
		// Normalizar extensão da regra para lowercase
		normalizedRuleExt := strings.ToLower(ruleExt)

		if ext == normalizedRuleExt {
			return true
		}
	}
	return false
}

// matchesContains verifica se o nome do arquivo contém alguma das strings especificadas
func matchesContains(nameWithoutExt string, patterns []string) bool {
	for _, pattern := range patterns {
		// Normalizar pattern para lowercase (matching case-insensitive)
		normalizedPattern := strings.ToLower(pattern)

		// Verificar se o nome contém o pattern
		if strings.Contains(nameWithoutExt, normalizedPattern) {
			return true
		}
	}
	return false
}

// matchesContainsAll verifica se o nome do arquivo contém TODAS as strings especificadas (AND logic)
func matchesContainsAll(nameWithoutExt string, patterns []string) bool {
	for _, pattern := range patterns {
		// Normalizar pattern para lowercase (matching case-insensitive)
		normalizedPattern := strings.ToLower(pattern)

		// Se qualquer pattern não for encontrado, retorna false
		if !strings.Contains(nameWithoutExt, normalizedPattern) {
			return false
		}
	}
	// Todos os patterns foram encontrados
	return true
}

// matchesStartsWith verifica se o nome do arquivo começa com alguma das strings especificadas
func matchesStartsWith(nameWithoutExt string, prefixes []string) bool {
	for _, prefix := range prefixes {
		// Normalizar prefix para lowercase (matching case-insensitive)
		normalizedPrefix := strings.ToLower(prefix)

		// Verificar se o nome começa com o prefix
		if strings.HasPrefix(nameWithoutExt, normalizedPrefix) {
			return true
		}
	}
	return false
}
