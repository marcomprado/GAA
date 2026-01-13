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
	ext := filepath.Ext(filename)

	// Normalizar extensão para lowercase
	ext = strings.ToLower(ext)

	// Se não tem extensão, retornar nil
	if ext == "" {
		return nil
	}

	// Iterar sobre as regras
	for i := range rules {
		rule := &rules[i]

		// Verificar se a extensão está na lista de extensões da regra
		for _, ruleExt := range rule.Extensions {
			// Normalizar extensão da regra também
			normalizedRuleExt := strings.ToLower(ruleExt)

			if ext == normalizedRuleExt {
				return rule
			}
		}
	}

	// Nenhuma regra correspondeu
	return nil
}
