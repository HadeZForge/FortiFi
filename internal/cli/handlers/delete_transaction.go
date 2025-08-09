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

func DeleteTransactionCLI(db *sql.DB, reader *bufio.Reader) {
	fmt.Println("Delete transactions by:")
	fmt.Println("1. 8-digit transaction ID")
	fmt.Println("2. Category (delete all transactions in category)")
	optionInput, err := utils.PromptInput(reader, "Select option (1 or 2): ")
	if err != nil {
		utils.PrintError("reading option", err)
		return
	}

	switch optionInput {
	case "1":
		fmt.Println()
		deleteByTransactionID(db, reader)
	case "2":
		fmt.Println()
		deleteByCategory(db, reader)
	default:
		fmt.Println("Invalid option. Please select 1 or 2.")
	}
}

func deleteByTransactionID(db *sql.DB, reader *bufio.Reader) {
	shortIDInput, err := utils.PromptInput(reader, "Enter the 8-digit transaction ID: ")
	if err != nil {
		utils.PrintError("reading transaction ID", err)
		return
	}

	if len(shortIDInput) != 8 {
		fmt.Println("Error: Please enter exactly 8 digits")
		return
	}

	// Find all transactions that start with these 8 digits
	rows, err := db.Query(`
		SELECT t.id, t.transaction_date, t.amount, t.description, t.category_id
		FROM transactions t
		WHERE t.id LIKE ? || '%'
		ORDER BY t.transaction_date DESC
	`, shortIDInput)
	if err != nil {
		utils.PrintError("retrieving transactions", err)
		return
	}
	defer rows.Close()

	var transactions []types.TableTransaction
	for rows.Next() {
		var t types.TableTransaction
		err := rows.Scan(&t.Id, &t.Date, &t.Amount, &t.Description, &t.CategoryID)
		if err != nil {
			utils.PrintError("reading transaction", err)
			return
		}
		transactions = append(transactions, t)
	}

	if len(transactions) == 0 {
		fmt.Printf("No transactions found starting with ID: %s\n", shortIDInput)
		return
	}

	var selectedTxn types.TableTransaction
	if len(transactions) == 1 {
		selectedTxn = transactions[0]
		fmt.Printf("\nTransaction:\n")
		if err := utils.PrintTransactionTable(db, []types.TableTransaction{selectedTxn}, false); err != nil {
			utils.PrintError("displaying transaction", err)
			return
		}
	} else {
		fmt.Printf("Multiple transactions found with ID starting with %s:\n\n", shortIDInput)
		for i, t := range transactions {
			fmt.Printf("%d. ", i+1)
			if err := utils.PrintTransactionTable(db, []types.TableTransaction{t}, false); err != nil {
				utils.PrintError("displaying transaction", err)
				return
			}
			fmt.Println()
		}

		selectionInput, err := utils.PromptInput(reader, "Select transaction number to delete: ")
		if err != nil {
			utils.PrintError("reading selection", err)
			return
		}

		selection, err := strconv.Atoi(selectionInput)
		if err != nil || selection < 1 || selection > len(transactions) {
			fmt.Println("Invalid selection")
			return
		}

		selectedTxn = transactions[selection-1]
		fmt.Printf("\nSelected transaction to delete:\n")
		if err := utils.PrintTransactionTable(db, []types.TableTransaction{selectedTxn}, false); err != nil {
			utils.PrintError("displaying transaction", err)
			return
		}
	}

	// Confirm deletion
	confirmInput, err := utils.PromptInput(reader, "Are you sure you want to delete this transaction? (yes/no): ")
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Transaction deletion cancelled.")
		return
	}

	// Delete the transaction
	result, err := db.Exec(`DELETE FROM transactions WHERE id = ?`, selectedTxn.Id)
	if err != nil {
		utils.PrintError("deleting transaction", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		categoryName, err := database.GetCategoryNameByID(db, selectedTxn.CategoryID)
		if err != nil {
			utils.PrintError("getting category name", err)
			return
		}
		fmt.Printf(" Transaction deleted successfully!\n")
		fmt.Printf("  Deleted: %s - %s (%s)\n",
			utils.FormatAmount(selectedTxn.Amount),
			utils.Truncate(selectedTxn.Description, 50),
			categoryName)
	} else {
		fmt.Println("No transaction was deleted.")
	}
}

func deleteByCategory(db *sql.DB, reader *bufio.Reader) {
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

	if transactionCount == 0 {
		fmt.Printf("No transactions found in category '%s'\n", selectedCategoryName)
		return
	}

	// Show preview of transactions to be deleted
	fmt.Printf("\nFound %d transaction(s) in category '%s':\n", transactionCount, selectedCategoryName)

	rows, err := db.Query(`
		SELECT t.id, t.transaction_date, t.amount, t.description, t.category_id
		FROM transactions t
		WHERE t.category_id = ?
		ORDER BY t.transaction_date DESC
		LIMIT 5
	`, selectedCategoryID)
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
	} else {
		fmt.Println("No transactions in this category.")
	}

	if transactionCount > 5 {
		fmt.Printf("  ... and %d more transactions\n", transactionCount-5)
	}

	// Confirm deletion
	confirmPrompt := fmt.Sprintf("\nAre you sure you want to delete ALL %d transactions in category '%s'? (yes/no): ",
		transactionCount, selectedCategoryName)
	confirmInput, err := utils.PromptInput(reader, confirmPrompt)
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Transaction deletion cancelled.")
		return
	}

	// Delete all transactions in the category
	result, err := db.Exec(`DELETE FROM transactions WHERE category_id = ?`, selectedCategoryID)
	if err != nil {
		utils.PrintError("deleting transactions", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf(" Successfully deleted %d transaction(s) from category '%s'\n", rowsAffected, selectedCategoryName)
}
