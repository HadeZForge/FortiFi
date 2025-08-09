package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

func DeleteCategoryCLI(db *sql.DB, reader *bufio.Reader) {
	// Get available categories
	categories, err := utils.GetAvailableCategories(db)
	if err != nil {
		utils.PrintError("retrieving categories", err)
		return
	}

	if len(categories) == 0 {
		fmt.Println("No categories found.")
		return
	}

	// Prompt user to select category to delete
	fmt.Println("Select category to delete:")
	selectedCategoryID, selectedCategoryName, err := utils.SelectCategory(db, reader, categories, false)
	if err != nil {
		utils.PrintError("selecting category", err)
		return
	}

	// Get count of transactions in this category
	var transactionCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE category_id = ?`, selectedCategoryID).Scan(&transactionCount)
	if err != nil {
		utils.PrintError("counting transactions", err)
		return
	}

	fmt.Printf("\nCategory '%s' has %d transaction(s).\n", selectedCategoryName, transactionCount)

	if transactionCount > 0 {
		fmt.Println("\nOptions:")
		fmt.Println("1. Delete all transactions in this category")
		fmt.Println("2. Move transactions to 'Uncategorized' category")
		fmt.Println("3. Cancel deletion")

		optionInput, err := utils.PromptInput(reader, "Select option (1, 2, or 3): ")
		if err != nil {
			utils.PrintError("reading option", err)
			return
		}

		switch optionInput {
		case "1":
			// Delete transactions
			handleDeleteTransactions(db, reader, selectedCategoryID, selectedCategoryName, transactionCount)
		case "2":
			// Move to uncategorized
			handleMoveToUncategorized(db, reader, selectedCategoryID, selectedCategoryName, transactionCount)
		case "3":
			fmt.Println("Category deletion cancelled.")
			return
		default:
			fmt.Println("Invalid option. Please select 1, 2, or 3.")
			return
		}
	} else {
		// No transactions, proceed with category deletion
		confirmCategoryDeletion(db, reader, selectedCategoryID, selectedCategoryName, 0)
	}
}

func handleDeleteTransactions(db *sql.DB, reader *bufio.Reader, categoryID int, categoryName string, transactionCount int) {
	// Show preview of transactions to be deleted
	fmt.Printf("\nPreview of transactions to be deleted from category '%s':\n", categoryName)

	rows, err := db.Query(`
		SELECT t.id, t.transaction_date, t.amount, t.description, t.category_id
		FROM transactions t
		WHERE t.category_id = ?
		ORDER BY t.transaction_date DESC
		LIMIT 5
	`, categoryID)
	if err != nil {
		utils.PrintError("retrieving transactions", err)
		return
	}
	defer rows.Close()

	var previewTransactions []types.TableTransaction
	for rows.Next() {
		var t types.TableTransaction
		err := rows.Scan(&t.Id, &t.Date, &t.Amount, &t.Description, &t.CategoryID)
		if err != nil {
			utils.PrintError("reading transaction", err)
			return
		}
		previewTransactions = append(previewTransactions, t)
	}

	if len(previewTransactions) > 0 {
		if err := utils.PrintTransactionTable(db, previewTransactions, true); err != nil {
			utils.PrintError("displaying transactions", err)
			return
		}
	}

	if transactionCount > 5 {
		fmt.Printf("  ... and %d more transactions\n", transactionCount-5)
	}

	confirmCategoryDeletion(db, reader, categoryID, categoryName, transactionCount)
}

func handleMoveToUncategorized(db *sql.DB, reader *bufio.Reader, categoryID int, categoryName string, transactionCount int) {
	// Get or create 'Uncategorized' category
	uncategorizedID, err := database.GetCategoryID(db, "Uncategorized")
	if err != nil {
		utils.PrintError("getting uncategorized category", err)
		return
	}

	// Show preview of transactions to be moved
	fmt.Printf("\nPreview of transactions to be moved from '%s' to 'Uncategorized':\n", categoryName)

	rows, err := db.Query(`
		SELECT t.id, t.transaction_date, t.amount, t.description, t.category_id
		FROM transactions t
		WHERE t.category_id = ?
		ORDER BY t.transaction_date DESC
		LIMIT 5
	`, categoryID)
	if err != nil {
		utils.PrintError("retrieving transactions", err)
		return
	}
	defer rows.Close()

	var previewTransactions []types.TableTransaction
	for rows.Next() {
		var t types.TableTransaction
		err := rows.Scan(&t.Id, &t.Date, &t.Amount, &t.Description, &t.CategoryID)
		if err != nil {
			utils.PrintError("reading transaction", err)
			return
		}
		previewTransactions = append(previewTransactions, t)
	}

	if len(previewTransactions) > 0 {
		if err := utils.PrintTransactionTable(db, previewTransactions, true); err != nil {
			utils.PrintError("displaying transactions", err)
			return
		}
	}

	if transactionCount > 5 {
		fmt.Printf("  ... and %d more transactions\n", transactionCount-5)
	}

	// Confirm the operation
	confirmPrompt := fmt.Sprintf("\nAre you sure you want to move %d transaction(s) from '%s' to 'Uncategorized' and delete the category? (yes/no): ",
		transactionCount, categoryName)
	confirmInput, err := utils.PromptInput(reader, confirmPrompt)
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Category deletion cancelled.")
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

	// Move transactions to uncategorized
	result, err := tx.Exec(`UPDATE transactions SET category_id = ? WHERE category_id = ?`, uncategorizedID, categoryID)
	if err != nil {
		utils.PrintError("moving transactions to uncategorized", err)
		return
	}

	// Delete the category
	_, err = tx.Exec(`DELETE FROM categories WHERE id = ?`, categoryID)
	if err != nil {
		utils.PrintError("deleting category", err)
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		utils.PrintError("committing transaction", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("\nSuccessfully moved %d transaction(s) to 'Uncategorized' and deleted category '%s'\n", rowsAffected, categoryName)
}

func confirmCategoryDeletion(db *sql.DB, reader *bufio.Reader, categoryID int, categoryName string, transactionCount int) {
	var confirmPrompt string
	if transactionCount > 0 {
		confirmPrompt = fmt.Sprintf("\nAre you sure you want to delete category '%s' and all %d transaction(s) in it? (yes/no): ",
			categoryName, transactionCount)
	} else {
		confirmPrompt = fmt.Sprintf("\nAre you sure you want to delete category '%s'? (yes/no): ",
			categoryName)
	}

	confirmInput, err := utils.PromptInput(reader, confirmPrompt)
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Category deletion cancelled.")
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

	// Delete transactions in the category (if any)
	if transactionCount > 0 {
		result, err := tx.Exec(`DELETE FROM transactions WHERE category_id = ?`, categoryID)
		if err != nil {
			utils.PrintError("deleting transactions", err)
			return
		}
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("Deleted %d transaction(s) from category '%s'\n", rowsAffected, categoryName)
	}

	// Delete the category
	result, err := tx.Exec(`DELETE FROM categories WHERE id = ?`, categoryID)
	if err != nil {
		utils.PrintError("deleting category", err)
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		utils.PrintError("committing transaction", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Successfully deleted category '%s'\n", categoryName)
	} else {
		fmt.Printf("Category '%s' was not found or already deleted\n", categoryName)
	}
}
