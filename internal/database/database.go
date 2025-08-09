package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

// Method to create needed tables if they don't already exist. Should be the first thing called upon startup or tables are not guaranteed to exist
func InitTables(db *sql.DB) {
	fmt.Println("Tables initializing")
	// Sql create table commands
	tableCreators := [9]string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			balance REAL NOT NULL DEFAULT 0.0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS account_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			snapshot_time TIMESTAMP,
			balance REAL NOT NULL DEFAULT 0.0,
			account_id INTEGER NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		);`,

		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			account_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			amount REAL NOT NULL,
			transaction_date DATE NOT NULL,
			description TEXT,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS exact_keywords (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			keyword TEXT UNIQUE NOT NULL,
			category_id INTEGER NOT NULL,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS includes_keywords (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			keyword TEXT UNIQUE NOT NULL,
			category_id INTEGER NOT NULL,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS budget_definitions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS budget_definition_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			budget_definition_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			FOREIGN KEY (budget_definition_id) REFERENCES budget_definitions(id) ON DELETE CASCADE,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
			UNIQUE (budget_definition_id, category_id)
		);`,

		`CREATE TABLE IF NOT EXISTS monthly_budget_instances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			budget_definition_id INTEGER NOT NULL,
			budget_month TEXT NOT NULL,
			budget_amount REAL NOT NULL,
			FOREIGN KEY (budget_definition_id) REFERENCES budget_definitions(id) ON DELETE CASCADE,
			UNIQUE (budget_definition_id, budget_month)
		);`}

	for _, element := range tableCreators {
		_, err := db.Exec(element)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Tables initialized successfully!")
}

// / #################################
// / Keywords
// / #################################
func GetExactKeywords(db *sql.DB) (map[string]int, error) {
	query := "SELECT keyword, category_id FROM exact_keywords"
	rows, err := db.Query(query)
	if err != nil {
		fmt.Println("GetExactKeywords failed while querying for keywords")
		return nil, err
	}
	defer rows.Close()

	keywordsMap := make(map[string]int)
	for rows.Next() {
		var keyword string
		var categoryID int
		if err := rows.Scan(&keyword, &categoryID); err != nil {
			fmt.Println("GetExactKeywords failed while scanning rows")
			return nil, err
		}
		keywordsMap[keyword] = categoryID
	}

	if err = rows.Err(); err != nil {
		fmt.Println("GetExactKeywords failed while iterating over rows")
		return nil, err
	}
	return keywordsMap, nil
}

func GetIncludesKeywords(db *sql.DB) (map[string]int, error) {
	query := "SELECT keyword, category_id FROM includes_keywords"
	rows, err := db.Query(query)
	if err != nil {
		fmt.Println("GetIncludesKeywords failed while querying for keywords")
		return nil, err
	}
	defer rows.Close()

	keywordsMap := make(map[string]int)
	for rows.Next() {
		var keyword string
		var categoryID int
		if err := rows.Scan(&keyword, &categoryID); err != nil {
			fmt.Println("GetIncludesKeywords failed while scanning rows")
			return nil, err
		}
		keywordsMap[keyword] = categoryID
	}

	if err = rows.Err(); err != nil {
		fmt.Println("GetIncludesKeywords failed while iterating over rows")
		return nil, err
	}
	return keywordsMap, nil
}

/// #################################
/// Category
/// #################################

// GetCategoryNameByID retrieves the category name by its ID
func GetCategoryNameByID(db *sql.DB, categoryID int) (string, error) {
	var categoryName string
	query := "SELECT name FROM categories WHERE id = ?"
	err := db.QueryRow(query, categoryID).Scan(&categoryName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("category with ID %d not found", categoryID)
		}
		return "", err
	}
	return categoryName, nil
}

// GetCategoryID fetches the category ID for a given category name
func GetCategoryID(db *sql.DB, categoryName string) (int, error) {
	// Validate category name
	trimmedName := strings.TrimSpace(categoryName)
	if trimmedName == "" {
		return 0, fmt.Errorf("category name cannot be empty or whitespace only")
	}

	var categoryID int
	query := "SELECT id FROM categories WHERE name = ?"

	err := db.QueryRow(query, trimmedName).Scan(&categoryID)
	if err == sql.ErrNoRows {
		return InsertCategory(db, trimmedName)
	} else if err != nil {
		return 0, err
	}

	return categoryID, nil
}

