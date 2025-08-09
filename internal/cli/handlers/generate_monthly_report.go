package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	_ "github.com/mattn/go-sqlite3"
)

func GenerateMonthReportCLI(db *sql.DB, reader *bufio.Reader) {
	yearInput, err := utils.PromptInput(reader, "Enter year (e.g., 2025): ")
	if err != nil {
		utils.PrintError("reading year", err)
		return
	}

	monthInput, err := utils.PromptInput(reader, "Enter month (e.g., 06): ")
	if err != nil {
		utils.PrintError("reading month", err)
		return
	}

	datePrefix := fmt.Sprintf("%s-%s", yearInput, monthInput)

	rows, err := db.Query(`
		SELECT t.id, t.transaction_date, t.amount, t.description, c.name
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.transaction_date LIKE ? || '%'
		ORDER BY t.transaction_date ASC
	`, datePrefix)
	if err != nil {
		utils.PrintError("retrieving transactions", err)
		return
	}
	defer rows.Close()

	type entry struct {
		id          string
		date        string
		amount      float64
		description string
		category    string
	}

	var entries []entry
	totals := make(map[string]float64)
	var totalIncome, totalSpend float64

	for rows.Next() {
		var e entry
		err := rows.Scan(&e.id, &e.date, &e.amount, &e.description, &e.category)
		if err != nil {
			utils.PrintError("reading row", err)
			return
		}
		entries = append(entries, e)
		totals[e.category] += e.amount
		// Track income vs spending
		if e.amount > 0 {
			totalIncome += e.amount
		} else {
			totalSpend += e.amount
		}
	}

	fmt.Println("\nTransactions:")
	fmt.Printf("%-10s | %-20s | %-30s | %-8s | %-8s\n", "Date", "Category", "Description", "Txn ID", "Amount")
	fmt.Println(strings.Repeat("-", 90))

	for _, e := range entries {
		parsedDate, err := time.Parse(time.RFC3339, e.date)
		if err != nil {
			utils.PrintError("parsing date", err)
			return
		}
		formattedDate := parsedDate.Format("01-02-06")
		amountStr := utils.FormatAmount(e.amount)

		// Color category blue if it's "Uncategorized"
		categoryStr := e.category
		if strings.ToLower(e.category) == "uncategorized" {
			categoryStr = fmt.Sprintf("%s%s%s", utils.Blue, e.category, utils.Reset)
		}

		// Use FormatRow to handle ANSI color codes properly
		columns := []string{
			formattedDate,
			categoryStr,
			utils.Truncate(e.description, 30),
			e.id[:8],
			amountStr,
		}
		widths := []int{10, 20, 30, 8, 8}
		fmt.Println(utils.FormatRow(columns, widths))
	}

	// Month Summary
	netGainLoss := totalIncome + totalSpend // totalSpend is negative, so this gives us the net
	fmt.Println("\nMonth Summary:")
	fmt.Printf("Total Income    : %s\n", utils.FormatAmount(totalIncome))
	fmt.Printf("Total Spend     : %s\n", utils.FormatAmount(totalSpend))
	fmt.Printf("Net Gain/Loss   : %s\n", utils.FormatAmount(netGainLoss))

	fmt.Println("\nCategory Summary:")
	for cat, amt := range totals {
		fmt.Printf("%-15s : %-8s\n", cat, utils.FormatAmount(amt))
	}
}
