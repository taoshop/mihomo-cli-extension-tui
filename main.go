package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// APIClient handles HTTP communication with mihomo-cli serve
type APIClient struct {
	baseURL string
	apiKey  string
}

func NewAPIClient(baseURL, apiKey string) *APIClient {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	return &APIClient{baseURL: baseURL, apiKey: apiKey}
}

func (c *APIClient) request(ctx context.Context, method, path string, result interface{}) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}

	if apiResp.Code != 0 {
		return fmt.Errorf(apiResp.Message)
	}

	// Marshal data back to JSON then unmarshal to result
	dataBytes, _ := json.Marshal(apiResp.Data)
	return json.Unmarshal(dataBytes, result)
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Status struct {
	Running   bool    `json:"running"`
	PID       int     `json:"pid"`
	Version   string  `json:"version"`
	Uptime    string  `json:"uptime"`
	UpTotal   uint64  `json:"up_total"`
	DownTotal uint64  `json:"down_total"`
	UpRate    float64 `json:"up_rate"`
	DownRate  float64 `json:"down_rate"`
}

type ProxyNode struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Server string `json:"server"`
	Port   int    `json:"port"`
}

type LogsResponse struct {
	Lines   []string `json:"lines"`
	Total   int      `json:"total"`
	LogFile string   `json:"logFile"`
}

func (c *APIClient) GetStatus(ctx context.Context) (*Status, error) {
	var status Status
	err := c.request(ctx, "GET", "/api/v1/status", &status)
	return &status, err
}

func (c *APIClient) GetNodes(ctx context.Context) ([]ProxyNode, error) {
	var nodes []ProxyNode
	err := c.request(ctx, "GET", "/api/v1/nodes", &nodes)
	return nodes, err
}

func (c *APIClient) SwitchNode(ctx context.Context, nodeName string) error {
	url := fmt.Sprintf("%s/api/v1/nodes/%s/switch", c.baseURL, nodeName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(`{"group":"GLOBAL"}`))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("switch failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *APIClient) GetRules(ctx context.Context) ([]string, error) {
	var rules []string
	err := c.request(ctx, "GET", "/api/v1/rules", &rules)
	return rules, err
}

func (c *APIClient) GetLogs(ctx context.Context, lines int) (*LogsResponse, error) {
	var logs LogsResponse
	path := fmt.Sprintf("/api/v1/logs?lines=%d", lines)
	err := c.request(ctx, "GET", path, &logs)
	return &logs, err
}

// ── tab indices ──
const (
	tabStatus = iota
	tabNodes
	tabRules
	tabLogs
	tabStats
)

var tabNames = []string{"Status", "Nodes", "Rules", "Logs", "Stats"}

// ── messages ──
type statusMsg struct{ status *Status }
type nodesMsg struct{ nodes []ProxyNode }
type rulesMsg struct{ rules []string }
type logsMsg struct{ logs *LogsResponse }
type tickMsg time.Time
type errMsg struct{ err error }

// ── model ──
type uiModel struct {
	client *APIClient

	tab      int
	width    int
	height   int
	quitting bool

	// data
	status *Status
	nodes  []ProxyNode
	rules  []string
	logs   *LogsResponse
	err    error

	// node table
	nodeTable table.Model
}

func newUIModel(client *APIClient) uiModel {
	t := table.New(
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetColumns([]table.Column{
		{Title: "Name", Width: 24},
		{Title: "Type", Width: 10},
		{Title: "Server", Width: 20},
		{Title: "Port", Width: 6},
	})
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62"))
	t.SetStyles(s)

	return uiModel{
		client:    client,
		nodeTable: t,
	}
}

func (m uiModel) Init() tea.Cmd {
	return tea.Batch(
		fetchStatus(m.client),
		fetchNodes(m.client),
		fetchRules(m.client),
		fetchLogs(m.client),
		tickEvery(),
	)
}

func (m uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.nodeTable.SetWidth(msg.Width - 4)
		m.nodeTable.SetHeight(msg.Height - 10)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "r":
			return m, tea.Batch(
				fetchStatus(m.client),
				fetchNodes(m.client),
				fetchRules(m.client),
				fetchLogs(m.client),
			)
		case "1":
			m.tab = tabStatus
		case "2":
			m.tab = tabNodes
		case "3":
			m.tab = tabRules
		case "4":
			m.tab = tabLogs
		case "5":
			m.tab = tabStats
		case "enter":
			if m.tab == tabNodes {
				return m, m.switchSelectedNode()
			}
		case "tab":
			m.tab = (m.tab + 1) % len(tabNames)
		}
		if m.tab == tabNodes {
			var cmd tea.Cmd
			m.nodeTable, cmd = m.nodeTable.Update(msg)
			return m, cmd
		}
		return m, nil

	case statusMsg:
		m.status = msg.status
		return m, nil

	case nodesMsg:
		m.nodes = msg.nodes
		rows := make([]table.Row, len(msg.nodes))
		for i, n := range msg.nodes {
			rows[i] = table.Row{n.Name, n.Type, n.Server, fmt.Sprintf("%d", n.Port)}
		}
		m.nodeTable.SetRows(rows)
		return m, nil

	case rulesMsg:
		m.rules = msg.rules
		return m, nil

	case logsMsg:
		m.logs = msg.logs
		return m, nil

	case tickMsg:
		return m, tea.Batch(fetchStatus(m.client), tickEvery())

	case errMsg:
		m.err = msg.err
		return m, nil

	default:
		return m, nil
	}
}

