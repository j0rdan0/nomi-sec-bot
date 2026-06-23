# Nomi-Sec CVE PoC Bot

A Telegram bot that monitors the [nomi-sec/PoC-in-GitHub](https://github.com/nomi-sec/PoC-in-GitHub) repository for new Proof of Concept (PoC) exploits and allows users to query CVEs by year.

## Features

- **Automated Monitoring:** Checks for new commits in the PoC repository every 24 hours and sends notifications to a specified Telegram chat.
- **Smart Startup & Catch-up:** On startup, the bot notifies the user of the last date it sent a PoC notification and aligns with the repository head without spamming old notifications. The user can catch up on missed PoCs at any time using `/catchup`.
- **Detailed Notifications:** Includes CVE ID, repository description, and direct links to the PoC repositories.
- **On-Demand Queries:** Use the `/cve` command to fetch the latest PoCs for any given year.
- **Environment Driven:** Configuration is handled via `.env` file for security.

## Commands

- `/cve <year> <count>` - Fetches the top `count` PoCs for the specified `year` (e.g. `/cve 2024 5`).
- `/cve <CVE-ID>` - Fetches the PoCs for a specific CVE (e.g. `/cve CVE-2002-1614`).
- `/catchup` - Catch up on missed PoCs that were added while the bot was offline.

## Setup

### Prerequisites

- Go 1.26 or higher
- A Telegram Bot Token (from [@BotFather](https://t.me/BotFather))
- A Telegram Chat ID (where the bot will send background updates)
- (Optional) A GitHub Personal Access Token to avoid API rate limiting

### Installation

1. Clone the repository:
   ```bash
   git clone <your-repo-url>
   cd nomi-sec-bot
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Create a `.env` file in the root directory:
   ```env
   TELEGRAM_BOT_TOKEN=your_bot_token_here
   TELEGRAM_CHAT_ID=your_chat_id_here
   GITHUB_TOKEN=your_github_token_here (optional)
   ```

4. Build both applications using the Makefile:
   ```bash
   make
   ```
   *Alternative commands:*
   - Build only Telegram Bot: `make nomi-sec-bot`
   - Build only TUI App: `make cve-tui`
   - Clean built binaries: `make clean`
   - Run TUI directly: `make run-tui`
   - Run Bot directly: `make run-bot`

## Running the Bot

Start the bot by running the executable:
```bash
./nomi-sec-bot
```

The bot will initialize, register its command menu, and start the background checker immediately.

## Running the TUI Console App

The project now includes a rich, high-fidelity terminal user interface (TUI) console app built with the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

Run the TUI:
```bash
./cve-tui
```
or run it directly:
```bash
go run cmd/tui/main.go
```

### TUI Features

- **Latest 5 PoCs:** The main page displays the 5 most recent PoCs fetched dynamically from recent commits on the repository.
- **On-Demand Query:** Search by Year and Count (e.g., `2024 5`) or directly by specific CVE ID (e.g., `CVE-2024-1234`).

### Keyboard Shortcuts

- **Tab / Shift+Tab:** Switch between the "Latest 5 PoCs" and "Query CVE" tabs
- **1, 2:** Quick jump to Tab 1 (Latest PoCs) or Tab 2 (Query CVE)
- **Up / Down Arrow:** Navigate / select PoC entries (in "Query CVE" tab, press `Down` to blur the search box and focus the results list; press `Up` on the first result to refocus the search box)
- **Esc / /:** Focus search input (Query CVE Tab)
- **R:** Refresh the latest 5 PoCs (Latest PoCs Tab)
- **Enter:** Open selected PoC in browser (Latest PoCs Tab, or Query CVE Tab when results are focused); submit search (Query CVE Tab when search input is focused)
- **O:** Open selected PoC in browser (both tabs)
- **Q / Ctrl+C:** Quit the application

## License

This project is open-source and available under the MIT License.
