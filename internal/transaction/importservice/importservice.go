package importservice

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	cliUtils "github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	"github.com/HadeZForge/FortiFi/internal/dataparse"
	"github.com/HadeZForge/FortiFi/internal/transaction/utils"
	"github.com/HadeZForge/FortiFi/internal/types"
)

// ImportStats holds statistics about the import process
type ImportStats struct {
	TotalRead    int
	TotalSkipped int
	TotalAdded   int
}

// ImportCSVFile imports a CSV file using the generic configuration-based approach
func ImportCSVFile(db *sql.DB, filePath string, configPath string) (*ImportStats, error) {
	// Initialize statistics
	stats := &ImportStats{}

	// Load configuration
	config, err := utils.LoadImportConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load import config: %w", err)
	}

	// Find appropriate format based on filename
	filename := filepath.Base(filePath)
	format, err := utils.FindImportFormat(config, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to find import format: %w", err)
	}

	// Get keyword mappings for categorization
	exactKeywords, err := database.GetExactKeywords(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get exact keywords: %w", err)
	}

	includesKeywords, err := database.GetIncludesKeywords(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get includes keywords: %w", err)
	}

	// Get account ID
	accountID, err := database.GetAccountID(db, format.AccountName)
	if err != nil {
		return nil, fmt.Errorf("failed to get account ID: %w", err)
	}

	// Parse CSV file
	transactions, err := dataparse.ParseGenericCSV(filePath, format)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	fmt.Printf("Parsed %d transactions from file: %s\n", len(transactions), filePath)

	// Set total read count
	stats.TotalRead = len(transactions)

	// Process each transaction
	for _, transaction := range transactions {
		// Handle balance tracking if configured
		if format.TrackBalance && transaction.Balance != nil {
			_, err = database.InsertAccountSnapshot(db, accountID, transaction.Date, *transaction.Balance)
			if err != nil {
				cliUtils.PrintError("inserting account snapshot", err)
				continue
			}
		}

		// Determine category
		categoryID, err := determineCategory(db, transaction, format, exactKeywords, includesKeywords)
		if err != nil {
			cliUtils.PrintError("categorizing transaction", err)
			stats.TotalSkipped++
			continue
		}

		// Generate transaction ID with daily sequence
		transactionID := utils.GenerateTransactionHash(
			transaction.Date,
			transaction.Amount,
			transaction.Description,
			transaction.DailySequence)

		// Check if transaction already exists
		exists, err := database.TransactionExists(db, transactionID)
		if err != nil {
			cliUtils.PrintError("checking transaction existence", err)
			stats.TotalSkipped++
			continue
		}
		if exists {
			fmt.Printf("Transaction already exists, skipping: %s\n", transactionID[:8])
			stats.TotalSkipped++
			continue
		}

		// Insert transaction
		query := `INSERT INTO transactions (id, account_id, category_id, amount, transaction_date, description) 
		          VALUES (?, ?, ?, ?, ?, ?)`

		_, err = db.Exec(query, transactionID, accountID, categoryID, transaction.Amount, transaction.Date, transaction.Description)
		if err != nil {
			cliUtils.PrintError("inserting transaction", err)
			fmt.Printf("Skipping transaction with amount: %.2f and description: %s\n", transaction.Amount, transaction.Description)
			stats.TotalSkipped++
			continue
		}

		// Transaction successfully added
		stats.TotalAdded++
	}

	fmt.Println("Completed importing transactions")
	return stats, nil
}

// Helpers
func categorizeTransaction(db *sql.DB, description string, defaultCategory string, exactKeywords map[string]int, includesKeywords map[string]int) (int, error) {
	var categoryID int
	var err error

	// First check exact keywords (highest priority)
	if id, exists := exactKeywords[description]; exists {
		return id, nil
	}

	// Then check includes keywords
	for keyword, id := range includesKeywords {
		if strings.Contains(description, keyword) {
			return id, nil
		}
	}

	// If no keyword match found, use default category or create new one
	categoryID, err = database.GetCategoryID(db, defaultCategory)
	if err != nil {
		categoryID, err = database.InsertCategory(db, defaultCategory)
		if err != nil {
			return 0, fmt.Errorf("failed to get/create category ID: %w", err)
		}
	}

	return categoryID, nil
}

// determineCategory determines the category for a transaction based on rules and keywords
func determineCategory(db *sql.DB, transaction types.GenericTransaction, format *types.ImportFormat, exactKeywords map[string]int, includesKeywords map[string]int) (int, error) {
	// Check special rules first
	for _, rule := range format.SpecialRules {
		if transaction.Description == rule.DescriptionExact {
			// Check amount if specified
			if rule.AmountExact != nil && !utils.CompareFloats(transaction.Amount, *rule.AmountExact) {
				continue
			}

			// This rule matches, use the forced category
			return database.GetCategoryID(db, rule.ForceCategory)
		}
	}

	// Use shared categorization logic with "Uncategorized" as default
	return categorizeTransaction(db, transaction.Description, "Uncategorized", exactKeywords, includesKeywords)
}