func (m uiModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	b.WriteString(headerStyle.Render("  mihomo-cli-tui v1.0.0"))
	b.WriteString("\n")

	b.WriteString(uiRenderTabs(m.tab, m.width))
	b.WriteString("\n")

	switch m.tab {
	case tabStatus:
		b.WriteString(m.renderStatus())
	case tabNodes:
		b.WriteString(m.nodeTable.View())
	case tabRules:
		b.WriteString(m.renderRules())
	case tabLogs:
		b.WriteString(m.renderLogs())
	case tabStats:
		b.WriteString(m.renderStats())
	}

	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().Faint(true)
	b.WriteString(footerStyle.Render("  ↑/↓ Navigate  Enter Switch node  r Refresh  1-5/Tab Switch tab  q Quit"))

	return b.String()
}

func uiRenderTabs(active, width int) string {
	var b strings.Builder
	for i, name := range tabNames {
		if i == active {
			activeStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("62")).
				Padding(0, 2).
				Bold(true)
			b.WriteString(activeStyle.Render(fmt.Sprintf("[%d]%s", i+1, name)))
		} else {
			inactiveStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Padding(0, 2)
			b.WriteString(inactiveStyle.Render(fmt.Sprintf("[%d]%s", i+1, name)))
		}
		b.WriteString(" ")
	}
	return b.String()
}

func (m uiModel) renderStatus() string {
	var b strings.Builder
	b.WriteString("\n")

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	if m.status == nil || !m.status.Running {
		b.WriteString("  ")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("○ Stopped"))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString("  ")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("● Running"))
	b.WriteString("\n\n")

	items := []struct{ label, value string }{
		{"PID", fmt.Sprintf("%d", m.status.PID)},
		{"Uptime", m.status.Uptime},
		{"Upload", uiFormatBytes(m.status.UpTotal)},
		{"Download", uiFormatBytes(m.status.DownTotal)},
	}

	for _, item := range items {
		b.WriteString("  ")
		b.WriteString(labelStyle.Render(item.label + ":"))
		b.WriteString(" ")
		b.WriteString(valStyle.Render(item.value))
		b.WriteString("\n")
	}

	return b.String()
}

