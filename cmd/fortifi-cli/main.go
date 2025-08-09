package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/handlers"
	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

const configFile = ".fortifi"

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		utils.PrintError("loading config", err)
		return
	}

	// Open database using config
	db, err := sql.Open("sqlite3", config.DatabasePath)
	if err != nil {
		utils.PrintError("connecting to database", err)
		return
	}
	defer db.Close()

	database.InitTables(db)

	// Check and create missing monthly budget instances
	if err := database.CheckAndCreateMissingMonthlyInstances(db); err != nil {
		utils.PrintWarning("checking for missing monthly budget instances", err)
	}

	reader := bufio.NewReader(os.Stdin)

	// Define available commands
	commands := []types.Command{
		{
			Tag:         "hlp",
			Name:        "Help",
			Description: "Show descriptions of all available commands",
			Handler:     nil,
		},
		{
			Tag:         "brk",
			Name:        "Report 	- Month Breakdown",
			Description: "Generate a detailed breakdown report for a specific month including all transactions for that month",
			Handler:     handlers.GenerateMonthReportCLI,
		},
		{
			Tag:         "sts",
			Name:        "Report 	- Generate Stats",
			Description: "Generate statistics for a specific year or all data",
			Handler:     handlers.GenerateStatsCLI,
		},
		{
			Tag:         "tim",
			Name:        "Report 	- Category Timeline",
			Description: "Show monthly spending timeline for a specific category",
			Handler:     handlers.CategoryTimelineCLI,
		},
		{
			Tag:         "bst",
			Name:        "Report 	- Budget Status",
			Description: "Show current budget status with spending vs budget amounts",
			Handler:     handlers.BudgetReportCLI,
		},
		{
			Tag:         "bhi",
			Name:        "Report 	- Budget History",
			Description: "Show monthly budget performance history for a selected budget",
			Handler:     handlers.BudgetHistoryCLI,
		},
		{
			Tag:         "abh",
			Name:        "Report 	- Account Balance History",
			Description: "Show account balance history over time for accounts with balance tracking",
			Handler:     handlers.AccountBalanceHistoryCLI,
		},
		{
			Tag:         "ade",
			Name:        "Category 	- Add Exact Rule",
			Description: "Add a new rule for transactions with exact description match. This will be used for future imports and updates the current database.",
			Handler:     handlers.UpdateCategoryExactCLI,
		},
		{
			Tag:         "adi",
			Name:        "Category 	- Add Includes Rule",
			Description: "Add a new rule for transactions containing a keyword. This will be used for future imports and updates the current database.",
			Handler:     handlers.UpdateCategoryIncludesCLI,
		},
		{
			Tag:         "dca",
			Name:        "Category 	- Delete Category",
			Description: "Delete a category and handle its transactions (delete or move to uncategorized)",
			Handler:     handlers.DeleteCategoryCLI,
		},
		{
			Tag:         "cbu",
			Name:        "Budget 	- Create Budget",
			Description: "Create a new budget with categories and set amount for current month",
			Handler:     handlers.CreateBudgetCLI,
		},
		{
			Tag:         "dbu",
			Name:        "Budget 	- Delete Budget",
			Description: "Delete a budget definition and/or its monthly instances",
			Handler:     handlers.DeleteBudgetCLI,
		},
		{
			Tag:         "ubu",
			Name:        "Budget 	- Update Amount",
			Description: "Update the budget amount for the current month",
			Handler:     handlers.UpdateBudgetAmountCLI,
		},
		{
			Tag:         "cbc",
			Name:        "Budget 	- Change Categories",
			Description: "Add or remove categories from a budget definition",
			Handler:     handlers.ChangeBudgetCategoriesCLI,
		},
		{
			Tag:         "add",
			Name:        "Transaction 	- Add Transaction",
			Description: "Add a new transaction to the database",
			Handler:     handlers.AddTransactionCLI,
		},
		{
			Tag:         "chg",
			Name:        "Transaction 	- Change Category",
			Description: "Change a specific transaction's category by ID",
			Handler:     handlers.ChangeTransactionCategoryCLI,
		},
		{
			Tag:         "del",
			Name:        "Transaction 	- Delete Transaction(s)",
			Description: "Delete transaction(s) by ID or category",
			Handler:     handlers.DeleteTransactionCLI,
		},
		{
			Tag:         "spl",
			Name:        "Transaction 	- Split Transaction",
			Description: "Split a transaction into two parts with different categories",
			Handler:     handlers.SplitTransactionCLI,
		},
		{
			Tag:         "ing",
			Name:        "Data 		- Ingest CSV",
			Description: "Import CSV files (single file or all files in ./raw/ directory)",
			Handler:     handlers.IngestDataCLI,
		},
		{
			Tag:         "cdb",
			Name:        "Data 		- Change Database",
			Description: "Change the database file path",
			Handler:     nil, // Special case for change database
		},
		{
			Tag:         "ext",
			Name:        "Exit",
			Description: "Exit the application",
			Handler:     nil,
		},
	}

	// Set the help handler to use the commands list
	commands[0].Handler = func(db *sql.DB, reader *bufio.Reader) { helpCLI(commands) }

	// Create a map for quick lookup
	commandMap := make(map[string]types.Command)
	for _, cmd := range commands {
		commandMap[cmd.Tag] = cmd
	}

	// Show welcome message and financial overview
	utils.DisplayHeaderWithClear(config.DatabasePath)
	fmt.Println("Welcome to FortiFi!")
	fmt.Printf("Version: %s\n", types.Version)
	fmt.Println("Your personal finance command center")
	fmt.Println()
	utils.PrintAccountBalances(db)
	utils.PrintBudgetReport(db)
	utils.WaitForEnter(reader)

	for {
		utils.DisplayHeaderWithClear(config.DatabasePath)
		utils.DisplayCommands(commands)

		fmt.Print("\nEnter command: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if cmd, exists := commandMap[input]; exists {
			if cmd.Tag == "ext" {
				fmt.Println("Exiting...")
				return
			}
			if cmd.Tag == "cdb" {
				utils.DisplayHeaderWithClear(config.DatabasePath)
				changeDatabaseCLI(&db, reader, config)
			} else if cmd.Handler != nil {
				utils.DisplayHeaderWithClear(config.DatabasePath)
				cmd.Handler(db, reader)
				utils.WaitForEnter(reader)
			}
		} else {
			utils.PrintError("invalid command", fmt.Errorf("'%s' is not a valid command", input))
			utils.WaitForEnter(reader)
		}
	}
}

