package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

func CategoryTimelineCLI(db *sql.DB, reader *bufio.Reader) {
	// Get all categories with transaction counts
	categories, err := utils.GetAvailableCategories(db)
	if err != nil {
		utils.PrintError("retrieving categories", err)
		return
	}

	if len(categories) == 0 {
		fmt.Println("No categories found with transactions.")
		return
	}

	// Use the new util code to let the user select a category
	_, categoryName, err := utils.SelectCategory(db, reader, categories, false)
	if err != nil {
		utils.PrintError("selecting category", err)
		return
	}

	// Get category timeline data
	timeline, avgMonthlySpend, err := getCategoryTimeline(db, categoryName)
	if err != nil {
		utils.PrintError("retrieving category timeline", err)
		return
	}

	if len(timeline) == 0 {
		fmt.Printf("No transaction data found for category '%s'.\n", categoryName)
		return
	}

	// Display results
	fmt.Printf("\n=== Category Timeline: %s ===\n", categoryName)
	fmt.Printf("Average Monthly Spend: %s\n\n", utils.FormatAmount(avgMonthlySpend))

	header := []string{"Month", "Total", "vs Previous", "vs Average"}
	widths := []int{10, 12, 15, 15}

	fmt.Println(utils.FormatRow(header, widths))
	fmt.Println(strings.Repeat("-", utils.Sum(widths)+9)) // includes padding for " | "
	for i, entry := range timeline {
		var prevDeltaStr, avgDeltaStr string

		// Previous delta
		if i == 0 {
			prevDeltaStr = utils.PadAnsi(fmt.Sprintf("%s---%s", utils.Blue, utils.Reset), widths[2])
		} else {
			delta := entry.Total - timeline[i-1].Total
			prevDeltaStr = utils.PadAnsi(utils.FormatDelta(delta), widths[2])
		}

		// Avg delta
		avgDelta := entry.Total - avgMonthlySpend
		avgDeltaStr = utils.PadAnsi(utils.FormatDelta(avgDelta), widths[3])

		row := []string{
			entry.Month,
			utils.PadAnsi(utils.FormatAmount(entry.Total), widths[1]),
			prevDeltaStr,
			avgDeltaStr,
		}
		fmt.Println(utils.FormatRow(row, widths))
	}

	fmt.Printf("\nTotal months: %d\n", len(timeline))
}

func getCategoryTimeline(db *sql.DB, categoryName string) ([]types.TimelineEntry, float64, error) {
	query := `
		SELECT 
			strftime('%Y-%m', t.transaction_date) as month,
			SUM(t.amount) as total
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE c.name = ?
		GROUP BY strftime('%Y-%m', t.transaction_date)
		ORDER BY month ASC
	`

	rows, err := db.Query(query, categoryName)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var timeline []types.TimelineEntry
	var totalSpend float64

	for rows.Next() {
		var entry types.TimelineEntry
		if err := rows.Scan(&entry.Month, &entry.Total); err != nil {
			return nil, 0, err
		}
		timeline = append(timeline, entry)
		totalSpend += entry.Total
	}

	// Calculate average monthly spend
	var avgMonthlySpend float64
	if len(timeline) > 0 {
		avgMonthlySpend = totalSpend / float64(len(timeline))
	}

	return timeline, avgMonthlySpend, nil
}
