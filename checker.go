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
	checkUpdates(bot, chatID, true)

	for range ticker.C {
		checkUpdates(bot, chatID, false)
	}
}

func checkUpdates(bot *telego.Bot, chatID int64, isStartup bool) {
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

	// If it's startup, save the latest commit as StartupCommitSHA and return (skip notifications)
	if isStartup {
		log.Printf("Startup check: setting startup commit SHA to %s and skipping notifications", commits[0].SHA)
		state.StartupCommitSHA = commits[0].SHA
		// If last commit SHA is empty (first run), initialize it too
		if state.LastCommitSHA == "" {
			state.LastCommitSHA = commits[0].SHA
		}
		if err := SaveState(state); err != nil {
			log.Printf("Error saving state: %v", err)
		}

		// Inform user about startup and last notification time
		if state.LastNotificationTime.IsZero() && state.LastCommitSHA != "" {
			t, err := GetCommitDate(state.LastCommitSHA)
			if err == nil {
				state.LastNotificationTime = t
				if err := SaveState(state); err != nil {
					log.Printf("Error saving state: %v", err)
				}
			}
		}

		var lastTimeStr string
		if state.LastNotificationTime.IsZero() {
			lastTimeStr = "*Never*"
		} else {
			lastTimeStr = fmt.Sprintf("`%s`", state.LastNotificationTime.Format("2006-01-02 15:04:05 MST"))
		}
		startupMsg := fmt.Sprintf("*Bot has started!*\n\n*Last PoC notification received on:* %s", lastTimeStr)
		_, err = bot.SendMessage(ctx, tu.Message(
			tu.ID(chatID),
			startupMsg,
		).WithParseMode(telego.ModeMarkdown))
		if err != nil {
			log.Printf("Error sending startup message: %v", err)
		}

		return
	}

	// Determine the base SHA to compare against.
	// If caught up or startup SHA is empty, we start from LastCommitSHA.
	// Otherwise, we start from StartupCommitSHA to avoid spamming the backlog.
	baseSHA := state.StartupCommitSHA
	if state.LastCommitSHA == state.StartupCommitSHA || state.StartupCommitSHA == "" {
		baseSHA = state.LastCommitSHA
	}

	if baseSHA == "" {
		state.LastCommitSHA = commits[0].SHA
		state.StartupCommitSHA = commits[0].SHA
		if err := SaveState(state); err != nil {
			log.Printf("Error saving state: %v", err)
		}
		return
	}

	var newCommits []Commit
	for _, c := range commits {
		if c.SHA == baseSHA {
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

				msg := fmt.Sprintf("*New PoC for %s*\n\n*Description:*\n%s\n\n[PoC Repository](%s)", 
					info.Name, desc, info.RepositoryURL)
				_, err = bot.SendMessage(ctx, tu.Message(
					tu.ID(chatID),
					msg,
				).WithParseMode(telego.ModeMarkdown))
				if err != nil {
					log.Printf("Error sending telegram message: %v", err)
				} else {
					state.LastNotificationTime = time.Now()
				}
			}
		}
	}

	// Update state
	if state.LastCommitSHA == state.StartupCommitSHA {
		state.LastCommitSHA = commits[0].SHA
		state.StartupCommitSHA = commits[0].SHA
	} else {
		state.StartupCommitSHA = commits[0].SHA
	}

	if err := SaveState(state); err != nil {
		log.Printf("Error saving state: %v", err)
	}
}

func CatchUp(bot *telego.Bot, chatID int64, forceCount int) {
	ctx := context.Background()
	state, err := LoadState()
	if err != nil {
		log.Printf("Error loading state: %v", err)
		_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), "Error loading state."))
		return
	}

	commits, err := GetRecentCommits()
	if err != nil {
		log.Printf("Error getting recent commits: %v", err)
		_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), "Error fetching recent commits from GitHub."))
		return
	}

	if len(commits) == 0 {
		_, _ = bot.SendMessage(ctx, tu.Message(tu.ID(chatID), "No commits found on GitHub."))
		return
	}

	var newCommits []Commit
	found := false

	if forceCount > 0 {
		if forceCount > len(commits) {
			forceCount = len(commits)
		}
		newCommits = commits[:forceCount]
		found = true
	} else {
		if state.LastCommitSHA == "" || state.StartupCommitSHA == "" || state.LastCommitSHA == state.StartupCommitSHA {
			_, _ = bot.SendMessage(ctx, tu.Message(
				tu.ID(chatID),
				"You are already caught up! No missed PoCs since the last session.\n\n*Tip:* If you want to force catch up on the last N commits, use `/catchup <count>` (e.g. `/catchup 10`).",
			).WithParseMode(telego.ModeMarkdown))
			return
		}

		for _, c := range commits {
			if c.SHA == state.LastCommitSHA {
				found = true
				break
			}
			newCommits = append(newCommits, c)
		}
	}

	if len(newCommits) == 0 {
		_, _ = bot.SendMessage(ctx, tu.Message(
			tu.ID(chatID),
			"You are already caught up!\n\n*Tip:* If you want to force catch up on the last N commits, use `/catchup <count>` (e.g. `/catchup 10`).",
		).WithParseMode(telego.ModeMarkdown))
		state.LastCommitSHA = state.StartupCommitSHA
		_ = SaveState(state)
		return
	}

	copedMsg := ""
	if !found {
		copedMsg = "*Note:* Showing only the 30 most recent missed PoCs due to GitHub API limits.\n\n"
	}

	_, _ = bot.SendMessage(ctx, tu.Message(
		tu.ID(chatID),
		fmt.Sprintf("*Catching up: Processing %d missed commits...*", len(newCommits)),
	).WithParseMode(telego.ModeMarkdown))

	var count int
	// Process new commits from oldest to newest
	for i := len(newCommits) - 1; i >= 0; i-- {
		commit := newCommits[i]
		files, err := GetCommitChangedFiles(commit.SHA)
		if err != nil {
			log.Printf("Error getting changed files for commit %s: %v", commit.SHA, err)
			continue
		}

		for _, file := range files {
			infos, err := FetchPoCInfo(file)
			if err != nil {
				log.Printf("Error fetching PoC info for %s: %v", file, err)
				continue
			}

			for _, info := range infos {
				desc := info.Description
				if desc == "" {
					desc = "_No description provided_"
				}

				msg := fmt.Sprintf("*PoC for %s*\n\n*Description:*\n%s\n\n[PoC Repository](%s)", 
					info.Name, desc, info.RepositoryURL)
				
				if copedMsg != "" && count == 0 {
					msg = copedMsg + msg
				}

				_, err = bot.SendMessage(ctx, tu.Message(
					tu.ID(chatID),
					msg,
				).WithParseMode(telego.ModeMarkdown))
				if err != nil {
					log.Printf("Error sending telegram message: %v", err)
				} else {
					state.LastNotificationTime = time.Now()
				}
				count++
			}
		}
	}

	state.LastCommitSHA = commits[0].SHA
	state.StartupCommitSHA = commits[0].SHA
	if err := SaveState(state); err != nil {
		log.Printf("Error saving state: %v", err)
	}

	_, _ = bot.SendMessage(ctx, tu.Message(
		tu.ID(chatID),
		fmt.Sprintf("*Catch-up complete!* Sent %d PoC notifications.", count),
	).WithParseMode(telego.ModeMarkdown))
}
