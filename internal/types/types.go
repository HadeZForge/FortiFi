package types

import (
	"bufio"
	"database/sql"
	"time"
)

type AccountSnapshot struct {
	ID           int       `json:"id"`
	SnapshotTime time.Time `json:"snapshot_time"`
	Balance      float64   `json:"balance"`
	AccountID    int       `json:"account_id"`
}

type Command struct {
	Tag         string
	Name        string
	Description string
	Handler     func(*sql.DB, *bufio.Reader)
}

type CategoryInfo struct {
	Name             string
	TransactionCount int
	Id               int
}

type TimelineEntry struct {
	Month string
	Total float64
}

// ImportConfig represents the entire configuration file
type ImportConfig struct {
	ImportFormats []ImportFormat `json:"import_formats"`
}

// ImportFormat defines how to parse a specific CSV format
type ImportFormat struct {
	Identifier        string        `json:"identifier"`
	AccountName       string        `json:"account_name"`
	ColumnMapping     ColumnMapping `json:"column_mapping"`
	DateFormat        string        `json:"date_format"`
	AmountMultiplier  float64       `json:"amount_multiplier"`
	TrackBalance      bool          `json:"track_balance"`
	BlacklistExact    []string      `json:"blacklist_exact"`
	BlacklistContains []string      `json:"blacklist_contains"`
	SpecialRules      []SpecialRule `json:"special_rules"`
}

// ColumnMapping defines which CSV columns map to which transaction fields
type ColumnMapping struct {
	Date        string `json:"date"`
	Description string `json:"description"`
	Amount      string `json:"amount"`
	Balance     string `json:"balance,omitempty"`
}

// SpecialRule defines special processing rules for specific transactions
type SpecialRule struct {
	DescriptionExact string   `json:"description_exact"`
	AmountExact      *float64 `json:"amount_exact,omitempty"`
	ForceCategory    string   `json:"force_category"`
}

// GenericTransaction represents a parsed transaction before database insertion
type GenericTransaction struct {
	Date          time.Time
	Description   string
	Amount        float64
	Balance       *float64
	DailySequence int
}

type TableTransaction struct {
	Id          string
	AccountID   int
	CategoryID  int
	Amount      float64
	Date        string
	Description string
}

// Main project .fortifi config file struct
type Config struct {
	DatabasePath string `json:"database_path"`
}
