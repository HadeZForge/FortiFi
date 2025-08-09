package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

func ChangeTransactionCategoryCLI(db *sql.DB, reader *bufio.Reader) {
	shortIDInput, err := utils.PromptInput(reader, "Enter the 8-digit transaction ID from the month breakdown report: ")
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
		SELECT t.id, t.transaction_date, t.Amount, t.description, t.category_id
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
		fmt.Printf("Found transaction:\n")
		displayTransaction(db, selectedTxn)
	} else {
		fmt.Printf("Multiple transactions found with ID starting with %s:\n\n", shortIDInput)
		for i, t := range transactions {
			fmt.Printf("%d. ", i+1)
			displayTransaction(db, t)
			fmt.Println()
		}

		selectionInput, err := utils.PromptInput(reader, "Select transaction number: ")
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
		fmt.Printf("\nSelected transaction:\n")
		displayTransaction(db, selectedTxn)
	}

	categories, err := utils.GetAvailableCategories(db)
	if err != nil {
		utils.PrintError("retrieving categories", err)
		return
	}
	newCategoryID, newCategoryName, err := utils.SelectCategory(db, reader, categories, true)
	if err != nil {
		utils.PrintError("selecting category", err)
		return
	}

	// Check if it's the same category
	if newCategoryID == selectedTxn.CategoryID {
		fmt.Printf("Transaction is already in Category '%s'\n", newCategoryName)
		return
	}

	// Update the transaction
	_, err = db.Exec(`UPDATE transactions SET Category_id = ? WHERE id = ?`, newCategoryID, selectedTxn.Id)
	if err != nil {
		utils.PrintError("updating transaction", err)
		return
	}

	fmt.Printf("  Transaction updated successfully!\n")
	fmt.Printf("  Category changed to '%s'\n", newCategoryName)
}

func displayTransaction(db *sql.DB, t types.TableTransaction) {
	parsedDate, err := time.Parse(time.RFC3339, t.Date)
	categoryName, catErr := database.GetCategoryNameByID(db, t.CategoryID)
	if catErr != nil {
		categoryName = fmt.Sprintf("(ID %d)", t.CategoryID)
	}
	if err != nil {
		fmt.Printf("ID: %s | Date: %s | Amount: %s | Category: %s | Description: %s\n",
			t.Id[:8], t.Date, utils.FormatAmount(t.Amount), categoryName, utils.Truncate(t.Description, 40))
		return
	}
	formattedDate := parsedDate.Format("01-02-06")
	fmt.Printf("ID: %s | Date: %s | Amount: %s | Category: %s | Description: %s\n",
		t.Id[:8], formattedDate, utils.FormatAmount(t.Amount), categoryName, utils.Truncate(t.Description, 40))
}
