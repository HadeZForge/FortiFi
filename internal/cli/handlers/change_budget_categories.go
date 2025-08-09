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

func ChangeBudgetCategoriesCLI(db *sql.DB, reader *bufio.Reader) {
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

	// Loop until user is finished
	for {
		// Refresh current categories after each operation
		_, budgetDescription, currentCategoryIDs, err := database.GetBudgetDefinitionWithCategories(db, budgetID)
		if err != nil {
			utils.PrintError("retrieving budget categories", err)
			return
		}

		// Get updated category names for display
		var currentCategoryNames []string
		for _, categoryID := range currentCategoryIDs {
			categoryName, err := database.GetCategoryNameByID(db, categoryID)
			if err != nil {
				utils.PrintError("retrieving category name", err)
				return
			}
			currentCategoryNames = append(currentCategoryNames, categoryName)
		}

		// Display current budget info
		fmt.Printf("\n=== Budget: %s ===\n", budgetName)
		if budgetDescription != "" {
			fmt.Printf("Description: %s\n", budgetDescription)
		}
		fmt.Printf("Current categories: %s\n", strings.Join(currentCategoryNames, ", "))

		// Check if there's only one category
		if len(currentCategoryIDs) == 1 {
			fmt.Println("\nNote: This budget has only one category. You can only add categories, not remove the existing one.")
		}

		// Ask user what they want to do
		fmt.Println("\nWhat would you like to do?")
		fmt.Println("1. Add a category")
		if len(currentCategoryIDs) > 1 {
			fmt.Println("2. Remove a category")
			fmt.Println("3. Finish (done with changes)")
		} else {
			fmt.Println("2. Finish (done with changes)")
		}

		actionChoice, err := utils.PromptInput(reader, "Enter your choice: ")
		if err != nil {
			utils.PrintError("reading action choice", err)
			return
		}

		switch actionChoice {
		case "1":
			// Add a category
			addCategoryToBudget(db, reader, budgetID, budgetName, currentCategoryIDs, currentCategoryNames)
		case "2":
			if len(currentCategoryIDs) > 1 {
				// Remove a category
				removeCategoryFromBudget(db, reader, budgetID, budgetName, currentCategoryIDs, currentCategoryNames)
			} else {
				// Finish
				fmt.Printf("\nBudget '%s' categories finalized with: %s\n", budgetName, strings.Join(currentCategoryNames, ", "))
				return
			}
		case "3":
			if len(currentCategoryIDs) > 1 {
				// Finish
				fmt.Printf("\nBudget '%s' categories finalized with: %s\n", budgetName, strings.Join(currentCategoryNames, ", "))
				return
			} else {
				fmt.Println("Invalid choice. Please try again.")
			}
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}

func addCategoryToBudget(db *sql.DB, reader *bufio.Reader, budgetID int, budgetName string, currentCategoryIDs []int, currentCategoryNames []string) {
	// Get all available categories
	allCategories, err := utils.GetAvailableCategories(db)
	if err != nil {
		utils.PrintError("retrieving categories", err)
		return
	}

	// Filter out categories that are already associated with this budget
	var availableCategories []types.CategoryInfo
	for _, category := range allCategories {
		found := false
		for _, currentID := range currentCategoryIDs {
			if category.Id == currentID {
				found = true
				break
			}
		}
		if !found {
			availableCategories = append(availableCategories, category)
		}
	}

	if len(availableCategories) == 0 {
		fmt.Println("All categories are already associated with this budget.")
		return
	}

	fmt.Printf("\nAvailable categories to add:\n")

	// Let user select a category to add
	categoryID, categoryName, err := utils.SelectCategory(db, reader, availableCategories, false)
	if err != nil {
		utils.PrintError("selecting category", err)
		return
	}

	// Confirm the addition
	fmt.Printf("\nAdd category '%s' to budget '%s'?\n", categoryName, budgetName)
	confirmInput, err := utils.PromptInput(reader, "Are you sure? (yes/no): ")
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}

	if strings.ToLower(confirmInput) != "yes" && strings.ToLower(confirmInput) != "y" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Add the category to the budget
	_, err = db.Exec(`INSERT INTO budget_definition_categories (budget_definition_id, category_id) VALUES (?, ?)`, budgetID, categoryID)
	if err != nil {
		utils.PrintError("adding category to budget", err)
		return
	}

	fmt.Printf("Successfully added category '%s' to budget '%s'.\n", categoryName, budgetName)
}

func removeCategoryFromBudget(db *sql.DB, reader *bufio.Reader, budgetID int, budgetName string, currentCategoryIDs []int, currentCategoryNames []string) {
	// Display current categories with numbers
	fmt.Printf("\nCurrent categories in budget '%s':\n", budgetName)
	for i, categoryName := range currentCategoryNames {
		fmt.Printf("%d. %s\n", i+1, categoryName)
	}

	// Let user select a category to remove
	categoryChoice, err := utils.PromptInput(reader, "\nSelect category number to remove: ")
	if err != nil {
		utils.PrintError("reading category choice", err)
		return
	}

	categoryIndex, err := strconv.Atoi(categoryChoice)
	if err != nil || categoryIndex < 1 || categoryIndex > len(currentCategoryNames) {
		fmt.Println("Invalid category selection.")
		return
	}

	selectedCategoryID := currentCategoryIDs[categoryIndex-1]
	selectedCategoryName := currentCategoryNames[categoryIndex-1]

	// Confirm the removal
	fmt.Printf("\nRemove category '%s' from budget '%s'?\n", selectedCategoryName, budgetName)
	confirmInput, err := utils.PromptInput(reader, "Are you sure? (yes/no): ")
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}

	if strings.ToLower(confirmInput) != "yes" && strings.ToLower(confirmInput) != "y" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Remove the category from the budget
	_, err = db.Exec(`DELETE FROM budget_definition_categories WHERE budget_definition_id = ? AND category_id = ?`, budgetID, selectedCategoryID)
	if err != nil {
		utils.PrintError("removing category from budget", err)
		return
	}

	fmt.Printf("Successfully removed category '%s' from budget '%s'.\n", selectedCategoryName, budgetName)
}
