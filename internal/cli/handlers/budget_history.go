package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

type BudgetHistoryEntry struct {
	Month        string
	BudgetAmount float64
	SpentAmount  float64
	PercentSpent float64
}

type BudgetHistorySummary struct {
	TotalMonths      int
	MonthsInBudget   int
	MonthsOverBudget int
	AverageSpent     float64
	AveragePercent   float64
}

func BudgetHistoryCLI(db *sql.DB, reader *bufio.Reader) {
	// Get all budget definitions
	budgetDefinitions, err := utils.GetBudgetDefinitions(db)
	if err != nil {
		utils.PrintError("retrieving budget definitions", err)
		return
	}

	if len(budgetDefinitions) == 0 {
		fmt.Println("No budgets found. Please create a budget first.")
		return
	}

	// Display available budgets
	fmt.Println("Available budgets:")
	for i, budget := range budgetDefinitions {
		budgetName := budget["name"].(string)
		description := budget["description"].(string)

		fmt.Printf("%d. %s", i+1, budgetName)
		if description != "" {
			fmt.Printf(" (%s)", description)
		}
		fmt.Println()
	}

	// Let user select a budget
	budgetChoice, err := utils.PromptInput(reader, "\nSelect budget number: ")
	if err != nil {
		utils.PrintError("reading budget choice", err)
		return
	}

	budgetIndex, err := strconv.Atoi(budgetChoice)
	if err != nil || budgetIndex < 1 || budgetIndex > len(budgetDefinitions) {
		fmt.Println("Invalid budget selection.")
		return
	}

	selectedBudget := budgetDefinitions[budgetIndex-1]
	budgetID := selectedBudget["id"].(int)
	budgetName := selectedBudget["name"].(string)

	// Get budget history
	history, summary, err := getBudgetHistory(db, budgetID)
	if err != nil {
		utils.PrintError("retrieving budget history", err)
		return
	}

	if len(history) == 0 {
		fmt.Printf("\nNo budget history found for '%s'.\n", budgetName)
		return
	}

	// Display budget history
	fmt.Printf("\n=== Budget History: %s ===\n", budgetName)
	fmt.Println()

	// Header
	header := []string{"Month", "Budget", "Spent", "Percent"}
	widths := []int{10, 12, 12, 10}
	fmt.Println(utils.FormatRow(header, widths))
	fmt.Println(strings.Repeat("-", utils.Sum(widths)+9)) // includes padding for " | "

	// Display each month
	for _, entry := range history {
		percentStr := fmt.Sprintf("%.1f%%", entry.PercentSpent)
		if entry.PercentSpent > 100 {
			percentStr = utils.Red + percentStr + utils.Reset
		} else {
			percentStr = utils.Green + percentStr + utils.Reset
		}

		row := []string{
			entry.Month,
			utils.PadAnsi(utils.FormatAmount(entry.BudgetAmount), widths[1]),
			utils.PadAnsi(utils.FormatAmount(entry.SpentAmount), widths[2]),
			utils.PadAnsi(percentStr, widths[3]),
		}
		fmt.Println(utils.FormatRow(row, widths))
	}

	// Display summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Printf("Total months tracked: %d\n", summary.TotalMonths)
	fmt.Printf("Months within budget: %d\n", summary.MonthsInBudget)
	fmt.Printf("Months over budget: %d\n", summary.MonthsOverBudget)
	fmt.Printf("Average monthly spend: %s\n", utils.FormatAmount(summary.AverageSpent))
	fmt.Printf("Average spend percentage: %.1f%%\n", summary.AveragePercent)
}

func getBudgetHistory(db *sql.DB, budgetID int) ([]BudgetHistoryEntry, BudgetHistorySummary, error) {
	// Get all monthly budget instances for this budget
	instances, err := database.GetMonthlyBudgetInstancesByDefinition(db, budgetID)
	if err != nil {
		return nil, BudgetHistorySummary{}, err
	}

	if len(instances) == 0 {
		return []BudgetHistoryEntry{}, BudgetHistorySummary{}, nil
	}

	// Get budget categories
	_, _, categoryIDs, err := database.GetBudgetDefinitionWithCategories(db, budgetID)
	if err != nil {
		return nil, BudgetHistorySummary{}, err
	}

	if len(categoryIDs) == 0 {
		return []BudgetHistoryEntry{}, BudgetHistorySummary{}, nil
	}

	var history []BudgetHistoryEntry
	var totalSpent float64
	var monthsInBudget, monthsOverBudget int

	// Process each monthly instance
	for _, instance := range instances {
		budgetMonth := instance["budget_month"].(string)
		budgetAmount := instance["budget_amount"].(float64)

		// Calculate spent amount for this month
		spentAmount, err := utils.CalculateBudgetSpending(db, categoryIDs, budgetMonth)
		if err != nil {
			return nil, BudgetHistorySummary{}, err
		}

		// Calculate percentage (spentAmount is negative, so we use absolute value)
		percentSpent := (-spentAmount / budgetAmount) * 100

		// Track summary stats
		totalSpent += spentAmount
		if percentSpent <= 100 {
			monthsInBudget++
		} else {
			monthsOverBudget++
		}

		history = append(history, BudgetHistoryEntry{
			Month:        budgetMonth,
			BudgetAmount: budgetAmount,
			SpentAmount:  spentAmount,
			PercentSpent: percentSpent,
		})
	}

	// Sort history by month (chronological order)
	sort.Slice(history, func(i, j int) bool {
		return history[i].Month < history[j].Month
	})

	// Calculate summary
	totalMonths := len(history)
	var averageSpent, averagePercent float64
	if totalMonths > 0 {
		averageSpent = totalSpent / float64(totalMonths)

		// Calculate average percentage
		totalPercent := 0.0
		for _, entry := range history {
			totalPercent += entry.PercentSpent
		}
		averagePercent = totalPercent / float64(totalMonths)
	}

	summary := BudgetHistorySummary{
		TotalMonths:      totalMonths,
		MonthsInBudget:   monthsInBudget,
		MonthsOverBudget: monthsOverBudget,
		AverageSpent:     averageSpent,
		AveragePercent:   averagePercent,
	}

	return history, summary, nil
}
