package dataparse

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"slices"

	cliUtils "github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/transaction/utils"
	"github.com/HadeZForge/FortiFi/internal/types"
)

// ParseGenericCSV parses a CSV file using the provided import format configuration
func ParseGenericCSV(filePath string, format *types.ImportFormat) ([]types.GenericTransaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Get headers and find column indices
	headers := records[0]

	startRow := 1

	if len(records) <= startRow {
		return nil, fmt.Errorf("CSV file has no data rows")
	}

	// Find column indices
	dateIndex, err := utils.GetColumnIndex(headers, format.ColumnMapping.Date)
	if err != nil {
		return nil, fmt.Errorf("date column error: %w", err)
	}

	descIndex, err := utils.GetColumnIndex(headers, format.ColumnMapping.Description)
	if err != nil {
		return nil, fmt.Errorf("description column error: %w", err)
	}

	amountIndex, err := utils.GetColumnIndex(headers, format.ColumnMapping.Amount)
	if err != nil {
		return nil, fmt.Errorf("amount column error: %w", err)
	}

	var balanceIndex int = -1
	if format.ColumnMapping.Balance != "" {
		balanceIndex, err = utils.GetColumnIndex(headers, format.ColumnMapping.Balance)
		if err != nil {
			return nil, fmt.Errorf("balance column error: %w", err)
		}
	}

	var rawTransactions []types.GenericTransaction

	// Parse each row
	for i, row := range records[startRow:] {
		rowNum := i + startRow + 1

		if len(row) <= maxIndex(dateIndex, descIndex, amountIndex, balanceIndex) {
			fmt.Printf("Skipping row %d: not enough fields\n", rowNum)
			continue
		}

		// Parse date
		date, err := time.Parse(format.DateFormat, strings.TrimSpace(row[dateIndex]))
		if err != nil {
			fmt.Printf("Skipping row %d: invalid date format: %v\n", rowNum, err)
			continue
		}

		// Parse amount
		amountStr := strings.TrimSpace(row[amountIndex])
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			fmt.Printf("Skipping row %d: invalid amount: %v\n", rowNum, err)
			continue
		}

		// Apply amount multiplier
		amount *= format.AmountMultiplier

		// Parse balance if configured
		var balance *float64
		if balanceIndex >= 0 {
			balanceStr := strings.TrimSpace(row[balanceIndex])
			if balanceStr != "" {
				bal, err := strconv.ParseFloat(balanceStr, 64)
				if err != nil {
					cliUtils.PrintWarning("parsing balance", err)
				} else {
					balance = &bal
				}
			}
		}

		description := strings.TrimSpace(row[descIndex])

		// Apply blacklist filters
		if isBlacklisted(description, amount, format) {
			fmt.Printf("Skipping blacklisted transaction: %s\n", description)
			continue
		}

		rawTransactions = append(rawTransactions, types.GenericTransaction{
			Date:        date,
			Description: description,
			Amount:      amount,
			Balance:     balance,
		})
	}

	// Process transactions to assign daily sequences
	return ProcessGenericTransactionsWithDailySequence(rawTransactions), nil
}

// isBlacklisted checks if a transaction should be filtered out
func isBlacklisted(description string, amount float64, format *types.ImportFormat) bool {
	// Check exact blacklist
	if slices.Contains(format.BlacklistExact, description) {
		return true
	}

	// Check contains blacklist
	for _, blacklisted := range format.BlacklistContains {
		if strings.Contains(description, blacklisted) {
			return true
		}
	}

	return false
}

// ProcessGenericTransactionsWithDailySequence assigns daily sequences to transactions
func ProcessGenericTransactionsWithDailySequence(transactions []types.GenericTransaction) []types.GenericTransaction {
	// Group transactions by day and identical content
	dailyGroups := make(map[string][]types.GenericTransaction)

	for _, tx := range transactions {
		// Create a key for grouping: date + amount + description
		key := fmt.Sprintf("%s|%.2f|%s",
			tx.Date.Format("2006-01-02"),
			tx.Amount,
			tx.Description)

		dailyGroups[key] = append(dailyGroups[key], tx)
	}

	// Assign sequences to each group
	var processedTransactions []types.GenericTransaction

	for _, group := range dailyGroups {
		for i, tx := range group {
			tx.DailySequence = i + 1 // Start from 1
			processedTransactions = append(processedTransactions, tx)
		}
	}

	return processedTransactions
}

// maxIndex returns the maximum of the provided indices, ignoring -1 values
func maxIndex(indices ...int) int {
	max := -1
	for _, idx := range indices {
		if idx > max {
			max = idx
		}
	}
	return max
}