func (m uiModel) renderRules() string {
	var b strings.Builder
	b.WriteString("\n")
	if len(m.rules) == 0 {
		b.WriteString("  (no rules)\n")
		return b.String()
	}
	for i, r := range m.rules {
		b.WriteString(fmt.Sprintf("  [%d] %s\n", i, r))
	}
	return b.String()
}

func (m uiModel) renderLogs() string {
	var b strings.Builder
	b.WriteString("\n")

	if m.logs == nil {
		b.WriteString("  Loading logs...\n")
		return b.String()
	}

	if len(m.logs.Lines) == 0 {
		b.WriteString(fmt.Sprintf("  (no log entries yet)\n  log file: %s\n", m.logs.LogFile))
		return b.String()
	}

	for _, line := range m.logs.Lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("\n  showing %d/%d lines from %s  |  press r to refresh\n",
		len(m.logs.Lines), m.logs.Total, m.logs.LogFile))
	return b.String()
}

func (m uiModel) renderStats() string {
	var b strings.Builder
	b.WriteString("\n")
	if m.status == nil || !m.status.Running {
		b.WriteString("  (service not running)\n")
		return b.String()
	}

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	items := []struct{ label, value string }{
		{"Total Upload", uiFormatBytes(m.status.UpTotal)},
		{"Total Download", uiFormatBytes(m.status.DownTotal)},
		{"Upload Rate", uiFormatRate(m.status.UpRate)},
		{"Download Rate", uiFormatRate(m.status.DownRate)},
	}
	for _, item := range items {
		b.WriteString("  ")
		b.WriteString(labelStyle.Render(item.label + ":"))
		b.WriteString(" ")
		b.WriteString(valStyle.Render(item.value))
		b.WriteString("\n")
	}
	return b.String()
}

func fetchStatus(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		status, err := client.GetStatus(ctx)
		if err != nil {
			return errMsg{err}
		}
		return statusMsg{status: status}
	}
}

func fetchNodes(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		nodes, err := client.GetNodes(ctx)
		if err != nil {
			return errMsg{err}
		}
		return nodesMsg{nodes: nodes}
	}
}

func fetchRules(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rules, err := client.GetRules(ctx)
		if err != nil {
			return errMsg{err}
		}
		return rulesMsg{rules: rules}
	}
}

func fetchLogs(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		logs, err := client.GetLogs(ctx, 50)
		if err != nil {
			return errMsg{err}
		}
		return logsMsg{logs: logs}
	}
}

func tickEvery() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *uiModel) switchSelectedNode() tea.Cmd {
	row := m.nodeTable.SelectedRow()
	if len(row) == 0 {
		return nil
	}
	nodeName := row[0]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.client.SwitchNode(ctx, nodeName); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func uiFormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func uiFormatRate(bytesPerSec float64) string {
	const unit = 1024
	b := bytesPerSec
	if b < unit {
		return fmt.Sprintf("%.0f B/s", b)
	}
	div, exp := float64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB/s", b/div, "KMGTPE"[exp])
}

func main() {
	// Get API endpoint from environment or use default
	apiAddr := os.Getenv("MIHOMO_CLI_API_ADDR")
	if apiAddr == "" {
		apiAddr = "http://127.0.0.1:8080"
	}
	apiKey := os.Getenv("MIHOMO_CLI_API_KEY")

	client := NewAPIClient(apiAddr, apiKey)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := client.GetStatus(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot connect to mihomo-cli serve at %s\n", apiAddr)
		fmt.Fprintf(os.Stderr, "Please ensure:\n")
		fmt.Fprintf(os.Stderr, "  1. mihomo-cli is running: mihomo-cli serve\n")
		fmt.Fprintf(os.Stderr, "  2. API address is correct (current: %s)\n", apiAddr)
		fmt.Fprintf(os.Stderr, "\nSet MIHOMO_CLI_API_ADDR to change the API endpoint.\n")
		os.Exit(1)
	}

	p := tea.NewProgram(newUIModel(client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
