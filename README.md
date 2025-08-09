# Fortifi

## Overview
I wanted a good way to track all my expenses and see where my money is going without spending $10/month on a budgeting app. I also wanted something that would let me keep my financial information private and out of some third party database. In my free time over a few months, FortiFi was born.

FortiFi is personal finance management CLI tool built in Go that tracks transactions and budgets, categorizes expenses, and generates financial reports. The application uses SQLite for data storage and provides automated transaction categorization through keyword matching. Everything is stored locally on your machine in the database.db file(s). There are no calls to external APIs, no servers, no data sharing.

This program is designed to bring transactions from multiple places into one central spot so you can view all your spending and income together. It takes some setup to create rules to categorize your transactions but once most regular transactions are categorized, new imports will be mostly categorized and you'll only need to check for new or obscure transactions. 

I personally use it to consolidate information across two bank accounts, multiple credit cards, amazon orders, and venmo transactions. Anything you can get into a csv format can be ingested.

The privacy and free-ness does come with the inconvenience that you can't pull transactions from your bank automatically. You also can't store information in the cloud or access your budgets from your phone. You have to sit down at a computer, download the csv files from your bank/credit card, open the tool, and run an ingest command.

## Getting Started
The basic steps to get started are
- Build using the instructions below, or download the built zip file
- Download your csv file from your bank, credit card, venmo, etc and place it in the raw folder
- Add an import format for your csv file to the import_config.json. A description of each field can be found below
- Run the executable. Hit enter to move past the welcome message
- Run the ing (ingest) command and enter 'raw' to ingest every file in the raw folder. Alternatively, give it the path to a specific csv file
- Once ingested, use the brk (monthly breakdown) command to view a month that contains data you've imported
- For convenience, Uncategorized transactions are highlighted in blue
- Begin adding exact/includes keywords to categorize transactions
  - These keywords will be applied to the current database and all future imported data
- Play around with the other commands, run hlp to see what each does
- Enjoy tracking finances for free!


## Building and Running

### Development
- Install Go: https://go.dev/doc/install
- Run the main.go file under cmd
  - `go run .\cmd\fortifi-cli\main.go`
- Select commands by typing their 3 letter tag
- Use hlp to get a description of all the commands

### Building for Distribution
To create a deployable package:

**Windows:**
```bash
.\build.bat
```

**Linux/macOS:**
```bash
./build.sh
```

This creates a `builds/` directory containing:
- `FortiFi.exe` (Windows) or `FortiFi` (Linux/macOS) - The executable
- `import_config.json` - A configuration template for CSV import formats
- `raw/` - The directory for placing CSV files to import
- `README.md` - Some instructions

The `builds/` directory is ready to distribute

## Setting up Import Config
The build directory comes with an 'import_config.json' in order to import csv files you must add import formats to this config. The examples should be pretty clear but the following are items that could use more explanation

### Configuration Options

#### `identifier` (string)
- This field is used to automatically identify csv files in the ./raw directory
- Set it to some keyword that always appears in your csv file names
  - For example "chase-checking" if your csv is named Chase-Checking-some-date.csv
- Note: the identifier ignores case. Enter it as all lowercase and it will check your filename converted to lowercase

#### `account_name` (string)
- The unique name of your account as it is tracked in the database


#### `date_format` (string)
This field uses Go's time formatting syntax, which is based on a specific reference date: *Mon Jan 2 15:04:05 MST 2006*.

**Common date formats:**
- *"2006-01-02"* - ISO format (YYYY-MM-DD) - Example: 2024-01-15
- *"01/02/2006"* - US format (MM/DD/YYYY) - Example: 01/15/2024
- *"02/01/2006"* - European format (DD/MM/YYYY) - Example: 15/01/2024
- *"2006-01-02 15:04:05"* - With time (YYYY-MM-DD HH:MM:SS)
- *"Jan 02, 2006"* - Text month format - Example: Jan 15, 2024
- *"02-Jan-2006"* - European text format - Example: 15-Jan-2024

#### `column_mapping`
Maps CSV column names to Fortifi transaction fields. Set each field to the header of the column with that data. For example 'date' might need to map to "Date", "Transaction Date", "Date Posted", etc.
- `date` - Transaction date column
- `description` - Transaction description/memo column
- `amount` - Transaction amount column
- `balance` - Savings and Checking accounts usually have a balance column (optional, can be omitted if track_balance is false)

#### `amount_multiplier` (number, optional)
- Multiplier to apply to amount values
- Default: 1 (no multiplication)
- Example: `0.01` converts cents to dollars, `100` converts dollars to cents
- Example: `-1.0` negates values. Useful when importing a credit card export that tracks your credit card balance as a positive number but you consider those transactions as expenses

#### `track_balance` (bool, optional)
- Indicate if this import type has balance information you'd like to track
- Default: false
- Example: You might set to true on your primary savings account to see how your savings do over time but not on a credit card that doesn't contain any balance information

#### `blacklist_exact` (list of strings, optional)
- Exact transaction descriptions to ignore
- Default: none
- Example: If you're tracking multiple bank accounts and don't want money transferred between them to show up as transactions, you can black list  the description that shows up on each of those transactions ie: "ACH Transfer to Some Bank - 1234" and "ACH Transfer from Some Bank - 1234"

#### `blacklist_contains` (list of strings, optional)
- Ignore all transactions with descriptions that contain any of the listed strings
- Default: none
- Example: If you don't want to track any transactions from a specific grocery store (Walmart) but that store adds random numbers or hashes to their transaction descriptions you could list 'walmart' here to ignore all of them

Note: Similar exact and contains rules can be set inside the CLI for transaction categorization

#### `special_rules` (list of strings, optional)
- A list of special rules to check for each transaction. These currently only support an exact description paired with an exact amount going to a specific category
- More special rule formats could be added in the future
- Default: none 
- description_exact: The exact string for the description
- amount_exact: The exact amount the transaction must have
- force_category: The name of the category to put the transaction in
- Example: If you have reoccurring expenses like rent that are paid through a platform like Venmo, you can set any Venmo payment that is the exact rent amount to be placed in the rent category