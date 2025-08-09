package utils

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/HadeZForge/FortiFi/internal/database"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

const (
	Reset = "\033[0m"
	Red   = "\033[31m"
	Green = "\033[32m"
	Blue  = "\033[34m"
)

func DisplayCommands(commands []types.Command) {
	fmt.Println("\nAvailable Commands:")

	// Find the longest tag to determine tab spacing
	maxTagLen := 0
	for _, cmd := range commands {
		tagLen := len(cmd.Tag) + 2 // +2 for parentheses
		if tagLen > maxTagLen {
			maxTagLen = tagLen
		}
	}

	// Display each command with proper alignment
	for _, cmd := range commands {
		tagStr := fmt.Sprintf("(%s)", cmd.Tag)
		// Calculate spaces needed for alignment
		spaces := maxTagLen - len(tagStr) + 4 // +4 for extra spacing
		fmt.Printf("%s%s%s\n", tagStr, strings.Repeat(" ", spaces), cmd.Name)
	}
}

// FormatAmount formats amounts with optional color and spacing
func FormatAmount(amount float64) string {
	raw := fmt.Sprintf("%.2f", abs(amount))
	if amount < 0 {
		return fmt.Sprintf("%s-$%s%s", Red, raw, Reset)
	}
	return fmt.Sprintf("%s $%s%s", Green, raw, Reset)
}

// FormatAmountPlain strips color formatting (used for delta display)
func FormatAmountPlain(amount float64) string {
	raw := fmt.Sprintf("%.2f", abs(amount))
	raw = strings.TrimSuffix(strings.TrimSuffix(raw, "0"), ".")
	return "$" + raw
}

