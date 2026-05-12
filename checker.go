package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func StartChecker(bot *telego.Bot, chatIDStr string) {
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_CHAT_ID: %v", err)
	}

	// 24 hour interval as requested by user
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once at start
	checkUpdates(bot, chatID)

	for range ticker.C {
		checkUpdates(bot, chatID)
	}
}

func checkUpdates(bot *telego.Bot, chatID int64) {
	ctx := context.Background()
	log.Println("Checking for updates...")
	state, err := LoadState()
	if err != nil {
		log.Printf("Error loading state: %v", err)
		return
	}

	commits, err := GetRecentCommits()
	if err != nil {
		log.Printf("Error getting recent commits: %v", err)
		return
	}

	if len(commits) == 0 {
		return
	}

	// If it's the first run, just save the latest commit and return
	if state.LastCommitSHA == "" {
		state.LastCommitSHA = commits[0].SHA
		if err := SaveState(state); err != nil {
			log.Printf("Error saving state: %v", err)
		}
		return
	}

	var newCommits []Commit
	for _, c := range commits {
		if c.SHA == state.LastCommitSHA {
			break
		}
		newCommits = append(newCommits, c)
	}

	if len(newCommits) == 0 {
		log.Println("No new commits found.")
		return
	}

	log.Printf("Processing %d new commits...", len(newCommits))

	// Process new commits from oldest to newest
	for i := len(newCommits) - 1; i >= 0; i-- {
		commit := newCommits[i]
		log.Printf("Checking commit %s...", commit.SHA)
		files, err := GetCommitChangedFiles(commit.SHA)
		if err != nil {
			log.Printf("Error getting changed files for commit %s: %v", commit.SHA, err)
			continue
		}

		for _, file := range files {
			log.Printf("Processing file: %s", file)
			infos, err := FetchPoCInfo(file)
			if err != nil {
				log.Printf("Error fetching PoC info for %s: %v", file, err)
				continue
			}

			for _, info := range infos {
				log.Printf("Sending notification for %s", info.Name)
				desc := info.Description
				if desc == "" {
					desc = "_No description provided_"
				}

				msg := fmt.Sprintf("🚨 *New PoC for %s*\n\n📝 *Description:*\n%s\n\n🔗 [PoC Repository](%s)", 
					info.Name, desc, info.RepositoryURL)
				_, err = bot.SendMessage(ctx, tu.Message(
					tu.ID(chatID),
					msg,
				).WithParseMode(telego.ModeMarkdown))
				if err != nil {
					log.Printf("Error sending telegram message: %v", err)
				}
			}
		}
	}

	state.LastCommitSHA = commits[0].SHA
	if err := SaveState(state); err != nil {
		log.Printf("Error saving state: %v", err)
	}
}
