.PHONY: all build clean run-bot run-tui tidy

# Default target builds both binaries
all: build

# Build both targets
build: nomi-sec-bot cve-tui

nomi-sec-bot: main.go checker.go commands.go poc/github.go poc/storage.go
	go build -o nomi-sec-bot .

cve-tui: cmd/tui/main.go poc/github.go poc/storage.go
	go build -o cve-tui cmd/tui/main.go

# Clean up built binaries
clean:
	rm -f nomi-sec-bot cve-tui

# Run bot
run-bot: nomi-sec-bot
	./nomi-sec-bot

# Run TUI
run-tui: cve-tui
	./cve-tui

# Tidy up Go modules
tidy:
	go mod tidy
