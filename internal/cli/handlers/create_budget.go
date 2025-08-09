package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

func CreateBudgetCLI(db *sql.DB, reader *bufio.Reader) {
	// Prompt for budget name
	budgetName, err := utils.PromptInput(reader, "Enter budget name: ")
	if err != nil {
		utils.PrintError("reading budget name", err)
		return
	}

	if budgetName == "" {
		fmt.Println("Error: Budget name cannot be empty")
		return
	}

	// Prompt for budget description (optional)
	budgetDescription, err := utils.PromptInput(reader, "Enter budget description (optional, press Enter to skip): ")
	if err != nil {
		utils.PrintError("reading budget description", err)
		return
	}

	// Get available categories
	categories, err := utils.GetAvailableCategories(db)
	if err != nil {
		utils.PrintError("retrieving categories", err)
		return
	}

	if len(categories) == 0 {
		fmt.Println("No categories found. Please create some categories first.")
		return
	}

	// Prompt for categories to associate with the budget
	fmt.Printf("\nSelect categories to associate with budget '%s':\n", budgetName)
	var selectedCategoryIDs []int
	var selectedCategoryNames []string

	for {
		// Show remaining categories
		remainingCategories := filterOutSelectedCategories(categories, selectedCategoryIDs)
		if len(remainingCategories) == 0 {
			break
		}

		fmt.Printf("\nRemaining categories:\n")

		// Show selected categories
		if len(selectedCategoryNames) > 0 {
			fmt.Printf("\nSelected categories: %s\n", strings.Join(selectedCategoryNames, ", "))
		}

		// Prompt for category selection
		categoryID, categoryName, err := utils.SelectCategory(db, reader, remainingCategories, false)
		if err != nil {
			if err.Error() == "no categories available" {
				break // No more categories to select
			}
			utils.PrintError("selecting category", err)
			return
		}

		selectedCategoryIDs = append(selectedCategoryIDs, categoryID)
		selectedCategoryNames = append(selectedCategoryNames, categoryName)

		// Ask if user wants to add more categories
		if len(remainingCategories) > 1 {
			moreInput, err := utils.PromptInput(reader, "\nAdd another category? (yes/no): ")
			if err != nil {
				utils.PrintError("reading response", err)
				return
			}
			moreInput = strings.ToLower(moreInput)
			if moreInput != "yes" && moreInput != "y" {
				break
			}
		} else {
			break // No more categories available
		}
	}

	if len(selectedCategoryIDs) == 0 {
		fmt.Println("Error: At least one category must be selected")
		return
	}

	// Prompt for budget amount
	budgetAmountInput, err := utils.PromptInput(reader, fmt.Sprintf("\nEnter budget amount for %s: ", utils.GetCurrentMonth()))
	if err != nil {
		utils.PrintError("reading budget amount", err)
		return
	}

	budgetAmount, err := strconv.ParseFloat(budgetAmountInput, 64)
	if err != nil {
		fmt.Println("Error: Please enter a valid number")
		return
	}

	if budgetAmount <= 0 {
		fmt.Println("Error: Budget amount must be greater than 0")
		return
	}

	// Show confirmation
	fmt.Printf("\nBudget Creation Summary:\n")
	fmt.Printf("Name: %s\n", budgetName)
	if budgetDescription != "" {
		fmt.Printf("Description: %s\n", budgetDescription)
	}
	fmt.Printf("Categories: %s\n", strings.Join(selectedCategoryNames, ", "))
	fmt.Printf("Amount for %s: %s\n", utils.GetCurrentMonth(), utils.FormatAmount(budgetAmount))

	// Confirm creation
	confirmInput, err := utils.PromptInput(reader, "\nAre you sure you want to create this budget? (yes/no): ")
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Budget creation cancelled.")
		return
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		utils.PrintError("beginning database transaction", err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Create budget definition
	budgetDefinitionID, err := database.InsertBudgetDefinition(db, budgetName, budgetDescription, selectedCategoryIDs)
	if err != nil {
		utils.PrintError("creating budget definition", err)
		return
	}

	// Create monthly budget instance for current month
	currentMonth := utils.GetCurrentMonth()
	err = database.InsertMonthlyBudgetInstance(db, budgetDefinitionID, currentMonth, budgetAmount)
	if err != nil {
		utils.PrintError("creating monthly budget instance", err)
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		utils.PrintError("committing transaction", err)
		return
	}

	fmt.Printf("\nBudget created successfully!\n")
	fmt.Printf("  Budget: %s\n", budgetName)
	fmt.Printf("  Categories: %s\n", strings.Join(selectedCategoryNames, ", "))
	fmt.Printf("  Amount for %s: %s\n", currentMonth, utils.FormatAmount(budgetAmount))
	fmt.Printf("  Budget ID: %d\n", budgetDefinitionID)
}

// Helper function to filter out already selected categories
func filterOutSelectedCategories(allCategories []types.CategoryInfo, selectedIDs []int) []types.CategoryInfo {
	var remaining []types.CategoryInfo
	for _, category := range allCategories {
		found := false
		for _, selectedID := range selectedIDs {
			if category.Id == selectedID {
				found = true
				break
			}
		}
		if !found {
			remaining = append(remaining, category)
		}
	}
	return remaining
}
