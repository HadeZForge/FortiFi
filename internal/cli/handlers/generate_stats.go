package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	_ "github.com/mattn/go-sqlite3"
)

func GenerateStatsCLI(db *sql.DB, reader *bufio.Reader) {
	input, err := utils.PromptInput(reader, "Enter year (e.g., 2025) or 'all' for all data: ")
	if err != nil {
		utils.PrintError("reading input", err)
		return
	}

	var whereClause string
	var timeDescription string

	if strings.ToLower(input) == "all" {
		whereClause = ""
		timeDescription = "All Time"
	} else {
		whereClause = fmt.Sprintf("WHERE t.transaction_date LIKE '%s%%'", input)
		timeDescription = fmt.Sprintf("Year %s", input)
	}

	// Query for all transactions with category names
	query := fmt.Sprintf(`
		SELECT t.amount, c.name, t.transaction_date
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		%s
		ORDER BY t.transaction_date ASC
	`, whereClause)

	rows, err := db.Query(query)
	if err != nil {
		utils.PrintError("retrieving transactions", err)
		return
	}
	defer rows.Close()

	type transaction struct {
		amount   float64
		category string
		date     string
	}

	var transactions []transaction
	categoryTotals := make(map[string]float64)
	monthlyTotals := make(map[string]float64) // YYYY-MM -> total

	for rows.Next() {
		var t transaction
		err := rows.Scan(&t.amount, &t.category, &t.date)
		if err != nil {
			utils.PrintError("reading row", err)
			return
		}
		transactions = append(transactions, t)
		categoryTotals[t.category] += t.amount

		// Extract year-month for monthly calculations
		if len(t.date) >= 7 {
			yearMonth := t.date[:7] // YYYY-MM
			monthlyTotals[yearMonth] += t.amount
		}
	}

	if len(transactions) == 0 {
		fmt.Printf("No transactions found for %s\n", timeDescription)
		return
	}

	// Calculate stats
	var totalSpend, totalIncome float64
	monthlySpend := make(map[string]float64)
	monthlyIncome := make(map[string]float64)
	monthlyCategoryTotals := make(map[string]map[string]float64) // month -> category -> amount

	for _, t := range transactions {
		if t.amount < 0 {
			totalSpend += -t.amount // Make positive for display
		} else {
			totalIncome += t.amount
		}

		if len(t.date) >= 7 {
			yearMonth := t.date[:7]

			if t.amount < 0 {
				monthlySpend[yearMonth] += -t.amount
			} else {
				monthlyIncome[yearMonth] += t.amount
			}

			if monthlyCategoryTotals[yearMonth] == nil {
				monthlyCategoryTotals[yearMonth] = make(map[string]float64)
			}
			monthlyCategoryTotals[yearMonth][t.category] += t.amount
		}
	}

	// Calculate averages
	numMonths := len(monthlyTotals)
	if numMonths == 0 {
		numMonths = 1 // Avoid division by zero
	}

	avgMonthlySpend := totalSpend / float64(numMonths)
	avgMonthlyIncome := totalIncome / float64(numMonths)

	// Calculate average monthly category breakdown
	avgCategoryTotals := make(map[string]float64)
	for _, monthCategories := range monthlyCategoryTotals {
		for category, amount := range monthCategories {
			avgCategoryTotals[category] += amount
		}
	}
	for category := range avgCategoryTotals {
		avgCategoryTotals[category] /= float64(numMonths)
	}

	// Display results
	fmt.Printf("\n=== Statistics for %s ===\n", timeDescription)
	fmt.Printf("Data covers %d months\n\n", numMonths)

	fmt.Println("MONTHLY AVERAGES:")
	fmt.Printf("Average Monthly Spend:  %s\n", utils.FormatAmount(-avgMonthlySpend))
	fmt.Printf("Average Monthly Income: %s\n\n", utils.FormatAmount(avgMonthlyIncome))

	fmt.Println("TOTALS:")
	fmt.Printf("Total Spend:  %s\n", utils.FormatAmount(-totalSpend))
	fmt.Printf("Total Income: %s\n\n", utils.FormatAmount(totalIncome))

	fmt.Println("AVERAGE MONTHLY BREAKDOWN BY CATEGORY:")
	fmt.Println(strings.Repeat("-", 40))
	for category, avgAmount := range avgCategoryTotals {
		fmt.Printf("%-20s: %s\n", category, utils.FormatAmount(avgAmount))
	}
}
