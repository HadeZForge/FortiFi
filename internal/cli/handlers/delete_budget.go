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

func DeleteBudgetCLI(db *sql.DB, reader *bufio.Reader) {
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
	selectionInput, err := utils.PromptInput(reader, "\nSelect budget number to delete: ")
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

	// Get monthly instances for this budget
	monthlyInstances, err := database.GetMonthlyBudgetInstancesByDefinition(db, budgetID)
	if err != nil {
		utils.PrintError("retrieving monthly instances", err)
		return
	}

	fmt.Printf("\nBudget '%s' has %d monthly instance(s):\n", budgetName, len(monthlyInstances))

	if len(monthlyInstances) > 0 {
		// Show monthly instances
		for _, instance := range monthlyInstances {
			month := instance["budget_month"].(string)
			amount := instance["budget_amount"].(float64)
			fmt.Printf("  %s: %s\n", month, utils.FormatAmount(amount))
		}

		fmt.Println("\nOptions:")
		fmt.Println("1. Delete budget definition and all monthly instances")
		fmt.Println("2. Delete only monthly instances (keep budget definition)")
		fmt.Println("3. Cancel deletion")

		optionInput, err := utils.PromptInput(reader, "Select option (1, 2, or 3): ")
		if err != nil {
			utils.PrintError("reading option", err)
			return
		}

		switch optionInput {
		case "1":
			// Delete everything
			confirmDeleteEverything(db, reader, budgetID, budgetName, len(monthlyInstances))
		case "2":
			// Delete only monthly instances
			confirmDeleteMonthlyInstances(db, reader, budgetID, budgetName, len(monthlyInstances))
		case "3":
			fmt.Println("Budget deletion cancelled.")
			return
		default:
			fmt.Println("Invalid option. Please select 1, 2, or 3.")
			return
		}
	} else {
		// No monthly instances, just delete the budget definition
		confirmDeleteBudgetDefinition(db, reader, budgetID, budgetName)
	}
}

func confirmDeleteEverything(db *sql.DB, reader *bufio.Reader, budgetID int, budgetName string, instanceCount int) {
	confirmPrompt := fmt.Sprintf("\nAre you sure you want to delete budget '%s' and all %d monthly instance(s)? (yes/no): ",
		budgetName, instanceCount)
	confirmInput, err := utils.PromptInput(reader, confirmPrompt)
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Budget deletion cancelled.")
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

	// Delete monthly instances first (due to foreign key constraint)
	result1, err := tx.Exec(`DELETE FROM monthly_budget_instances WHERE budget_definition_id = ?`, budgetID)
	if err != nil {
		utils.PrintError("deleting monthly instances", err)
		return
	}

	// Delete budget definition
	_, err = tx.Exec(`DELETE FROM budget_definitions WHERE id = ?`, budgetID)
	if err != nil {
		utils.PrintError("deleting budget definition", err)
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		utils.PrintError("committing transaction", err)
		return
	}

	instancesDeleted, _ := result1.RowsAffected()

	fmt.Printf("\nSuccessfully deleted budget '%s' and %d monthly instance(s)\n", budgetName, instancesDeleted)
}

func confirmDeleteMonthlyInstances(db *sql.DB, reader *bufio.Reader, budgetID int, budgetName string, instanceCount int) {
	confirmPrompt := fmt.Sprintf("\nAre you sure you want to delete all %d monthly instance(s) for budget '%s'? (yes/no): ",
		instanceCount, budgetName)
	confirmInput, err := utils.PromptInput(reader, confirmPrompt)
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Monthly instance deletion cancelled.")
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

	// Delete monthly instances
	result, err := tx.Exec(`DELETE FROM monthly_budget_instances WHERE budget_definition_id = ?`, budgetID)
	if err != nil {
		utils.PrintError("deleting monthly instances", err)
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		utils.PrintError("committing transaction", err)
		return
	}

	instancesDeleted, _ := result.RowsAffected()
	fmt.Printf("\nSuccessfully deleted %d monthly instance(s) for budget '%s'\n", instancesDeleted, budgetName)
	fmt.Printf("Budget definition '%s' has been preserved and can be used for future months.\n", budgetName)
}

func confirmDeleteBudgetDefinition(db *sql.DB, reader *bufio.Reader, budgetID int, budgetName string) {
	confirmPrompt := fmt.Sprintf("\nAre you sure you want to delete budget definition '%s'? (yes/no): ", budgetName)
	confirmInput, err := utils.PromptInput(reader, confirmPrompt)
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Budget deletion cancelled.")
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

	// Delete budget definition (cascades to category associations)
	result, err := tx.Exec(`DELETE FROM budget_definitions WHERE id = ?`, budgetID)
	if err != nil {
		utils.PrintError("deleting budget definition", err)
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		utils.PrintError("committing transaction", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("\nSuccessfully deleted budget definition '%s'\n", budgetName)
	} else {
		fmt.Printf("\nBudget definition '%s' was not found or already deleted\n", budgetName)
	}
}