// FormatDelta returns formatted delta string with color
func FormatDelta(delta float64) string {
	raw := fmt.Sprintf("%.2f", abs(delta))
	switch {
	case delta > 0:
		return fmt.Sprintf("%s+%s%s", Green, "$"+raw, Reset)
	case delta < 0:
		return fmt.Sprintf("%s-%s%s", Red, "$"+raw, Reset)
	default:
		return fmt.Sprintf("%s $0.00%s", Blue, Reset)
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func ResolveTransactionID(db *sql.DB, shortID string) (string, error) {
	rows, err := db.Query(`SELECT id FROM transactions WHERE id LIKE ?`, shortID+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var matches []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		matches = append(matches, id)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no transaction matches ID starting with '%s'", shortID)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple transactions match ID '%s': %v", shortID, matches)
	}
	return matches[0], nil
}

// SelectTransaction prompts the user to select a transaction by ID.
// Returns the selected transaction or an error.
func SelectTransaction(db *sql.DB, reader *bufio.Reader) (types.TableTransaction, error) {
	input, err := PromptInput(reader, "Enter transaction ID (first 8 characters): ")
	if err != nil {
		return types.TableTransaction{}, err
	}

	if len(input) != 8 {
		return types.TableTransaction{}, fmt.Errorf("transaction ID must be exactly 8 characters")
	}

	// Resolve the short ID to full ID
	fullID, err := ResolveTransactionID(db, input)
	if err != nil {
		return types.TableTransaction{}, err
	}

	// Get the transaction details
	query := `
		SELECT t.id, t.account_id, t.category_id, t.amount, t.transaction_date, t.description
		FROM transactions t
		WHERE t.id = ?
	`
	var transaction types.TableTransaction
	err = db.QueryRow(query, fullID).Scan(
		&transaction.Id,
		&transaction.AccountID,
		&transaction.CategoryID,
		&transaction.Amount,
		&transaction.Date,
		&transaction.Description,
	)
	if err != nil {
		return types.TableTransaction{}, err
	}

	return transaction, nil
}

func GetAvailableCategories(db *sql.DB) ([]types.CategoryInfo, error) {
	query := `
		SELECT c.id, c.name, COUNT(t.id) as transaction_count
		FROM categories c
		LEFT JOIN transactions t ON c.id = t.category_id
		GROUP BY c.id, c.name
		ORDER BY c.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []types.CategoryInfo
	for rows.Next() {
		var category types.CategoryInfo
		if err := rows.Scan(&category.Id, &category.Name, &category.TransactionCount); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, nil
}

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visibleLength(s string) int {
	return utf8.RuneCountInString(ansiRegexp.ReplaceAllString(s, ""))
}

func PadAnsi(s string, width int) string {
	visible := visibleLength(s)
	if visible >= width {
		return s
	}
	padding := strings.Repeat(" ", width-visible)
	return s + padding
}

func FormatRow(columns []string, widths []int) string {
	if len(columns) != len(widths) {
		return ""
	}
	var parts []string
	for i, col := range columns {
		parts = append(parts, PadAnsi(col, widths[i]))
	}
	return strings.Join(parts, " | ")
}

func Sum(ints []int) int {
	total := 0
	for _, v := range ints {
		total += v
	}
	return total
}

func PrintCategoriesInColumns(categories []types.CategoryInfo, columns int) {
	if len(categories) == 0 {
		return
	}

	// Calculate the maximum width needed for each column
	maxWidth := 0
	for i, cat := range categories {
		entry := fmt.Sprintf("%d. %s", i+1, cat.Name)
		if len(entry) > maxWidth {
			maxWidth = len(entry)
		}
	}

	// Add some padding
	maxWidth += 3

	// Print categories in columns
	rows := (len(categories) + columns - 1) / columns // Ceiling division
	for row := range rows {
		for col := range columns {
			index := col*rows + row
			if index < len(categories) {
				entry := fmt.Sprintf("%d. %s", index+1, categories[index].Name)
				fmt.Printf("%-*s", maxWidth, entry)
			}
		}
		fmt.Println()
	}
}

func PrintTransactionTable(db *sql.DB, transactions []types.TableTransaction, includeHeader bool) error {
	if includeHeader {
		fmt.Printf("%-10s | %-20s | %-30s | %-8s | %-8s\n", "Date", "Category", "Description", "Txn ID", "Amount")
		fmt.Println(strings.Repeat("-", 90))

		for _, t := range transactions {
			parsedDate, err := ParseDate(t.Date)
			if err != nil {
				return fmt.Errorf("error parsing date: %w", err)
			}
			formattedDate := parsedDate.Format("01-02-06")
			amountStr := FormatAmount(t.Amount)

			categoryStr, err := database.GetCategoryNameByID(db, t.CategoryID)
			if err != nil {
				return fmt.Errorf("failed to print transaction due to: %w", err)
			}
			if strings.ToLower(categoryStr) == "uncategorized" {
				categoryStr = fmt.Sprintf("%s%s%s", Blue, categoryStr, Reset)
			}

			// Use FormatRow to handle ANSI color codes properly
			columns := []string{
				formattedDate,
				categoryStr,
				Truncate(t.Description, 30),
				t.Id[:8],
				amountStr,
			}
			widths := []int{10, 20, 30, 8, 8}
			fmt.Println(FormatRow(columns, widths))
		}
	} else {
		for _, t := range transactions {
			parsedDate, err := ParseDate(t.Date)
			if err != nil {
				return fmt.Errorf("error parsing date: %w", err)
			}
			formattedDate := parsedDate.Format("01-02-06")
			amountStr := FormatAmount(t.Amount)

			categoryStr, err := database.GetCategoryNameByID(db, t.CategoryID)
			if err != nil {
				return fmt.Errorf("failed to print transaction due to: %w", err)
			}
			if strings.ToLower(categoryStr) == "uncategorized" {
				categoryStr = fmt.Sprintf("%s%s%s", Blue, categoryStr, Reset)
			}

			fmt.Printf("Date: %s | Category: %s | Description: %s | Txn ID: %s | Amount: %s\n",
				formattedDate, categoryStr, Truncate(t.Description, 30), t.Id[:8], amountStr)

		}
		fmt.Println()
	}
	return nil
}

func ParseDate(dateStr string) (time.Time, error) {
	parsedDate, err := time.Parse(time.RFC3339, dateStr)
	if err == nil {
		return parsedDate, nil
	}
	// Try "2006-01-02" as fallback
	parsedDate, err = time.Parse("2006-01-02", dateStr)
	if err == nil {
		return parsedDate, nil
	}
	return time.Time{}, fmt.Errorf("could not parse date '%s' as RFC3339 or YYYY-MM-DD: %w", dateStr, err)
}

// PromptInput prints a prompt, reads a line from the user, trims it, and returns the result.
func PromptInput(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// PrintError prints errors in a consistent format with context.
func PrintError(context string, err error) {
	fmt.Printf("Error %s: %v\n", context, err)
}

// PrintWarning prints warnings in a consistent format with context.
func PrintWarning(context string, err error) {
	fmt.Printf("Warning %s: %v\n", context, err)
}

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SelectCategory prompts the user to select a category by number or name from a list.
// If createNew is true, the user can create a new category.
// Returns the selected category's ID and name, or an error.
func SelectCategory(db *sql.DB, reader *bufio.Reader, categories []types.CategoryInfo, createNew bool) (int, string, error) {
	if len(categories) == 0 {
		return 0, "", fmt.Errorf("no categories available")
	}
	PrintCategoriesInColumns(categories, 3)
	input, err := PromptInput(reader, "\nEnter category name (or number from list): ")
	if err != nil {
		return 0, "", err
	}
	// Try number selection
	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(categories) {
			return categories[num-1].Id, categories[num-1].Name, nil
		}
		return 0, "", fmt.Errorf("invalid category number")
	}
	// Try name selection
	for _, cat := range categories {
		if strings.EqualFold(cat.Name, input) {
			return cat.Id, cat.Name, nil
		}
	}
	if createNew {
		// If category not found, create a new one
		fmt.Printf("Creating new category: %s\n", input)
		categoryID, err := database.InsertCategory(db, input)
		if err != nil {
			PrintError("creating category", err)
			return 0, "", err
		}
		return categoryID, input, nil
	}
	return 0, "", fmt.Errorf("category '%s' not found", input)
}

// SelectAccount prompts the user to select an account by number or name from a list.
// Returns the selected account's ID and name, or an error.
func SelectAccount(reader *bufio.Reader, accounts []AccountInfo) (int, string, error) {
	if len(accounts) == 0 {
		return 0, "", fmt.Errorf("no accounts available")
	}
	PrintAccountsList(accounts)
	input, err := PromptInput(reader, "\nEnter account name (or number from list): ")
	if err != nil {
		return 0, "", err
	}
	// Try number selection
	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(accounts) {
			return accounts[num-1].Id, accounts[num-1].Name, nil
		}
		return 0, "", fmt.Errorf("invalid account number")
	}
	// Try name selection
	for _, acc := range accounts {
		if strings.EqualFold(acc.Name, input) {
			return acc.Id, acc.Name, nil
		}
	}
	return 0, "", fmt.Errorf("account '%s' not found", input)
}

// AccountInfo represents account information for selection
type AccountInfo struct {
	Id   int
	Name string
}

// GetAvailableAccounts retrieves all available accounts from the database
func GetAvailableAccounts(db *sql.DB) ([]AccountInfo, error) {
	rows, err := db.Query(`SELECT DISTINCT a.id, a.name FROM accounts a ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []AccountInfo
	for rows.Next() {
		var account AccountInfo
		err := rows.Scan(&account.Id, &account.Name)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// PrintAccountsList prints a list of accounts in a formatted way
func PrintAccountsList(accounts []AccountInfo) {
	fmt.Print("\nAvailable Accounts:\n")
	for _, account := range accounts {
		fmt.Printf("%d. %s\n", account.Id, account.Name)
	}
}

// PrintAccounts prints all available accounts from the database.
func PrintAccounts(db *sql.DB) error {
	accounts, err := GetAvailableAccounts(db)
	if err != nil {
		PrintError("retrieving accounts", err)
		return err
	}

	PrintAccountsList(accounts)
	return nil
}

// GetBudgetDefinitions retrieves all budget definitions from the database
func GetBudgetDefinitions(db *sql.DB) ([]map[string]any, error) {
	query := `SELECT id, name, description, created_at FROM budget_definitions ORDER BY name`
	rows, err := db.Query(query)
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

// SelectBudget prompts the user to select a budget by number or name from a list.
// Returns the selected budget's ID, name, and description, or an error.
func SelectBudget(db *sql.DB, reader *bufio.Reader, budgetDefinitions []map[string]any) (int, string, string, error) {
	if len(budgetDefinitions) == 0 {
		return 0, "", "", fmt.Errorf("no budgets available")
	}

	fmt.Println("Available budget definitions:")
	for i, budget := range budgetDefinitions {
		fmt.Printf("%d. %s", i+1, budget["name"])
		if description, ok := budget["description"].(string); ok && description != "" {
			fmt.Printf(" - %s", description)
		}
		fmt.Println()
	}

	input, err := PromptInput(reader, "\nEnter budget name (or number from list): ")
	if err != nil {
		return 0, "", "", err
	}

	// Try number selection
	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(budgetDefinitions) {
			budget := budgetDefinitions[num-1]
			return budget["id"].(int), budget["name"].(string), budget["description"].(string), nil
		}
		return 0, "", "", fmt.Errorf("invalid budget number")
	}

	// Try name selection
	for _, budget := range budgetDefinitions {
		if strings.EqualFold(budget["name"].(string), input) {
			return budget["id"].(int), budget["name"].(string), budget["description"].(string), nil
		}
	}

	return 0, "", "", fmt.Errorf("budget '%s' not found", input)
}

// GetCurrentMonth returns the current month in YYYY-MM format
func GetCurrentMonth() string {
	return time.Now().Format("2006-01")
}

// ClearTerminal clears the terminal screen
func ClearTerminal() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// DisplayHeader displays the stylized FortiFi header with database info
func DisplayHeader(databasePath string) {
	fmt.Println("\033[36m" + strings.Repeat("=", 70))
	fmt.Println("  ███████╗ ██████╗ ██████╗ ████████╗██╗███████╗██╗")
	fmt.Println("  ██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝  ║██╔════╝  ║")
	fmt.Println("  █████╗  ██║   ██║██████╔╝   ██║   ██║█████╗  ██║")
	fmt.Println("  ██╔══╝  ██║   ██║██╔══██╗   ██║   ██║██╔══╝  ██║")
	fmt.Println("  ██║     ╚██████╔╝██║  ██║   ██║   ██║██║     ██║")
	fmt.Println("  ╚═╝      ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═╝╚═╝     ╚═╝")
	fmt.Println(strings.Repeat("=", 70) + "\033[0m")
	fmt.Printf("\033[33mDatabase: %s\033[0m\n", databasePath)
	fmt.Println()
}

// DisplayHeaderWithClear displays the header and clears the terminal first
func DisplayHeaderWithClear(databasePath string) {
	ClearTerminal()
	DisplayHeader(databasePath)
}

// WaitForEnter waits for user to press Enter to return to menu
func WaitForEnter(reader *bufio.Reader) {
	fmt.Print("\nPress Enter to return to menu...")
	reader.ReadString('\n')
}

// ConfirmAction prompts the user for confirmation and returns true if they confirm.
func ConfirmAction(reader *bufio.Reader, message string) (bool, error) {
	input, err := PromptInput(reader, message+" (yes/no): ")
	if err != nil {
		return false, err
	}

	input = strings.ToLower(strings.TrimSpace(input))
	return input == "yes" || input == "y", nil
}

// PrintBudgetReport prints the current month's budget report
func PrintBudgetReport(db *sql.DB) {
	currentMonth := GetCurrentMonth()
	budgetDefinitions, err := GetBudgetDefinitions(db)
	if err != nil {
		PrintError("retrieving budget definitions", err)
		return
	}
	if len(budgetDefinitions) == 0 {
		fmt.Println("No budgets found.")
		return
	}
	fmt.Printf("\nBudget Report for %s:\n", currentMonth)
	fmt.Println("=" + strings.Repeat("=", 60))
	for _, budget := range budgetDefinitions {
		budgetID := budget["id"].(int)
		budgetName := budget["name"].(string)
		budgetAmount, err := database.GetMonthlyBudgetInstance(db, budgetID, currentMonth)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("%s: No budget set for %s\n", budgetName, currentMonth)
				continue
			}
			PrintError(fmt.Sprintf("getting budget amount for %s", budgetName), err)
			continue
		}
		_, _, categoryIDs, err := database.GetBudgetDefinitionWithCategories(db, budgetID)
		if err != nil {
			PrintError(fmt.Sprintf("getting categories for budget %s", budgetName), err)
			continue
		}
		var categoryNames []string
		for _, categoryID := range categoryIDs {
			categoryName, err := database.GetCategoryNameByID(db, categoryID)
			if err != nil {
				PrintError(fmt.Sprintf("getting category name for ID %d", categoryID), err)
				continue
			}
			categoryNames = append(categoryNames, categoryName)
		}
		if len(categoryNames) == 0 {
			fmt.Printf("%s: No categories assigned (Budget: %s)\n", budgetName, FormatAmount(budgetAmount))
			continue
		}
		spentAmount, err := CalculateBudgetSpending(db, categoryIDs, currentMonth)
		if err != nil {
			PrintError(fmt.Sprintf("calculating spending for budget %s", budgetName), err)
			continue
		}
		remaining := budgetAmount + spentAmount
		var colorCode, percentColor string
		absSpent := abs(spentAmount)
		percent := 0.0
		if budgetAmount > 0 {
			percent = (absSpent / budgetAmount) * 100
		}
		if absSpent <= budgetAmount {
			colorCode = "\033[32m" // Green
			percentColor = "\033[32m"
		} else {
			colorCode = "\033[31m" // Red
			percentColor = "\033[31m"
		}
		fmt.Printf("%s%s\033[0m\n", colorCode, budgetName)
		fmt.Printf("  Categories: %s\n", strings.Join(categoryNames, ", "))
		fmt.Printf("  Spent: %s / %s  (%s%.0f%%%s)\n", FormatAmount(spentAmount), FormatAmount(budgetAmount), percentColor, percent, "\033[0m")
		fmt.Printf("  Remaining: %s\n", FormatAmount(remaining))
		fmt.Println()
	}
}

// CalculateBudgetSpending calculates total spending for given categories in a specific month
func CalculateBudgetSpending(db *sql.DB, categoryIDs []int, month string) (float64, error) {
	if len(categoryIDs) == 0 {
		return 0, nil
	}
	placeholders := strings.Repeat("?,", len(categoryIDs))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(`SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE category_id IN (%s) AND strftime('%%Y-%%m', transaction_date) = ?`, placeholders)
	args := make([]any, len(categoryIDs)+1)
	for i, id := range categoryIDs {
		args[i] = id
	}
	args[len(categoryIDs)] = month
	var totalSpent float64
	err := db.QueryRow(query, args...).Scan(&totalSpent)
	if err != nil {
		return 0, err
	}
	return totalSpent, nil
}

// PrintAccountBalances prints all accounts and their current balances
func PrintAccountBalances(db *sql.DB) {
	accounts, err := GetAvailableAccounts(db)
	if err != nil {
		PrintError("retrieving accounts", err)
		return
	}
	if len(accounts) == 0 {
		fmt.Println("No accounts found.")
		return
	}
	fmt.Println("Account Balances:")
	fmt.Println(strings.Repeat("-", 60))
	for _, acc := range accounts {
		// Get latest balance and snapshot time from account_snapshots
		var balance float64
		var snapshotTime string
		query := `SELECT balance, snapshot_time FROM account_snapshots WHERE account_id = ? ORDER BY snapshot_time DESC LIMIT 1`
		err := db.QueryRow(query, acc.Id).Scan(&balance, &snapshotTime)
		if err != nil && err != sql.ErrNoRows {
			PrintError("retrieving balance for account "+acc.Name, err)
			continue
		}
		if balance == 0.0 {
			continue
		}
		// Format date as YYYY-MM-DD
		dateStr := ""
		if len(snapshotTime) >= 10 {
			dateStr = snapshotTime[:10]
		}
		fmt.Printf("%-20s %15s   (Last updated: %s)\n", acc.Name, FormatAmount(balance), dateStr)
	}
	fmt.Println(strings.Repeat("-", 60))
}
