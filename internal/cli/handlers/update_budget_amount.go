package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

func UpdateBudgetAmountCLI(db *sql.DB, reader *bufio.Reader) {
	// Get all budget definitions
	budgetDefinitions, err := utils.GetBudgetDefinitions(db)
	if err != nil {
		utils.PrintError("retrieving budget definitions", err)
		return
	}

	if len(budgetDefinitions) == 0 {
		fmt.Println("No budget definitions found.")
		return
	}

	// Display available budget definitions
	fmt.Println("Available budget definitions:")
	for i, budget := range budgetDefinitions {
		fmt.Printf("%d. %s", i+1, budget["name"])
		if description, ok := budget["description"].(string); ok && description != "" {
			fmt.Printf(" - %s", description)
		}
		fmt.Printf("\n")
	}

	// Prompt for budget selection
	selectionInput, err := utils.PromptInput(reader, "\nSelect budget number to update: ")
	if err != nil {
		utils.PrintError("reading selection", err)
		return
	}

	selection, err := strconv.Atoi(selectionInput)
	if err != nil || selection < 1 || selection > len(budgetDefinitions) {
		fmt.Println("Invalid selection")
		return
	}

	selectedBudget := budgetDefinitions[selection-1]
	budgetID := selectedBudget["id"].(int)
	budgetName := selectedBudget["name"].(string)

	// Get current month
	currentMonth := utils.GetCurrentMonth()

	// Check if monthly instance exists for current month
	currentAmount, err := database.GetMonthlyBudgetInstance(db, budgetID, currentMonth)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			// No instance exists for current month, create one
			fmt.Printf("\nNo budget amount set for %s. Creating new monthly instance.\n", currentMonth)
			createNewMonthlyInstance(db, reader, budgetID, budgetName, currentMonth)
			return
		}
		utils.PrintError("checking current budget amount", err)
		return
	}

	// Show current amount and prompt for new amount
	fmt.Printf("\nCurrent budget amount for %s: %s\n", currentMonth, utils.FormatAmount(currentAmount))

	newAmountInput, err := utils.PromptInput(reader, "Enter new budget amount: ")
	if err != nil {
		utils.PrintError("reading new amount", err)
		return
	}

	newAmount, err := strconv.ParseFloat(newAmountInput, 64)
	if err != nil {
		fmt.Println("Error: Please enter a valid number")
		return
	}

	if newAmount <= 0 {
		fmt.Println("Error: Budget amount must be greater than 0")
		return
	}

	// Show confirmation
	fmt.Printf("\nBudget Update Summary:\n")
	fmt.Printf("Budget: %s\n", budgetName)
	fmt.Printf("Month: %s\n", currentMonth)
	fmt.Printf("Current amount: %s\n", utils.FormatAmount(currentAmount))
	fmt.Printf("New amount: %s\n", utils.FormatAmount(newAmount))
	fmt.Printf("Change: %s\n", utils.FormatDelta(newAmount-currentAmount))

	// Confirm update
	confirmInput, err := utils.PromptInput(reader, "\nAre you sure you want to update this budget amount? (yes/no): ")
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Budget update cancelled.")
		return
	}

	// Update the monthly budget instance
	err = database.UpdateMonthlyBudgetInstance(db, budgetID, currentMonth, newAmount)
	if err != nil {
		utils.PrintError("updating budget amount", err)
		return
	}

	fmt.Printf("\nBudget amount updated successfully!\n")
	fmt.Printf("  Budget: %s\n", budgetName)
	fmt.Printf("  Month: %s\n", currentMonth)
	fmt.Printf("  New amount: %s\n", utils.FormatAmount(newAmount))
}

func createNewMonthlyInstance(db *sql.DB, reader *bufio.Reader, budgetID int, budgetName string, currentMonth string) {
	// Prompt for initial amount
	amountInput, err := utils.PromptInput(reader, fmt.Sprintf("Enter budget amount for %s: ", currentMonth))
	if err != nil {
		utils.PrintError("reading budget amount", err)
		return
	}

	amount, err := strconv.ParseFloat(amountInput, 64)
	if err != nil {
		fmt.Println("Error: Please enter a valid number")
		return
	}

	if amount <= 0 {
		fmt.Println("Error: Budget amount must be greater than 0")
		return
	}

	// Show confirmation
	fmt.Printf("\nBudget Creation Summary:\n")
	fmt.Printf("Budget: %s\n", budgetName)
	fmt.Printf("Month: %s\n", currentMonth)
	fmt.Printf("Amount: %s\n", utils.FormatAmount(amount))

	// Confirm creation
	confirmInput, err := utils.PromptInput(reader, "\nAre you sure you want to create this monthly budget instance? (yes/no): ")
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Budget creation cancelled.")
		return
	}

	// Create the monthly budget instance
	err = database.InsertMonthlyBudgetInstance(db, budgetID, currentMonth, amount)
	if err != nil {
		utils.PrintError("creating monthly budget instance", err)
		return
	}

	fmt.Printf("\nMonthly budget instance created successfully!\n")
	fmt.Printf("  Budget: %s\n", budgetName)
	fmt.Printf("  Month: %s\n", currentMonth)
	fmt.Printf("  Amount: %s\n", utils.FormatAmount(amount))
}
