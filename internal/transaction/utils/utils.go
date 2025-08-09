package utils

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math"
	"time"
)

// Comparison methods for floats
// roundToTwoDecimalPlaces rounds a float to two decimal places
func roundToTwoDecimalPlaces(value float64) float64 {
	return math.Round(value*100) / 100
}

// compareFloats compares two floats rounded to two decimal places
func CompareFloats(a, b float64) bool {
	return roundToTwoDecimalPlaces(a) == roundToTwoDecimalPlaces(b)
}

func GenerateTransactionHash(transactionDate time.Time, amount float64, description string, dailySequence int) string {
	// Create a string representation of the transaction with daily sequence
	transactionString := fmt.Sprintf("%s|%.2f|%s|%d",
		transactionDate.Format("2006-01-02"),
		amount,
		description,
		dailySequence)

	// Generate SHA-256 hash
	hash := sha256.Sum256([]byte(transactionString))
	return fmt.Sprintf("%x", hash)
}

// Helper function to get the next daily sequence for identical transactions
func GetNextDailySequence(db *sql.DB, transactionDate time.Time, amount float64, description string) (int, error) {
	// Query to find all existing transactions with the same date, amount, and description
	query := `SELECT COUNT(*) FROM transactions 
	          WHERE DATE(transaction_date) = DATE(?) 
	          AND ABS(amount - ?) < 0.001 
	          AND description = ?`

	var count int
	err := db.QueryRow(query, transactionDate, amount, description).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get daily sequence count: %w", err)
	}

	// Return the next sequence number (count + 1)
	return count + 1, nil
}
