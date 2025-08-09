package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/transaction/importservice"
)

const DEFAULT_CONFIG_PATH = "./import_config.json"

// IngestDataCLI handles the data ingestion command using the new generic import system
func IngestDataCLI(db *sql.DB, reader *bufio.Reader) {
	fmt.Println("\n=== Data Ingestion ===")
	fmt.Println("Enter file path or 'raw' to process all files in ./raw/ directory:")

	input, err := utils.PromptInput(reader, "")
	if err != nil {
		utils.PrintError("reading input", err)
		return
	}

	if strings.ToLower(input) == "raw" {
		// Process all CSV files in ./raw/ directory
		totalStats, err := processRawDirectory(db)
		if err != nil {
			utils.PrintError("processing raw directory", err)
			return
		}

		// Display total statistics
		if totalStats != nil {
			fmt.Printf("\n=== Total Import Summary ===\n")
			fmt.Printf("Total transactions read: %d\n", totalStats.TotalRead)
			fmt.Printf("Total transactions skipped: %d\n", totalStats.TotalSkipped)
			fmt.Printf("Total transactions added: %d\n", totalStats.TotalAdded)
		}
	} else {
		// Process single file
		stats, err := processSingleFile(db, input)
		if err != nil {
			utils.PrintError("processing file", err)
			return
		}

		// Display statistics
		if stats != nil {
			fmt.Printf("\n=== Import Summary ===\n")
			fmt.Printf("Transactions read: %d\n", stats.TotalRead)
			fmt.Printf("Transactions skipped: %d\n", stats.TotalSkipped)
			fmt.Printf("Transactions added: %d\n", stats.TotalAdded)
		}
	}

	fmt.Println("Data ingestion completed!")
}

// processRawDirectory processes all CSV files in the ./raw/ directory
func processRawDirectory(db *sql.DB) (*importservice.ImportStats, error) {
	rawDir := "./raw/"

	// Check if raw directory exists
	if !utils.FileExists(rawDir) {
		return nil, fmt.Errorf("raw directory does not exist: %s", rawDir)
	}

	// Read all files in the raw directory
	files, err := os.ReadDir(rawDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw directory: %w", err)
	}

	csvFiles := []string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".csv") {
			csvFiles = append(csvFiles, filepath.Join(rawDir, file.Name()))
		}
	}

	if len(csvFiles) == 0 {
		fmt.Println("No CSV files found in ./raw/ directory")
		return &importservice.ImportStats{}, nil
	}

	fmt.Printf("Found %d CSV files to process:\n", len(csvFiles))
	for _, file := range csvFiles {
		fmt.Printf("  - %s\n", file)
	}

	// Initialize total statistics
	totalStats := &importservice.ImportStats{}

	// Process each CSV file using the new generic import system
	for _, filePath := range csvFiles {
		fmt.Printf("\nProcessing: %s\n", filePath)
		stats, err := processSingleFile(db, filePath)
		if err != nil {
			utils.PrintError("processing file", err)
			continue
		}

		// Add to total statistics
		if stats != nil {
			totalStats.TotalRead += stats.TotalRead
			totalStats.TotalSkipped += stats.TotalSkipped
			totalStats.TotalAdded += stats.TotalAdded
		}

		fmt.Printf("Successfully processed: %s\n", filePath)
	}

	return totalStats, nil
}

// processSingleFile processes a single CSV file using the generic import system
func processSingleFile(db *sql.DB, filePath string) (*importservice.ImportStats, error) {
	// Check if file exists
	if !utils.FileExists(filePath) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Check if config file exists
	if !utils.FileExists(DEFAULT_CONFIG_PATH) {
		return nil, fmt.Errorf("import configuration file not found: %s. Please create this file with your import format definitions", DEFAULT_CONFIG_PATH)
	}

	// Use the new generic import system
	fmt.Println("Processing file using configuration-based import...")
	stats, err := importservice.ImportCSVFile(db, filePath, DEFAULT_CONFIG_PATH)
	if err != nil {
		return nil, fmt.Errorf("failed to import file: %w", err)
	}

	return stats, nil
}
