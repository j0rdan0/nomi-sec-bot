# Nomi-Sec CVE PoC Bot

A Telegram bot that monitors the [nomi-sec/PoC-in-GitHub](https://github.com/nomi-sec/PoC-in-GitHub) repository for new Proof of Concept (PoC) exploits and allows users to query CVEs by year.

## Features

- **Automated Monitoring:** Checks for new commits in the PoC repository every 24 hours and sends notifications to a specified Telegram chat.
- **Detailed Notifications:** Includes CVE ID, repository description, and direct links to the PoC repositories.
- **On-Demand Queries:** Use the `/cve` command to fetch the latest PoCs for any given year.
- **Environment Driven:** Configuration is handled via `.env` file for security.

## Commands

- `/cve <year> <count>` - Fetches the top `count` PoCs for the specified `year`.
  - Example: `/cve 2024 5`

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

4. Build the application:
   ```bash
   go build -o nomi-sec-bot .
   ```

## Running the Bot

Start the bot by running the executable:
```bash
./nomi-sec-bot
```

The bot will initialize, register its command menu, and start the background checker immediately.

## License

This project is open-source and available under the MIT License.
