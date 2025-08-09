package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	txnUtils "github.com/HadeZForge/FortiFi/internal/transaction/utils"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

func SplitTransactionCLI(db *sql.DB, reader *bufio.Reader) {
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
		SELECT t.id, t.transaction_date, t.amount, t.description, t.category_id, t.account_id
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
		err := rows.Scan(&t.Id, &t.Date, &t.Amount, &t.Description, &t.CategoryID, &t.AccountID)
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
		fmt.Printf("\nTransaction to split:\n")
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

		selectionInput, err := utils.PromptInput(reader, "Select transaction number to split: ")
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
		fmt.Printf("\nSelected transaction to split:\n")
		if err := utils.PrintTransactionTable(db, []types.TableTransaction{selectedTxn}, false); err != nil {
			utils.PrintError("displaying transaction", err)
			return
		}
	}

	// Display the transaction amount
	fmt.Printf("Transaction amount: %s\n", utils.FormatAmount(selectedTxn.Amount))

	// Prompt for split amount
	splitAmountInput, err := utils.PromptInput(reader, fmt.Sprintf("Enter amount to split off (e.g., %s): ", utils.FormatAmount(selectedTxn.Amount/2)))
	if err != nil {
		utils.PrintError("reading split amount", err)
		return
	}

	splitAmount, err := strconv.ParseFloat(splitAmountInput, 64)
	if err != nil {
		fmt.Println("Error: Please enter a valid number")
		return
	}

	// Validate split amount
	if splitAmount == 0 {
		fmt.Println("Error: Split amount cannot be zero")
		return
	}

	// Force the split amount to match the sign of the original transaction
	if (selectedTxn.Amount > 0 && splitAmount < 0) || (selectedTxn.Amount < 0 && splitAmount > 0) {
		splitAmount = -splitAmount
		fmt.Printf("Note: Adjusted split amount to match transaction sign: %s\n", utils.FormatAmount(splitAmount))
	}

	if abs(splitAmount) >= abs(selectedTxn.Amount) {
		utils.PrintError("invalid split amount", fmt.Errorf("split amount (%s) must be less than the original transaction amount (%s)",
			utils.FormatAmount(splitAmount), utils.FormatAmount(selectedTxn.Amount)))
		return
	}

	// Calculate remaining amount
	remainingAmount := selectedTxn.Amount - splitAmount

	// Get available categories for the new transaction
	categories, err := utils.GetAvailableCategories(db)
	if err != nil {
		utils.PrintError("retrieving categories", err)
		return
	}

	if len(categories) == 0 {
		fmt.Println("No categories found.")
		return
	}

	// Prompt for category of the new transaction
	fmt.Printf("\nSelect category for the split transaction (amount: %s):\n", utils.FormatAmount(splitAmount))
	selectedCategoryID, selectedCategoryName, err := utils.SelectCategory(db, reader, categories, true)
	if err != nil {
		utils.PrintError("selecting category", err)
		return
	}

	// Prompt for optional note to append to description
	noteInput, err := utils.PromptInput(reader, "Enter optional note to append to description (or press Enter to skip): ")
	if err != nil {
		utils.PrintError("reading note", err)
		return
	}

	// Create the description for the split transaction
	splitDescription := selectedTxn.Description
	if strings.TrimSpace(noteInput) != "" {
		splitDescription = selectedTxn.Description + " - " + strings.TrimSpace(noteInput)
	}

	// Show confirmation
	fmt.Printf("\nSplit Transaction Summary:\n")
	fmt.Printf("Original transaction: %s - %s\n", utils.FormatAmount(selectedTxn.Amount), utils.Truncate(selectedTxn.Description, 50))
	fmt.Printf("Split amount: %s - %s (%s)\n", utils.FormatAmount(splitAmount), utils.Truncate(splitDescription, 50), selectedCategoryName)
	fmt.Printf("Remaining amount: %s - %s\n", utils.FormatAmount(remainingAmount), utils.Truncate(selectedTxn.Description, 50))

	// Confirm the split
	confirmInput, err := utils.PromptInput(reader, "\nAre you sure you want to proceed with this split? (yes/no): ")
	if err != nil {
		utils.PrintError("reading confirmation", err)
		return
	}
	confirmInput = strings.ToLower(confirmInput)

	if confirmInput != "yes" && confirmInput != "y" {
		fmt.Println("Transaction split cancelled.")
		return
	}

	// Parse the transaction date
	transactionDate, err := utils.ParseDate(selectedTxn.Date)
	if err != nil {
		utils.PrintError("parsing transaction date", err)
		return
	}

	// Generate transaction ID for the split transaction
	splitTransactionID := txnUtils.GenerateTransactionHash(transactionDate, splitAmount, splitDescription, 0)

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

	// Insert the new split transaction
	_, err = tx.Exec(`
		INSERT INTO transactions (id, account_id, category_id, amount, transaction_date, description)
		VALUES (?, ?, ?, ?, ?, ?)
	`, splitTransactionID, selectedTxn.AccountID, selectedCategoryID, splitAmount, selectedTxn.Date, splitDescription)
	if err != nil {
		utils.PrintError("inserting split transaction", err)
		return
	}

	// Update the original transaction amount
	_, err = tx.Exec(`
		UPDATE transactions 
		SET amount = ? 
		WHERE id = ?
	`, remainingAmount, selectedTxn.Id)
	if err != nil {
		utils.PrintError("updating original transaction", err)
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		utils.PrintError("committing transaction", err)
		return
	}

	fmt.Printf("\nTransaction split successfully!\n")
	fmt.Printf("  Original transaction updated: %s - %s\n", utils.FormatAmount(remainingAmount), utils.Truncate(selectedTxn.Description, 50))
	fmt.Printf("  New transaction created: %s - %s (%s) [ID: %s]\n", utils.FormatAmount(splitAmount), utils.Truncate(splitDescription, 50), selectedCategoryName, splitTransactionID[:8])
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
