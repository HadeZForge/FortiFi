package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/types"
)

const DEFAULT_CONFIG_PATH = "./import_config.json"

// LoadImportConfig loads the import configuration from a JSON file
func LoadImportConfig(configPath string) (*types.ImportConfig, error) {
	if configPath == "" {
		configPath = DEFAULT_CONFIG_PATH
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", configPath, err)
	}
	defer file.Close()

	var config types.ImportConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

// FindImportFormat finds the appropriate import format based on filename
func FindImportFormat(config *types.ImportConfig, filename string) (*types.ImportFormat, error) {
	lowerFilename := strings.ToLower(filename)

	for _, format := range config.ImportFormats {
		if strings.Contains(lowerFilename, strings.ToLower(format.Identifier)) {
			return &format, nil
		}
	}

	return nil, fmt.Errorf("no matching import format found for file: %s", filename)
}

// GetColumnIndex finds the index of a column header in the CSV headers
func GetColumnIndex(headers []string, columnName string) (int, error) {
	for i, header := range headers {
		if strings.TrimSpace(header) == columnName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("column '%s' not found in headers", columnName)
}