func loadConfig() (*types.Config, error) {
	config := &types.Config{
		DatabasePath: "./FortiFi.db", // Default database path
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return config, nil // Return default config if file doesn't exist
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func saveConfig(config *types.Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func changeDatabaseCLI(currentDB **sql.DB, reader *bufio.Reader, config *types.Config) {
	dbPath, err := utils.PromptInput(reader, "Enter new database file path: ")
	if err != nil {
		utils.PrintError("reading database path", err)
		utils.WaitForEnter(reader)
		return
	}

	if dbPath == "" {
		fmt.Println("Database path cannot be empty.")
		utils.WaitForEnter(reader)
		return
	}

	if !strings.HasSuffix(dbPath, ".db") {
		dbPath = dbPath + ".db"
	}

	// Close current database
	if *currentDB != nil {
		(*currentDB).Close()
	}

	// Open new database
	newDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		utils.PrintError("connecting to new database", err)
		// Try to reopen the old database
		config, _ := loadConfig()
		*currentDB, _ = sql.Open("sqlite3", config.DatabasePath)
		utils.WaitForEnter(reader)
		return
	}

	// Test the connection
	if err := newDB.Ping(); err != nil {
		utils.PrintError("pinging new database", err)
		newDB.Close()
		// Try to reopen the old database
		config, _ := loadConfig()
		*currentDB, _ = sql.Open("sqlite3", config.DatabasePath)
		utils.WaitForEnter(reader)
		return
	}

	// Initialize tables in the new database
	database.InitTables(newDB)

	// Update the database reference
	*currentDB = newDB

	// Save the new database path to config
	config.DatabasePath = dbPath
	if err := saveConfig(config); err != nil {
		utils.PrintWarning("saving config", err)
	}

	fmt.Printf("Successfully switched to database: %s\n", dbPath)
	utils.WaitForEnter(reader)
}

func helpCLI(commands []types.Command) {
	fmt.Println("Available Commands:")
	fmt.Println("===================")

	// Display all commands except help and exit
	for _, cmd := range commands {
		if cmd.Tag != "hlp" && cmd.Tag != "ext" {
			fmt.Printf("(%s) %s\n", cmd.Tag, cmd.Name)
			fmt.Printf("   %s\n\n", cmd.Description)
		}
	}
}
