package handlers

import (
	"bufio"
	"database/sql"
	"fmt"

	"github.com/HadeZForge/FortiFi/internal/cli/utils"
	_ "github.com/mattn/go-sqlite3"
)

func UpdateCategoryExactCLI(db *sql.DB, reader *bufio.Reader) {
	description, err := utils.PromptInput(reader, "Enter exact description to match: ")
	if err != nil {
		utils.PrintError("reading description", err)
		return
	}

	// Get available categories for the new transaction
	categories, err := utils.GetAvailableCategories(db)
	if err != nil {
		utils.PrintError("retrieving categories", err)
		return
	}

	categoryID, categoryName, err := utils.SelectCategory(db, reader, categories, false)
	if err != nil {
		utils.PrintError("selecting category", err)
		return
	}

	// Update transactions with exact description match
	result, err := db.Exec(`UPDATE transactions SET category_id = ? WHERE description = ?`, categoryID, description)
	if err != nil {
		utils.PrintError("updating transactions", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Updated %d transactions with description '%s' to category '%s'\n", rowsAffected, description, categoryName)

	// Store the exact keyword for future imports
	_, err = db.Exec(`INSERT OR REPLACE INTO exact_keywords (keyword, category_id) VALUES (?, ?)`, description, categoryID)
	if err != nil {
		utils.PrintWarning("saving exact keyword", err)
	} else {
		fmt.Printf("Saved exact keyword rule for future imports\n")
	}
}

func UpdateCategoryIncludesCLI(db *sql.DB, reader *bufio.Reader) {
	keyword, err := utils.PromptInput(reader, "Enter keyword/string to search for in descriptions: ")
	if err != nil {
		utils.PrintError("reading keyword", err)
		return
	}

	// Get available categories for the new transaction
	categories, err := utils.GetAvailableCategories(db)
	if err != nil {
		utils.PrintError("retrieving categories", err)
		return
	}

	categoryID, categoryName, err := utils.SelectCategory(db, reader, categories, false)
	if err != nil {
		utils.PrintError("selecting category", err)
		return
	}

	// Update transactions containing the keyword
	result, err := db.Exec(`UPDATE transactions SET category_id = ? WHERE description LIKE ?`, categoryID, "%"+keyword+"%")
	if err != nil {
		utils.PrintError("updating transactions", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Updated %d transactions containing '%s' to category '%s'\n", rowsAffected, keyword, categoryName)

	// Store the includes keyword for future imports
	_, err = db.Exec(`INSERT OR REPLACE INTO includes_keywords (keyword, category_id) VALUES (?, ?)`, keyword, categoryID)
	if err != nil {
		utils.PrintWarning("saving includes keyword", err)
	} else {
		fmt.Printf("Saved includes keyword rule for future imports\n")
	}
}