// InsertCategory inserts a new category and returns the inserted ID
func InsertCategory(db *sql.DB, categoryName string) (int, error) {
	// Validate category name
	trimmedName := strings.TrimSpace(categoryName)
	if trimmedName == "" {
		return 0, fmt.Errorf("category name cannot be empty or whitespace only")
	}

	query := "INSERT INTO categories (name) VALUES (?)"
	result, err := db.Exec(query, trimmedName)
	if err != nil {
		return 0, err
	}

	// Get the last inserted ID
	categoryID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(categoryID), nil
}

/// #################################
/// Account
/// #################################

// GetAccountID fetches the account ID for a given account name
func GetAccountID(db *sql.DB, accountName string) (int, error) {
	// Validate account name
	trimmedName := strings.TrimSpace(accountName)
	if trimmedName == "" {
		return 0, fmt.Errorf("account name cannot be empty or whitespace only")
	}

	var accountID int
	query := "SELECT id FROM accounts WHERE name = ?"

	err := db.QueryRow(query, trimmedName).Scan(&accountID)
	if err == sql.ErrNoRows {
		return InsertAccount(db, trimmedName)
	} else if err != nil {
		return 0, err
	}

	return accountID, nil
}

// GetAccountID fetches the account name for a given account id
func GetAccountName(db *sql.DB, accountId int) (string, error) {
	var accountName string
	query := "SELECT name FROM accounts WHERE id = ?"

	err := db.QueryRow(query, accountId).Scan(&accountName)
	if err != nil {
		return "Account with ID: " + fmt.Sprintf("%d", accountId) + " does not exist", err
	}

	return accountName, nil
}

// InsertAccount inserts a new account and returns the inserted ID
func InsertAccount(db *sql.DB, accountName string) (int, error) {
	// Validate account name
	trimmedName := strings.TrimSpace(accountName)
	if trimmedName == "" {
		return 0, fmt.Errorf("account name cannot be empty or whitespace only")
	}

	query := "INSERT INTO accounts (name) VALUES (?)"
	result, err := db.Exec(query, trimmedName)
	if err != nil {
		return 0, err
	}

	// Get the last inserted ID
	accountID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(accountID), nil
}

