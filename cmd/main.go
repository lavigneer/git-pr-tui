package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	git "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v63/github"
	"github.com/joho/godotenv"
)

const (
	githubHTTPPrefix = "https://github.com/"
	githubSSHPrefix  = "git@github.com:"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table table.Model
	prs   []*github.PullRequest
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			pr := m.prs[m.table.Cursor()]
			if pr != nil && (strings.HasPrefix(*pr.HTMLURL, "https://") || strings.HasPrefix(*pr.HTMLURL, "http://")) {
				cmd := exec.Command("xdg-open", *pr.HTMLURL) //nolint gosec
				return m, tea.ExecProcess(cmd, nil)
			}
		}
	default:
		break
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func getRepoAndOwner() (string, string) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		log.Fatal("Error loading repo")
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		log.Fatal("Error loading origin remote")
	}

	remoteURLs := remote.Config().URLs
	if len(remoteURLs) < 1 {
		log.Fatal("Error getting origin's url")
	}

	remoteURL := remoteURLs[0]
	switch {
	case strings.HasPrefix(remoteURL, githubHTTPPrefix):
		remoteURL, _ = strings.CutPrefix(remoteURL, githubHTTPPrefix)
	case strings.HasPrefix(remoteURL, githubSSHPrefix):
		remoteURL, _ = strings.CutPrefix(remoteURL, githubSSHPrefix)
	default:
		log.Fatal("Error parsing remote github url")
	}
	remoteURL, _ = strings.CutSuffix(remoteURL, ".git")

	remoteURLSplit := strings.Split(remoteURL, "/")
	return remoteURLSplit[0], remoteURLSplit[1]
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	githubAPIToken := os.Getenv("GITHUB_API_TOKEN")
	owner, repo := getRepoAndOwner()

	githubClient := github.NewClient(nil).WithAuthToken(githubAPIToken)
	// user, _, err := githubClient.Users.Get(context.Background(), "")
	prs, _, err := githubClient.PullRequests.List(context.Background(), owner, repo, nil)
	if err != nil {
		log.Fatalf("Error fetching prs %s", err)
	}

	rows := []table.Row{}
	for _, pr := range prs {
		labels := []string{}
		for _, label := range pr.Labels {
			labels = append(labels, *label.Name)
		}
		rows = append(rows, table.Row{*pr.Title, pr.GetUser().GetLogin(), strings.Join(labels, ", "), pr.CreatedAt.Format("Mon Jan 2 15:04 MST 2006")})
	}

	columns := []table.Column{
		{Title: "Summary", Width: 35},
		{Title: "Author", Width: 20},
		{Title: "Labels", Width: 30},
		{Title: "Date", Width: 26},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{t, prs}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		log.Fatal("Error running program:", err)
	}
}
