package handlers

import (
	"bufio"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	"github.com/HadeZForge/FortiFi/internal/database"
	txnUtils "github.com/HadeZForge/FortiFi/internal/transaction/utils"
	"github.com/HadeZForge/FortiFi/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

type AccountBalanceInfo struct {
	ID   int
	Name string
}

type MonthlyBalance struct {
	Month           string
	StartingBalance float64
	EndingBalance   float64
	Change          float64
}

func AccountBalanceHistoryCLI(db *sql.DB, reader *bufio.Reader) {
	// Get accounts with balance tracking enabled
	balanceAccounts, err := getAccountsWithBalanceTracking(db)
	if err != nil {
		utils.PrintError("retrieving accounts with balance tracking", err)
		return
	}

	if len(balanceAccounts) == 0 {
		fmt.Println("No accounts found with balance tracking enabled.")
		fmt.Println("To enable balance tracking, set 'track_balance': true in your import configuration.")
		return
	}

	// Display available accounts
	fmt.Println("Accounts with balance tracking enabled:")
	for i, account := range balanceAccounts {
		fmt.Printf("%d. %s\n", i+1, account.Name)
	}

	// Let user select an account
	accountChoice, err := utils.PromptInput(reader, "\nSelect account number: ")
	if err != nil {
		utils.PrintError("reading account choice", err)
		return
	}

	accountIndex, err := strconv.Atoi(accountChoice)
	if err != nil || accountIndex < 1 || accountIndex > len(balanceAccounts) {
		fmt.Println("Invalid account selection.")
		return
	}

	selectedAccount := balanceAccounts[accountIndex-1]

	// Get account balance history
	history, err := database.GetAccountHistory(db, selectedAccount.ID)
	if err != nil {
		utils.PrintError("retrieving account history", err)
		return
	}

	if len(history) == 0 {
		fmt.Printf("\nNo balance history found for account '%s'.\n", selectedAccount.Name)
		return
	}

	// Convert to monthly averages
	monthlyBalances := convertToMonthlyAverages(history)

	if len(monthlyBalances) == 0 {
		fmt.Printf("\nNo monthly data available for account '%s'.\n", selectedAccount.Name)
		return
	}

	// Display account balance history
	fmt.Printf("\n=== Account Balance History: %s ===\n", selectedAccount.Name)
	fmt.Println()

	// Display monthly summary table
	displayMonthlyTable(monthlyBalances)

	// Display summary
	fmt.Println()
	displayBalanceSummary(monthlyBalances)
}

// getAccountsWithBalanceTracking returns accounts that have balance tracking enabled in the config
func getAccountsWithBalanceTracking(db *sql.DB) ([]AccountBalanceInfo, error) {
	// Load import configuration
	config, err := txnUtils.LoadImportConfig("")
	if err != nil {
		return nil, fmt.Errorf("failed to load import config: %w", err)
	}

	var balanceAccounts []AccountBalanceInfo

	// Find formats with track_balance enabled
	for _, format := range config.ImportFormats {
		if format.TrackBalance {
			// Get account ID for this account name
			accountID, err := database.GetAccountID(db, format.AccountName)
			if err != nil {
				// Skip accounts that don't exist in the database yet
				continue
			}

			balanceAccounts = append(balanceAccounts, AccountBalanceInfo{
				ID:   accountID,
				Name: format.AccountName,
			})
		}
	}

	return balanceAccounts, nil
}

// convertToMonthlyAverages converts daily snapshots to monthly starting/ending balances
func convertToMonthlyAverages(history []types.AccountSnapshot) []MonthlyBalance {
	if len(history) == 0 {
		return []MonthlyBalance{}
	}

	// Group snapshots by month
	monthlyData := make(map[string][]types.AccountSnapshot)
	for _, snapshot := range history {
		month := snapshot.SnapshotTime.Format("2006-01")
		monthlyData[month] = append(monthlyData[month], snapshot)
	}

	// Calculate monthly starting/ending balances and sort by month
	var monthlyBalances []MonthlyBalance
	for month, snapshots := range monthlyData {
		// Sort snapshots within the month by date
		sort.Slice(snapshots, func(i, j int) bool {
			return snapshots[i].SnapshotTime.Before(snapshots[j].SnapshotTime)
		})

		startingBalance := snapshots[0].Balance
		endingBalance := snapshots[len(snapshots)-1].Balance

		monthlyBalances = append(monthlyBalances, MonthlyBalance{
			Month:           month,
			StartingBalance: startingBalance,
			EndingBalance:   endingBalance,
			Change:          endingBalance - startingBalance, // Change within the month
		})
	}

	// Sort by month
	sort.Slice(monthlyBalances, func(i, j int) bool {
		return monthlyBalances[i].Month < monthlyBalances[j].Month
	})

	// Calculate month-over-month changes (ending balance vs previous month's ending balance)
	for i := 1; i < len(monthlyBalances); i++ {
		monthlyBalances[i].Change = monthlyBalances[i].EndingBalance - monthlyBalances[i-1].EndingBalance
	}

	return monthlyBalances
}

// displayMonthlyTable displays the monthly balance data in a table format
func displayMonthlyTable(monthlyBalances []MonthlyBalance) {
	header := []string{"Month", "Starting Balance", "Ending Balance", "Change"}
	widths := []int{10, 15, 15, 15}
	fmt.Println(utils.FormatRow(header, widths))
	fmt.Println(strings.Repeat("-", utils.Sum(widths)+9))

	for i, balance := range monthlyBalances {
		var changeStr string
		if i == 0 {
			// First month - no previous month to compare against
			changeStr = utils.PadAnsi("---", widths[3])
		} else {
			// Month-over-month change
			if balance.Change > 0 {
				changeStr = utils.Green + utils.FormatAmount(balance.Change) + utils.Reset
			} else if balance.Change < 0 {
				changeStr = utils.Red + utils.FormatAmount(balance.Change) + utils.Reset
			} else {
				changeStr = utils.Blue + "$0.00" + utils.Reset
			}
		}

		row := []string{
			balance.Month,
			utils.PadAnsi(utils.FormatAmount(balance.StartingBalance), widths[1]),
			utils.PadAnsi(utils.FormatAmount(balance.EndingBalance), widths[2]),
			utils.PadAnsi(changeStr, widths[3]),
		}
		fmt.Println(utils.FormatRow(row, widths))
	}
}

// displayBalanceSummary shows summary statistics
func displayBalanceSummary(monthlyBalances []MonthlyBalance) {
	fmt.Println("=== Summary ===")
	fmt.Printf("Total months tracked: %d\n", len(monthlyBalances))

	if len(monthlyBalances) > 1 {
		firstBalance := monthlyBalances[0].StartingBalance
		lastBalance := monthlyBalances[len(monthlyBalances)-1].EndingBalance
		totalChange := lastBalance - firstBalance

		fmt.Printf("Starting balance: %s\n", utils.FormatAmount(firstBalance))
		fmt.Printf("Current balance: %s\n", utils.FormatAmount(lastBalance))
		fmt.Printf("Total change: %s\n", utils.FormatAmount(totalChange))

		// Calculate average monthly change
		avgMonthlyChange := totalChange / float64(len(monthlyBalances)-1)
		fmt.Printf("Average monthly change: %s\n", utils.FormatAmount(avgMonthlyChange))
	}
}