// GetAccountHistory retrieves all snapshots for a given accountID.
func GetAccountHistory(db *sql.DB, accountID int) ([]types.AccountSnapshot, error) {
	// SQL query to retrieve all account snapshots for the given accountID
	query := `SELECT id, snapshot_time, balance, account_id
	          FROM account_snapshots
	          WHERE account_id = ?
	          ORDER BY snapshot_time`

	// Prepare the query
	rows, err := db.Query(query, accountID)
	if err != nil {
		return nil, fmt.Errorf("could not execute query: %w", err)
	}
	defer rows.Close()

	var snapshots []types.AccountSnapshot

	// Iterate over the rows and scan data into AccountSnapshot struct
	for rows.Next() {
		var snapshot types.AccountSnapshot
		if err := rows.Scan(&snapshot.ID, &snapshot.SnapshotTime, &snapshot.Balance, &snapshot.AccountID); err != nil {
			return nil, fmt.Errorf("could not scan row: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	// Check for errors during row iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return snapshots, nil
}

// InsertCategory inserts a new category and returns the inserted ID
func InsertAccountSnapshot(db *sql.DB, accountID int, snapshotTime time.Time, newBalance float64) (int, error) {
	var existingID int

	// Check if a snapshot already exists with the same timestamp and balance
	queryCheck := `SELECT id FROM account_snapshots WHERE account_id = ? AND snapshot_time = ? AND balance = ?`
	err := db.QueryRow(queryCheck, accountID, snapshotTime, newBalance).Scan(&existingID)
	if err == nil {
		// Snapshot already exists, return the existing ID
		return existingID, nil
	} else if err != sql.ErrNoRows {
		// An actual database error occurred
		return 0, err
	}

	// Insert a new snapshot if no existing record matches
	queryInsert := `INSERT INTO account_snapshots (account_id, snapshot_time, balance) 
	          VALUES (?, ?, ?)`
	result, err := db.Exec(queryInsert, accountID, snapshotTime, newBalance)
	if err != nil {
		return 0, err
	}

	// Get the new snapshotID
	snapshotID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(snapshotID), nil
}

// TransactionExists checks if a transaction with the given ID already exists
func TransactionExists(db *sql.DB, transactionID string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM transactions WHERE id = ?)"
	err := db.QueryRow(query, transactionID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// / #################################
// / Budgets
// / #################################
func InsertBudgetDefinition(db *sql.DB, name string, description string, categoryIDs []int) (int, error) {
	// Insert budget definition
	query := `INSERT INTO budget_definitions (name, description) VALUES (?, ?)`
	result, err := db.Exec(query, name, description)
	if err != nil {
		return 0, err
	}

	// Get the newly inserted budget definition ID
	budgetDefinitionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Insert category associations
	for _, categoryID := range categoryIDs {
		_, err := db.Exec(`INSERT INTO budget_definition_categories (budget_definition_id, category_id) VALUES (?, ?)`, budgetDefinitionID, categoryID)
		if err != nil {
			return 0, err
		}
	}

	return int(budgetDefinitionID), nil
}

func InsertMonthlyBudgetInstance(db *sql.DB, budgetDefinitionID int, budgetMonth string, budgetAmount float64) error {
	// Insert monthly budget instance
	query := `INSERT INTO monthly_budget_instances (budget_definition_id, budget_month, budget_amount) VALUES (?, ?, ?)`
	_, err := db.Exec(query, budgetDefinitionID, budgetMonth, budgetAmount)
	return err
}

func GetBudgetDefinitionsByName(db *sql.DB, budgetName string) ([]map[string]any, error) {
	query := `SELECT id, name, description, created_at FROM budget_definitions WHERE name = ? ORDER BY created_at DESC`
	rows, err := db.Query(query, budgetName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgetDefinitions []map[string]any
	for rows.Next() {
		var id int
		var name string
		var description string
		var createdAt string
		if err := rows.Scan(&id, &name, &description, &createdAt); err != nil {
			return nil, err
		}

		budgetDefinitions = append(budgetDefinitions, map[string]any{
			"id":          id,
			"name":        name,
			"description": description,
			"created_at":  createdAt,
		})
	}

	return budgetDefinitions, nil
}

func GetMonthlyBudgetInstancesByDefinition(db *sql.DB, budgetDefinitionID int) ([]map[string]any, error) {
	query := `SELECT id, budget_definition_id, budget_month, budget_amount FROM monthly_budget_instances WHERE budget_definition_id = ? ORDER BY budget_month DESC`
	rows, err := db.Query(query, budgetDefinitionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []map[string]any
	for rows.Next() {
		var id int
		var definitionID int
		var month string
		var amount float64
		if err := rows.Scan(&id, &definitionID, &month, &amount); err != nil {
			return nil, err
		}

		instances = append(instances, map[string]any{
			"id":                   id,
			"budget_definition_id": definitionID,
			"budget_month":         month,
			"budget_amount":        amount,
		})
	}

	return instances, nil
}

func GetBudgetDefinitionWithCategories(db *sql.DB, budgetDefinitionID int) (string, string, []int, error) {
	var budgetName string
	var budgetDescription string

	// Get budget definition details
	err := db.QueryRow(`SELECT name, description FROM budget_definitions WHERE id = ?`, budgetDefinitionID).
		Scan(&budgetName, &budgetDescription)
	if err != nil {
		return "", "", nil, err
	}

	// Get associated categories
	rows, err := db.Query(`SELECT category_id FROM budget_definition_categories WHERE budget_definition_id = ?`, budgetDefinitionID)
	if err != nil {
		return "", "", nil, err
	}
	defer rows.Close()

	var categoryIDs []int
	for rows.Next() {
		var categoryID int
		if err := rows.Scan(&categoryID); err != nil {
			return "", "", nil, err
		}
		categoryIDs = append(categoryIDs, categoryID)
	}

	return budgetName, budgetDescription, categoryIDs, nil
}

func GetMonthlyBudgetInstance(db *sql.DB, budgetDefinitionID int, budgetMonth string) (float64, error) {
	var budgetAmount float64
	err := db.QueryRow(`SELECT budget_amount FROM monthly_budget_instances WHERE budget_definition_id = ? AND budget_month = ?`,
		budgetDefinitionID, budgetMonth).Scan(&budgetAmount)
	return budgetAmount, err
}

func DeleteBudgetDefinition(db *sql.DB, budgetDefinitionID int) error {
	query := `DELETE FROM budget_definitions WHERE id = ?`
	_, err := db.Exec(query, budgetDefinitionID)
	return err
}

func UpdateBudgetDefinition(db *sql.DB, budgetDefinitionID int, newName string, newDescription string, categoryIDs []int) error {
	// Update the budget definition
	query := `UPDATE budget_definitions SET name = ?, description = ? WHERE id = ?`
	_, err := db.Exec(query, newName, newDescription, budgetDefinitionID)
	if err != nil {
		return err
	}

	// Remove old category associations
	_, err = db.Exec(`DELETE FROM budget_definition_categories WHERE budget_definition_id = ?`, budgetDefinitionID)
	if err != nil {
		return err
	}

	// Insert new category associations
	for _, categoryID := range categoryIDs {
		_, err := db.Exec(`INSERT INTO budget_definition_categories (budget_definition_id, category_id) VALUES (?, ?)`, budgetDefinitionID, categoryID)
		if err != nil {
			return err
		}
	}

	return nil
}

func UpdateMonthlyBudgetInstance(db *sql.DB, budgetDefinitionID int, budgetMonth string, newAmount float64) error {
	query := `UPDATE monthly_budget_instances SET budget_amount = ? WHERE budget_definition_id = ? AND budget_month = ?`
	_, err := db.Exec(query, newAmount, budgetDefinitionID, budgetMonth)
	return err
}

func GetMonthlyBudgetInstancesByMonth(db *sql.DB, month string) ([]map[string]any, error) {
	query := `
		SELECT mbi.id, mbi.budget_definition_id, mbi.budget_month, mbi.budget_amount, 
		       bd.name as budget_name, bd.description
		FROM monthly_budget_instances mbi
		JOIN budget_definitions bd ON mbi.budget_definition_id = bd.id
		WHERE mbi.budget_month = ?
		ORDER BY bd.name
	`
	rows, err := db.Query(query, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgetInstances []map[string]any
	for rows.Next() {
		var id int
		var definitionID int
		var budgetMonth string
		var amount float64
		var budgetName string
		var description string

		if err := rows.Scan(&id, &definitionID, &budgetMonth, &amount, &budgetName, &description); err != nil {
			return nil, err
		}

		budgetInstances = append(budgetInstances, map[string]any{
			"id":                   id,
			"budget_definition_id": definitionID,
			"budget_month":         budgetMonth,
			"budget_amount":        amount,
			"budget_name":          budgetName,
			"description":          description,
		})
	}

	return budgetInstances, nil
}

// Helper function to roll over budgets from one month to another
func RollOverBudgets(db *sql.DB, fromMonth string, toMonth string) error {
	query := `
		INSERT INTO monthly_budget_instances (budget_definition_id, budget_month, budget_amount)
		SELECT budget_definition_id, ?, budget_amount 
		FROM monthly_budget_instances 
		WHERE budget_month = ?
	`
	_, err := db.Exec(query, toMonth, fromMonth)
	return err
}

// CheckAndCreateMissingMonthlyInstances checks for budget definitions without monthly instances
// and creates them based on the last available instance amount
func CheckAndCreateMissingMonthlyInstances(db *sql.DB) error {
	// Get all budget definitions
	query := `SELECT id FROM budget_definitions`
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get budget definitions: %w", err)
	}
	defer rows.Close()

	var budgetDefinitionIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan budget definition ID: %w", err)
		}
		budgetDefinitionIDs = append(budgetDefinitionIDs, id)
	}

	currentMonth := time.Now().Format("2006-01")
	createdCount := 0

	for _, budgetID := range budgetDefinitionIDs {
		// Get the last monthly instance for this budget
		lastInstanceQuery := `
			SELECT budget_month, budget_amount 
			FROM monthly_budget_instances 
			WHERE budget_definition_id = ? 
			ORDER BY budget_month DESC 
			LIMIT 1`

		var lastMonth string
		var lastAmount float64
		err := db.QueryRow(lastInstanceQuery, budgetID).Scan(&lastMonth, &lastAmount)
		if err != nil {
			if err == sql.ErrNoRows {
				// No instances exist, skip this budget
				continue
			}
			return fmt.Errorf("failed to get last instance for budget %d: %w", budgetID, err)
		}

		// Generate missing months between last instance and current month
		missingMonths := generateMissingMonths(lastMonth, currentMonth)
		if len(missingMonths) == 0 {
			continue
		}

		// Create missing monthly instances
		for _, month := range missingMonths {
			err := InsertMonthlyBudgetInstance(db, budgetID, month, lastAmount)
			if err != nil {
				return fmt.Errorf("failed to create monthly instance for budget %d, month %s: %w", budgetID, month, err)
			}
			createdCount++
		}
	}

	if createdCount > 0 {
		fmt.Printf("Created %d missing monthly budget instances\n", createdCount)
	}

	return nil
}

// generateMissingMonths generates a list of months between startMonth and endMonth (inclusive of endMonth)
func generateMissingMonths(startMonth, endMonth string) []string {
	start, err := time.Parse("2006-01", startMonth)
	if err != nil {
		return nil
	}

	end, err := time.Parse("2006-01", endMonth)
	if err != nil {
		return nil
	}

	var months []string
	current := start.AddDate(0, 1, 0) // Start from the month after startMonth

	for current.Before(end) || current.Equal(end) {
		months = append(months, current.Format("2006-01"))
		current = current.AddDate(0, 1, 0)
	}

	return months
}
