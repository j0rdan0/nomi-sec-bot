package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"nomi-sec-bot/poc"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

type tabID int

const (
	tabLatest tabID = iota
	tabQuery
)

var (
	// Colors
	bgColor       = lipgloss.Color("#1a1a24") // Charcoal/Dark blue
	primaryColor  = lipgloss.Color("#79a3ec") // Pastel Blue
	accentColor   = lipgloss.Color("#f1a5c2") // Pink
	activeColor   = lipgloss.Color("#a277ff") // Bright Purple/Indigo
	inactiveColor = lipgloss.Color("#4c4c5e") // Slate Gray
	greenColor    = lipgloss.Color("#7cd5a1") // Pastel Green
	redColor      = lipgloss.Color("#ff757f") // Pastel Red
	borderColor   = lipgloss.Color("#2f303d") // Border gray
	textColor     = lipgloss.Color("#e2e4e9") // Light gray

	// Styles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a1a24")).
			Background(activeColor).
			Padding(0, 2).
			Bold(true).
			MarginBottom(1)

	tabStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(inactiveColor).
			Padding(0, 2).
			MarginRight(1)

	activeTabStyle = tabStyle.
			Background(activeColor).
			Foreground(lipgloss.Color("#1a1a24")).
			Bold(true)

	docStyle = lipgloss.NewStyle().
			Padding(1, 2)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1)

	highlightStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(greenColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(redColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626880")).
			Italic(true)
)

type latestPoCsMsg []poc.PoCInfo
type latestPoCsErrMsg error

type searchResultItem struct {
	id       string
	pocInfos []poc.PoCInfo
	err      error
}

type searchResultsMsg []searchResultItem
type searchErrMsg error

type model struct {
	activeTab           tabID
	width               int
	height              int

	// Tab 1: Latest PoCs
	latestPoCs          []poc.PoCInfo
	latestLoading       bool
	latestErr           error
	latestViewport      viewport.Model
	latestSelectedIndex int

	// Tab 2: Query CVE
	searchInput         textinput.Model
	searchLoading       bool
	searchErr           error
	searchItems         []searchResultItem
	flatSearchPoCs      []poc.PoCInfo
	searchViewport      viewport.Model
	querySelectedIndex  int

	// Global spinner
	spinner             spinner.Model

	// Status message for feedback
	statusMessage       string
}

func initialModel(initialCVE string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	si := textinput.New()
	si.Placeholder = "e.g. 2024 5 or CVE-2024-1234"

	activeTab := tabLatest
	searchLoading := false
	if initialCVE != "" {
		activeTab = tabQuery
		si.SetValue(initialCVE)
		si.Blur()
		searchLoading = true
	} else {
		si.Focus()
	}

	return model{
		activeTab:      activeTab,
		latestLoading:  true,
		spinner:        s,
		searchInput:    si,
		latestViewport: viewport.New(0, 0),
		searchViewport: viewport.New(0, 0),
		width:          80,
		height:         24,
		searchLoading:  searchLoading,
	}
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

func fetchLatest5PoCsCmd() tea.Cmd {
	return func() tea.Msg {
		commits, err := poc.GetRecentCommits()
		if err != nil {
			return latestPoCsErrMsg(err)
		}

		currentYearStr := strconv.Itoa(time.Now().Year())

		var latestPoCs []poc.PoCInfo
		for _, commit := range commits {
			files, err := poc.GetCommitChangedFiles(commit.SHA)
			if err != nil {
				continue
			}

			for _, file := range files {
				parts := strings.Split(file, "/")
				if len(parts) == 0 || parts[0] != currentYearStr {
					continue
				}

				pocs, err := poc.FetchPoCInfo(file)
				if err == nil {
					latestPoCs = append(latestPoCs, pocs...)
					if len(latestPoCs) >= 5 {
						latestPoCs = latestPoCs[:5]
						return latestPoCsMsg(latestPoCs)
					}
				}
			}
		}

		return latestPoCsMsg(latestPoCs)
	}
}

func searchCVEByYearCmd(year string, count int) tea.Cmd {
	return func() tea.Msg {
		cveIDs, err := poc.GetCVEsForYear(year, count)
		if err != nil {
			return searchErrMsg(err)
		}

		var items []searchResultItem
		for _, id := range cveIDs {
			filePath := fmt.Sprintf("%s/%s.json", year, id)
			pocs, err := poc.FetchPoCInfo(filePath)
			items = append(items, searchResultItem{
				id:       id,
				pocInfos: pocs,
				err:      err,
			})
		}
		return searchResultsMsg(items)
	}
}

func searchCVEByIDCmd(cveID string) tea.Cmd {
	return func() tea.Msg {
		cveIDUpper := strings.ToUpper(strings.TrimSpace(cveID))
		cveIDUpper = strings.ReplaceAll(cveIDUpper, " ", "-")
		if !strings.HasPrefix(cveIDUpper, "CVE-") && strings.Contains(cveIDUpper, "-") {
			cveIDUpper = "CVE-" + cveIDUpper
		}

		parts := strings.Split(cveIDUpper, "-")
		if len(parts) < 3 || parts[0] != "CVE" {
			return searchErrMsg(fmt.Errorf("invalid CVE ID format: expected CVE-YYYY-NNNN"))
		}
		year := parts[1]
		filePath := fmt.Sprintf("%s/%s.json", year, cveIDUpper)
		pocs, err := poc.FetchPoCInfo(filePath)
		if err != nil {
			return searchErrMsg(fmt.Errorf("no PoCs found or error fetching details for %s", cveIDUpper))
		}
		return searchResultsMsg([]searchResultItem{
			{
				id:       cveIDUpper,
				pocInfos: pocs,
			},
		})
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		fetchLatest5PoCsCmd(),
		m.spinner.Tick,
		textinput.Blink,
	}
	if m.searchInput.Value() != "" {
		cmds = append(cmds, searchCVEByIDCmd(m.searchInput.Value()))
	}
	return tea.Batch(cmds...)
}

func (m *model) updateFocus() {
	if m.activeTab == tabQuery {
		m.searchInput.Focus()
	} else {
		m.searchInput.Blur()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		// Tab switching
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			m.updateFocus()
			m.statusMessage = ""
		case "shift+tab":
			if m.activeTab == 0 {
				m.activeTab = 1
			} else {
				m.activeTab--
			}
			m.updateFocus()
			m.statusMessage = ""
		case "1":
			m.activeTab = tabLatest
			m.updateFocus()
			m.statusMessage = ""
		case "2":
			m.activeTab = tabQuery
			m.updateFocus()
			m.statusMessage = ""

		// Tab 1: Refresh actions
		case "r":
			if m.activeTab == tabLatest {
				m.latestLoading = true
				m.latestErr = nil
				m.statusMessage = ""
				m.latestViewport.SetContent("")
				cmds = append(cmds, fetchLatest5PoCsCmd())
			}

		// Navigation
		case "up", "down":
			if m.activeTab == tabLatest && len(m.latestPoCs) > 0 {
				if msg.String() == "up" {
					if m.latestSelectedIndex > 0 {
						m.latestSelectedIndex--
					}
				} else {
					if m.latestSelectedIndex < len(m.latestPoCs)-1 {
						m.latestSelectedIndex++
					}
				}
				m.latestViewport.SetContent(formatLatestPoCs(m.latestPoCs, m.latestSelectedIndex))
			} else if m.activeTab == tabQuery && len(m.flatSearchPoCs) > 0 {
				if m.searchInput.Focused() {
					if msg.String() == "down" {
						m.searchInput.Blur()
						m.querySelectedIndex = 0
						m.searchViewport.SetContent(formatSearchResults(m.searchItems, m.querySelectedIndex))
					}
				} else {
					if msg.String() == "up" {
						if m.querySelectedIndex > 0 {
							m.querySelectedIndex--
							m.searchViewport.SetContent(formatSearchResults(m.searchItems, m.querySelectedIndex))
						} else {
							m.searchInput.Focus()
							m.searchViewport.SetContent(formatSearchResults(m.searchItems, -1))
						}
					} else if msg.String() == "down" {
						if m.querySelectedIndex < len(m.flatSearchPoCs)-1 {
							m.querySelectedIndex++
							m.searchViewport.SetContent(formatSearchResults(m.searchItems, m.querySelectedIndex))
						}
					}
				}
			}

		// Open in browser / Search Enter
		case "o", "enter":
			if m.activeTab == tabLatest && len(m.latestPoCs) > 0 {
				url := m.latestPoCs[m.latestSelectedIndex].RepositoryURL
				err := openBrowser(url)
				if err != nil {
					m.statusMessage = fmt.Sprintf("Error opening browser: %v", err)
				} else {
					m.statusMessage = fmt.Sprintf("Opened: %s", url)
				}
			} else if m.activeTab == tabQuery {
				if !m.searchInput.Focused() && len(m.flatSearchPoCs) > 0 {
					url := m.flatSearchPoCs[m.querySelectedIndex].RepositoryURL
					err := openBrowser(url)
					if err != nil {
						m.statusMessage = fmt.Sprintf("Error opening browser: %v", err)
					} else {
						m.statusMessage = fmt.Sprintf("Opened: %s", url)
					}
				} else if m.searchInput.Focused() && msg.String() == "enter" {
					val := m.searchInput.Value()
					if val != "" {
						m.searchLoading = true
						m.searchErr = nil
						m.searchItems = nil
						m.flatSearchPoCs = nil
						m.statusMessage = ""
						m.searchViewport.SetContent("")

						valUpper := strings.ToUpper(strings.TrimSpace(val))
						isCVEQuery := strings.HasPrefix(valUpper, "CVE-") ||
							(strings.Contains(valUpper, "-") && !strings.Contains(valUpper, " "))

						if isCVEQuery {
							cmds = append(cmds, searchCVEByIDCmd(val))
						} else {
							fields := strings.Fields(val)
							if len(fields) == 1 {
								cmds = append(cmds, searchCVEByIDCmd(fields[0]))
							} else if len(fields) == 2 {
								year := fields[0]
								count, err := strconv.Atoi(fields[1])
								if err != nil || count <= 0 {
									m.searchLoading = false
									m.searchErr = fmt.Errorf("invalid count: please enter a positive number")
								} else {
									cmds = append(cmds, searchCVEByYearCmd(year, count))
								}
							} else {
								m.searchLoading = false
								m.searchErr = fmt.Errorf("invalid query format. Enter '<year> <count>' or '<CVE-ID>'")
							}
						}
					}
				}
			}

		case "esc", "/":
			if m.activeTab == tabQuery && !m.searchInput.Focused() {
				m.searchInput.Focus()
				m.searchViewport.SetContent(formatSearchResults(m.searchItems, -1))
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		m.latestViewport.Width = m.width - 6
		m.latestViewport.Height = m.height - 12
		m.searchViewport.Width = m.width - 8
		m.searchViewport.Height = m.height - 14

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case latestPoCsMsg:
		m.latestPoCs = msg
		m.latestLoading = false
		m.latestSelectedIndex = 0
		m.latestViewport.SetContent(formatLatestPoCs(msg, m.latestSelectedIndex))

	case latestPoCsErrMsg:
		m.latestErr = msg
		m.latestLoading = false

	case searchResultsMsg:
		m.searchItems = msg
		m.searchLoading = false
		m.flatSearchPoCs = nil
		for _, item := range msg {
			if item.err == nil {
				m.flatSearchPoCs = append(m.flatSearchPoCs, item.pocInfos...)
			}
		}
		m.querySelectedIndex = 0
		if !m.searchInput.Focused() && len(m.flatSearchPoCs) > 0 {
			m.searchViewport.SetContent(formatSearchResults(msg, 0))
		} else {
			m.searchViewport.SetContent(formatSearchResults(msg, -1))
		}

	case searchErrMsg:
		m.searchErr = msg
		m.searchLoading = false
	}

	if m.activeTab == tabQuery && m.searchInput.Focused() {
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.activeTab == tabLatest {
		m.latestViewport, cmd = m.latestViewport.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.activeTab == tabQuery {
		m.searchViewport, cmd = m.searchViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func formatLatestPoCs(pocs []poc.PoCInfo, selectedIndex int) string {
	if len(pocs) == 0 {
		return "No recent PoCs found."
	}
	var sb strings.Builder
	for i, info := range pocs {
		desc := info.Description
		if desc == "" {
			desc = "_No description provided_"
		}

		var rendered string
		if i == selectedIndex {
			highlightedBorder := lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(activeColor).
				Padding(1)

			rendered = highlightedBorder.Render(
				fmt.Sprintf("%s\n\n%s\n\n%s: %s",
					successStyle.Render(fmt.Sprintf("> %d. %s", i+1, info.Name)),
					desc,
					highlightStyle.Render("Repository"),
					info.RepositoryURL,
				),
			)
		} else {
			rendered = borderStyle.Render(
				fmt.Sprintf("  %d. %s\n\n%s\n\n%s: %s",
					i+1, info.Name,
					desc,
					highlightStyle.Render("Repository"),
					info.RepositoryURL,
				),
			)
		}
		sb.WriteString(rendered)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func formatSearchResults(items []searchResultItem, selectedIndex int) string {
	if len(items) == 0 {
		return "No results found."
	}
	var sb strings.Builder
	globalPoCIndex := 0

	for _, item := range items {
		if item.err != nil {
			sb.WriteString(errorStyle.Render(fmt.Sprintf("Error fetching %s: %v\n\n", item.id, item.err)))
			continue
		}
		if len(item.pocInfos) == 0 {
			sb.WriteString(highlightStyle.Render(fmt.Sprintf("ID: %s\n", item.id)))
			sb.WriteString("No PoCs found.\n\n")
			continue
		}

		sb.WriteString(highlightStyle.Render("ID: "+item.id) + "\n\n")

		for _, info := range item.pocInfos {
			desc := info.Description
			if desc == "" {
				desc = "_No description provided_"
			}

			var rendered string
			if globalPoCIndex == selectedIndex {
				highlightedBorder := lipgloss.NewStyle().
					Border(lipgloss.DoubleBorder()).
					BorderForeground(activeColor).
					Padding(1)

				rendered = highlightedBorder.Render(
					fmt.Sprintf("%s\n\n%s\n\n%s: %s",
						successStyle.Render("> "+info.Name),
						desc,
						highlightStyle.Render("Link"),
						info.RepositoryURL,
					),
				)
			} else {
				rendered = borderStyle.Render(
					fmt.Sprintf("  %s\n\n%s\n\n%s: %s",
						info.Name,
						desc,
						highlightStyle.Render("Link"),
						info.RepositoryURL,
					),
				)
			}
			sb.WriteString(rendered)
			sb.WriteString("\n\n")
			globalPoCIndex++
		}

		divider := "────────────────────────────────────────────────"
		sb.WriteString(divider + "\n\n")
	}
	return sb.String()
}

func formatPoCLinks(infos []poc.PoCInfo) string {
	var links []string
	for _, info := range infos {
		links = append(links, fmt.Sprintf("• %s: %s", info.Name, info.RepositoryURL))
	}
	return strings.Join(links, "\n")
}

func (m model) View() string {
	var sb strings.Builder

	// Header / Title
	sb.WriteString(titleStyle.Render("NOMI-SEC LATEST PoCs"))
	sb.WriteString("\n")

	// Tabs
	var tabs []string
	tabNames := []string{"[1] Latest 5 PoCs", "[2] Query CVE"}
	for i, name := range tabNames {
		if tabID(i) == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, tabStyle.Render(name))
		}
	}
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
	sb.WriteString("\n\n")

	// Content based on tab
	var tabContent string
	switch m.activeTab {
	case tabLatest:
		var latestSB strings.Builder
		if m.latestLoading {
			latestSB.WriteString(m.spinner.View() + " Loading latest PoCs from GitHub...")
		} else if m.latestErr != nil {
			latestSB.WriteString(errorStyle.Render(fmt.Sprintf("Error loading PoCs: %v", m.latestErr)))
		} else {
			latestSB.WriteString(m.latestViewport.View())
		}
		tabContent = borderStyle.Width(m.width - 6).Height(m.height - 10).Render(latestSB.String())

	case tabQuery:
		var querySB strings.Builder
		querySB.WriteString(highlightStyle.Render("Query CVE by ID or Year + Count") + "\n\n")
		querySB.WriteString("Enter query: (Format: 'CVE-YYYY-NNNN' or 'YYYY Count')\n")
		querySB.WriteString(m.searchInput.View() + "\n\n")

		if m.searchLoading {
			querySB.WriteString(m.spinner.View() + " Searching GitHub...")
		} else if m.searchErr != nil {
			querySB.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.searchErr)))
		} else {
			querySB.WriteString(m.searchViewport.View())
		}
		tabContent = borderStyle.Width(m.width - 6).Height(m.height - 10).Render(querySB.String())
	}
	sb.WriteString(tabContent)
	sb.WriteString("\n\n")

	// Visual Feedback status message
	if m.statusMessage != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Italic(true).Render(m.statusMessage))
		sb.WriteString("\n\n")
	}

	var helpMsg string
	switch m.activeTab {
	case tabLatest:
		helpMsg = "↑/↓: navigate • enter/o: open in browser • r: refresh • tab: next tab • q: quit"
	case tabQuery:
		if m.searchInput.Focused() {
			helpMsg = "enter: search • down: focus results list • tab: next tab • q: quit"
		} else {
			helpMsg = "↑/↓: navigate results • enter/o: open in browser • esc: focus search input • tab: next tab • q: quit"
		}
	}
	sb.WriteString(helpStyle.Render(helpMsg))

	return docStyle.Render(sb.String())
}

func main() {
	if err := godotenv.Load(); err != nil {
		// Env file not found is fine
	}

	// Disable standard logging to stderr/stdout to avoid corrupting Bubble Tea TUI
	f, err := os.OpenFile("tui.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		defer f.Close()
		log.SetOutput(f)
	} else {
		nullFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err == nil {
			defer nullFile.Close()
			log.SetOutput(nullFile)
		}
	}

	var initialCVE string
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "-h" || arg == "--help" || arg == "-help" {
			fmt.Printf("Usage: %s [CVE-YYYY-NNNN]\n", os.Args[0])
			os.Exit(0)
		}
		initialCVE = arg
	}

	p := tea.NewProgram(initialModel(initialCVE), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
