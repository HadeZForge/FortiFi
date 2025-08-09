package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	cliUtils "github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	txnUtils "github.com/HadeZForge/FortiFi/internal/transaction/utils"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

func AddTransactionCLI(db *sql.DB, reader *bufio.Reader) {
	fmt.Println("Adding transaction")

	var newTransaction types.TableTransaction

	// Get transaction values from the user
	dateInput, err := cliUtils.PromptInput(reader, "\nEnter the transaction date in the format YYYY-MM-DD: ")
	if err != nil {
		cliUtils.PrintError("reading date", err)
		return
	}
	newTransaction.Date = dateInput

	parsedDate, err := time.Parse("2006-01-02", dateInput)
	if err != nil {
		cliUtils.PrintError("parsing date", err)
		return
	}
	descriptionInput, err := cliUtils.PromptInput(reader, "\nEnter a description for the transaction: ")
	if err != nil {
		cliUtils.PrintError("reading description", err)
		return
	}
	newTransaction.Description = descriptionInput

	amountInput, err := cliUtils.PromptInput(reader, "\nEnter transaction amount (4.25 for income or -4.25 for expense): ")
	if err != nil {
		cliUtils.PrintError("reading amount", err)
		return
	}
	parsedAmount, err := strconv.ParseFloat(amountInput, 64)
	if err != nil {
		cliUtils.PrintError("parsing amount", err)
		return
	}
	newTransaction.Amount = parsedAmount

	categories, err := cliUtils.GetAvailableCategories(db)
	if err != nil {
		cliUtils.PrintError("getting available categories", err)
		return
	}

	// Select an existing category or create a new one
	categoryID, _, err := cliUtils.SelectCategory(db, reader, categories, true)
	if err != nil {
		cliUtils.PrintError("selecting category", err)
		return
	}

	newTransaction.CategoryID = categoryID

	accounts, err := cliUtils.GetAvailableAccounts(db)
	if err != nil {
		cliUtils.PrintError("getting available accounts", err)
		return
	}

	// Try to select existing account
	accountID, _, err := cliUtils.SelectAccount(reader, accounts)
	if err != nil {
		// If account not found, create a new one
		if strings.Contains(err.Error(), "not found") {
			// Extract the account name from the error message
			// The error format is "account 'AccountName' not found"
			parts := strings.Split(err.Error(), "'")
			if len(parts) >= 2 {
				accountName := parts[1]
				fmt.Printf("Creating new account: %s\n", accountName)
				accountID, err = database.InsertAccount(db, accountName)
				if err != nil {
					cliUtils.PrintError("creating account", err)
					return
				}
			} else {
				cliUtils.PrintError("parsing account name from error", err)
				return
			}
		} else {
			cliUtils.PrintError("selecting account", err)
			return
		}
	}

	// Generate transaction ID with daily sequence
	newTransaction.Id = txnUtils.GenerateTransactionHash(
		parsedDate,
		newTransaction.Amount,
		newTransaction.Description,
		0)

	// Check if transaction already exists
	exists, err := database.TransactionExists(db, newTransaction.Id)
	if err != nil {
		cliUtils.PrintError("checking transaction existence", err)
		return
	}
	if exists {
		fmt.Printf("Transaction already exists, skipping: %s\n", newTransaction.Id[:8])
		return
	}

	// Confirm transaction details
	fmt.Println("You've entered the following transaction details:")
	err = cliUtils.PrintTransactionTable(db, []types.TableTransaction{newTransaction}, false)
	if err != nil {
		cliUtils.PrintError("printing transaction table", err)
		return
	}
	fmt.Println("Is that correct? (yes/no)")
	confirmInput, err := cliUtils.PromptInput(reader, "")
	if err != nil {
		cliUtils.PrintError("reading confirmation", err)
		return
	}
	if confirmInput != "yes" {
		fmt.Println("Transaction not added.")
		return
	}

	// Insert transaction
	query := `INSERT INTO transactions (id, account_id, category_id, amount, transaction_date, description) 
				VALUES (?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(query, newTransaction.Id, accountID, newTransaction.CategoryID, newTransaction.Amount, newTransaction.Date, newTransaction.Description)
	if err != nil {
		cliUtils.PrintError("inserting transaction", err)
		return
	}
	fmt.Println("Transaction added successfully.")
}
