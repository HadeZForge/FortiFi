package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
)

func BudgetReportCLI(db *sql.DB, reader *bufio.Reader) {
	// Get current month
	currentMonth := utils.GetCurrentMonth()

	// Get all budget definitions
	budgetDefinitions, err := utils.GetBudgetDefinitions(db)
	if err != nil {
		utils.PrintError("retrieving budget definitions", err)
		return
	}

	if len(budgetDefinitions) == 0 {
		fmt.Println("No budgets found.")
		return
	}

	fmt.Printf("\nBudget Report for %s:\n", currentMonth)
	fmt.Println("=" + strings.Repeat("=", 60))

	for _, budget := range budgetDefinitions {
		budgetID := budget["id"].(int)
		budgetName := budget["name"].(string)

		// Get budget amount for current month
		budgetAmount, err := database.GetMonthlyBudgetInstance(db, budgetID, currentMonth)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("%s: No budget set for %s\n", budgetName, currentMonth)
				continue
			}
			utils.PrintError(fmt.Sprintf("getting budget amount for %s", budgetName), err)
			continue
		}

		// Get categories for this budget
		_, _, categoryIDs, err := database.GetBudgetDefinitionWithCategories(db, budgetID)
		if err != nil {
			utils.PrintError(fmt.Sprintf("getting categories for budget %s", budgetName), err)
			continue
		}

		// Get category names
		var categoryNames []string
		for _, categoryID := range categoryIDs {
			categoryName, err := database.GetCategoryNameByID(db, categoryID)
			if err != nil {
				utils.PrintError(fmt.Sprintf("getting category name for ID %d", categoryID), err)
				continue
			}
			categoryNames = append(categoryNames, categoryName)
		}

		if len(categoryNames) == 0 {
			fmt.Printf("%s: No categories assigned (Budget: %s)\n", budgetName, utils.FormatAmount(budgetAmount))
			continue
		}

		// Calculate total spent in budget categories for current month
		spentAmount, err := utils.CalculateBudgetSpending(db, categoryIDs, currentMonth)
		if err != nil {
			utils.PrintError(fmt.Sprintf("calculating spending for budget %s", budgetName), err)
			continue
		}

		// Calculate remaining budget (spentAmount is negative, so we add it)
		remaining := budgetAmount + spentAmount

		// Calculate percentage and determine colors
		absSpent := -spentAmount // Convert negative to positive
		percent := 0.0
		if budgetAmount > 0 {
			percent = (absSpent / budgetAmount) * 100
		}

		var colorCode, percentColor string
		if absSpent <= budgetAmount {
			colorCode = "\033[32m" // Green
			percentColor = "\033[32m"
		} else {
			colorCode = "\033[31m" // Red
			percentColor = "\033[31m"
		}

		// Display budget information
		fmt.Printf("%s%s\033[0m\n", colorCode, budgetName)
		fmt.Printf("  Categories: %s\n", strings.Join(categoryNames, ", "))
		fmt.Printf("  Spent: %s / %s  (%s%.0f%%%s)\n", utils.FormatAmount(spentAmount), utils.FormatAmount(budgetAmount), percentColor, percent, "\033[0m")
		fmt.Printf("  Remaining: %s\n", utils.FormatAmount(remaining))
		fmt.Println()
	}
}
