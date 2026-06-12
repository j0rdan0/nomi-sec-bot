package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func handleCVECommand(bot *telego.Bot, message telego.Message) {
	ctx := context.Background()
	args := strings.Fields(message.Text)
	
	if len(args) == 2 {
		cveID := strings.TrimSpace(args[1])
		cveIDUpper := strings.ToUpper(cveID)
		if strings.HasPrefix(cveIDUpper, "CVE-") {
			handleSpecificCVEQuery(bot, message.Chat.ID, cveIDUpper)
			return
		}
	}

	if len(args) != 3 {
		_, _ = bot.SendMessage(ctx, tu.Message(
			tu.ID(message.Chat.ID),
			"Usage:\n• `/cve <year> <count>`\n• `/cve <CVE-ID>` (e.g., `/cve CVE-2002-1614`)",
		).WithParseMode(telego.ModeMarkdown))
		return
	}

	year := args[1]
	count, err := strconv.Atoi(args[2])
	if err != nil || count <= 0 {
		_, _ = bot.SendMessage(ctx, tu.Message(
			tu.ID(message.Chat.ID),
			"Invalid count. Please provide a positive number.",
		))
		return
	}

	if count > 20 {
		count = 20 // Lower limit for detailed output
	}

	cveIDs, err := GetCVEsForYear(year, count)
	if err != nil {
		_, _ = bot.SendMessage(ctx, tu.Message(
			tu.ID(message.Chat.ID),
			fmt.Sprintf("Error fetching CVEs: %v", err),
		))
		return
	}

	if len(cveIDs) == 0 {
		_, _ = bot.SendMessage(ctx, tu.Message(
			tu.ID(message.Chat.ID),
			fmt.Sprintf("No CVEs found for year %s.", year),
		))
		return
	}

	_, _ = bot.SendMessage(ctx, tu.Message(
		tu.ID(message.Chat.ID),
		fmt.Sprintf("🔍 *Fetching details for top %d CVEs in %s...*", len(cveIDs), year),
	).WithParseMode(telego.ModeMarkdown))

	for _, id := range cveIDs {
		filePath := fmt.Sprintf("%s/%s.json", year, id)
		infos, err := FetchPoCInfo(filePath)
		if err != nil {
			log.Printf("Error fetching info for %s: %v", id, err)
			continue
		}

		if len(infos) == 0 {
			continue
		}

		// Use the first PoC for the main ID/Description
		mainInfo := infos[0]
		desc := mainInfo.Description
		if desc == "" {
			desc = "_No description provided_"
		}

		divider := "────────────────────"
		msg := fmt.Sprintf("%s\n📦 *ID:* %s\n\n📝 *Description:*\n%s\n\n🔗 *PoC Links:*\n", divider, id, desc)
		
		var links []string
		for _, info := range infos {
			links = append(links, fmt.Sprintf("• [%s](%s)", info.Name, info.RepositoryURL))
		}
		msg += strings.Join(links, "\n")

		_, err = bot.SendMessage(ctx, tu.Message(
			tu.ID(message.Chat.ID),
			msg,
		).WithParseMode(telego.ModeMarkdown))
		if err != nil {
			log.Printf("Error sending message for %s: %v", id, err)
		}
	}
}

func handleCatchUpCommand(bot *telego.Bot, message telego.Message) {
	CatchUp(bot, message.Chat.ID)
}

func handleSpecificCVEQuery(bot *telego.Bot, chatID int64, cveID string) {
	ctx := context.Background()
	parts := strings.Split(cveID, "-")
	if len(parts) < 2 {
		_, _ = bot.SendMessage(ctx, tu.Message(
			tu.ID(chatID),
			"❌ Invalid CVE format. Expected `CVE-YYYY-NNNN`.",
		).WithParseMode(telego.ModeMarkdown))
		return
	}

	year := parts[1]
	filePath := fmt.Sprintf("%s/%s.json", year, cveID)

	_, _ = bot.SendMessage(ctx, tu.Message(
		tu.ID(chatID),
		fmt.Sprintf("🔍 *Fetching details for %s...*", cveID),
	).WithParseMode(telego.ModeMarkdown))

	infos, err := FetchPoCInfo(filePath)
	if err != nil {
		log.Printf("Error fetching info for %s: %v", cveID, err)
		_, _ = bot.SendMessage(ctx, tu.Message(
			tu.ID(chatID),
			fmt.Sprintf("❌ No PoCs found or error fetching details for *%s*.", cveID),
		).WithParseMode(telego.ModeMarkdown))
		return
	}

	if len(infos) == 0 {
		_, _ = bot.SendMessage(ctx, tu.Message(
			tu.ID(chatID),
			fmt.Sprintf("❌ No PoCs found for *%s*.", cveID),
		).WithParseMode(telego.ModeMarkdown))
		return
	}

	mainInfo := infos[0]
	desc := mainInfo.Description
	if desc == "" {
		desc = "_No description provided_"
	}

	divider := "────────────────────"
	msg := fmt.Sprintf("%s\n📦 *ID:* %s\n\n📝 *Description:*\n%s\n\n🔗 *PoC Links:*\n", divider, cveID, desc)

	var links []string
	for _, info := range infos {
		links = append(links, fmt.Sprintf("• [%s](%s)", info.Name, info.RepositoryURL))
	}
	msg += strings.Join(links, "\n")

	_, err = bot.SendMessage(ctx, tu.Message(
		tu.ID(chatID),
		msg,
	).WithParseMode(telego.ModeMarkdown))
	if err != nil {
		log.Printf("Error sending message for %s: %v", cveID, err)
	}
}
