package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	list               list.Model
	filters            []filter
	filterIndex        int
	tabIndex           int
	status             string
	lastUpdated        time.Time
	loading            bool
	err                error
	width              int
	height             int
	tokenMissing       bool
	showDetail         bool
	detailLoading      bool
	detailErr          error
	detailItem         detail
	spinner            spinner.Model
	textarea           textarea.Model
	commentMode        bool
	confirmMode        bool
	actionLoading      bool
	actionItem         issueItem
	confirmTargetState string
	statusOverride     bool
	commentPage        int
	styles             uiStyles
}

func newModel(l list.Model, styles uiStyles) model {
	m := model{
		list:        l,
		filters:     filters,
		filterIndex: 0,
		tabIndex:    0,
		status:      "Loading…",
		loading:     true,
		spinner:     spinner.New(spinner.WithSpinner(spinner.Line)),
		commentPage: 1,
		styles:      styles,
	}
	m.textarea = textarea.New()
	m.textarea.Placeholder = "Write a comment..."
	m.textarea.CharLimit = 4000
	m.textarea.SetWidth(80)
	m.textarea.SetHeight(10)
	m.textarea.FocusedStyle.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#1e66f5"))
	m.textarea.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca0b0"))
	m.textarea.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#4c4f69"))
	m.textarea.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("#1e66f5")).Bold(true)
	m.textarea.FocusedStyle.Base = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#bcc0cc"))
	m.textarea.BlurredStyle = m.textarea.FocusedStyle
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchCmd(m.filters[m.filterIndex], tabs[m.tabIndex].Kind), m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-2, msg.Height-7)
		m.textarea.SetWidth(max(20, msg.Width-4))
		m.textarea.SetHeight(max(6, msg.Height-10))
		return m, nil
		case tea.KeyMsg:
			if m.commentMode {
				switch msg.String() {
				case "esc":
					m.commentMode = false
					m.textarea.Blur()
					m.textarea.SetValue("")
					return m, nil
				case "ctrl+g":
					body := strings.TrimSpace(m.textarea.Value())
					if body == "" {
						m.status = "Comment is empty"
						m.statusOverride = true
						return m, nil
				}
				m.commentMode = false
				m.textarea.Blur()
				m.textarea.SetValue("")
				m.actionLoading = true
				m.status = fmt.Sprintf("Sending comment to %s#%d...", m.actionItem.Repo, m.actionItem.Number)
				m.statusOverride = true
				return m, postCommentCmd(m.actionItem, body)
			}
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		if m.confirmMode {
			switch msg.String() {
			case "y":
				m.confirmMode = false
				m.actionLoading = true
				m.status = actionProgress(m.confirmTargetState)
				return m, updateIssueStateCmd(m.actionItem, m.confirmTargetState)
			case "n", "esc":
				m.confirmMode = false
				return m, nil
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.showDetail {
				m.showDetail = false
				m.detailErr = nil
				return m, nil
			}
		case "j", "down":
			if !m.showDetail {
				m.list.CursorDown()
			}
			return m, nil
		case "k", "up":
			if !m.showDetail {
				m.list.CursorUp()
			}
			return m, nil
		case "r":
			if m.showDetail {
				if item, ok := m.list.SelectedItem().(issueItem); ok {
					m.detailLoading = true
					m.detailErr = nil
					return m, fetchDetailCmd(item, m.commentPage)
				}
				return m, nil
			}
			m.loading = true
			m.status = "Refreshing..."
			return m, fetchCmd(m.filters[m.filterIndex], tabs[m.tabIndex].Kind)
		case "f":
			if !m.showDetail {
				m.filterIndex = (m.filterIndex + 1) % len(m.filters)
				m.loading = true
				m.status = "Loading..."
				m.list.SetItems(nil)
				return m, fetchCmd(m.filters[m.filterIndex], tabs[m.tabIndex].Kind)
			}
			return m, nil
		case "tab":
			if !m.showDetail {
				m.tabIndex = (m.tabIndex + 1) % len(tabs)
				m.loading = true
				m.status = "Loading..."
				m.list.SetItems(nil)
				return m, fetchCmd(m.filters[m.filterIndex], tabs[m.tabIndex].Kind)
			}
			return m, nil
		case "c":
			if m.showDetail && m.detailItem.Title != "" {
				m.commentMode = true
				m.actionItem = issueItem{
					TitleText: m.detailItem.Title,
					Repo:      m.detailItem.Repo,
					Number:    m.detailItem.Number,
					URL:       m.detailItem.URL,
					Kind:      m.detailItem.Kind,
				}
				m.textarea.Focus()
				m.textarea.SetValue("")
				return m, nil
			}
			if item, ok := m.list.SelectedItem().(issueItem); ok {
				m.commentMode = true
				m.actionItem = item
				m.textarea.Focus()
				m.textarea.SetValue("")
				return m, nil
			}
			return m, nil
		case "n":
			if m.showDetail && m.detailItem.HasNextComments {
				m.commentPage = m.detailItem.CommentPage + 1
				m.detailLoading = true
				m.detailErr = nil
				return m, fetchDetailCmd(m.actionItem, m.commentPage)
			}
			return m, nil
		case "p":
			if m.showDetail && m.detailItem.HasPrevComments {
				m.commentPage = m.detailItem.CommentPage - 1
				if m.commentPage < 1 {
					m.commentPage = 1
				}
				m.detailLoading = true
				m.detailErr = nil
				return m, fetchDetailCmd(m.actionItem, m.commentPage)
			}
			return m, nil
		case "x":
			if m.showDetail && m.detailItem.Title != "" {
				m.confirmMode = true
				m.actionItem = issueItem{
					TitleText: m.detailItem.Title,
					Repo:      m.detailItem.Repo,
					Number:    m.detailItem.Number,
					URL:       m.detailItem.URL,
					Kind:      m.detailItem.Kind,
				}
				if m.detailItem.State == "closed" {
					m.confirmTargetState = "open"
				} else {
					m.confirmTargetState = "closed"
				}
				return m, nil
			}
			if item, ok := m.list.SelectedItem().(issueItem); ok {
				m.confirmMode = true
				m.actionItem = item
				m.confirmTargetState = "closed"
				return m, nil
			}
			return m, nil
		case "o":
			if item, ok := m.list.SelectedItem().(issueItem); ok {
				return m, openURLCmd(item.URL)
			}
			return m, nil
		case "enter":
			if item, ok := m.list.SelectedItem().(issueItem); ok {
				m.showDetail = true
				m.commentPage = 1
				m.detailLoading = true
				m.detailErr = nil
				return m, fetchDetailCmd(item, m.commentPage)
			}
			return m, nil
		}
	case fetchResult:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
			m.statusOverride = true
			return m, nil
		}

		items := make([]list.Item, 0, len(msg.items))
		for _, item := range msg.items {
			items = append(items, item)
		}
		m.list.SetItems(items)
		m.lastUpdated = time.Now()
		m.status = fmt.Sprintf("Loaded %d items • updated %s", len(items), humanizeSince(m.lastUpdated))
		m.statusOverride = false
		return m, nil
	case detailResult:
		m.detailLoading = false
		m.detailErr = msg.err
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
			m.statusOverride = true
			return m, nil
		}
		m.detailItem = msg.item
		m.commentPage = msg.item.CommentPage
		return m, nil
	case commentResult:
		m.actionLoading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
			m.statusOverride = true
			return m, nil
		}
		m.status = fmt.Sprintf("Comment posted to %s#%d", m.actionItem.Repo, m.actionItem.Number)
		m.statusOverride = true
		if m.showDetail {
			m.detailLoading = true
			return m, fetchDetailCmd(m.actionItem, m.commentPage)
		}
		return m, nil
	case stateResult:
		m.actionLoading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %s", msg.err.Error())
			m.statusOverride = true
			return m, nil
		}
		if msg.state == "closed" {
			m.status = "Closed"
		} else {
			m.status = "Reopened"
		}
		m.statusOverride = true
		m.loading = true
		if m.showDetail {
			m.detailLoading = true
			return m, tea.Batch(fetchCmd(m.filters[m.filterIndex], tabs[m.tabIndex].Kind), fetchDetailCmd(m.actionItem, m.commentPage))
		}
		return m, fetchCmd(m.filters[m.filterIndex], tabs[m.tabIndex].Kind)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}

	header := m.styles.Title.Render("GitHub Inbox")
	filterText := m.styles.Filter.Render(m.filters[m.filterIndex].Name)
	title := fmt.Sprintf("%s  ·  %s", header, filterText)
	tabsLine := renderTabs(tabs, m.tabIndex, m.styles)

	body := m.list.View()
	if m.showDetail {
		body = m.detailView()
	}
	if m.confirmMode {
		body = m.confirmView()
	}
	if m.commentMode {
		body = m.commentView()
	}
	if m.loading && !m.showDetail && !m.confirmMode && !m.commentMode {
		body = fmt.Sprintf("%s %s", m.spinner.View(), m.styles.MutedText.Render("Loading list..."))
	}

	hotkeyStyle := m.styles.HelpKey
	helpTextStyle := m.styles.HelpText
	help := fmt.Sprintf(
		"%s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s",
		hotkeyStyle.Render("↑/↓ j/k"), helpTextStyle.Render("navigate"),
		hotkeyStyle.Render("enter"), helpTextStyle.Render("details"),
		hotkeyStyle.Render("o"), helpTextStyle.Render("open"),
		hotkeyStyle.Render("r"), helpTextStyle.Render("refresh"),
		hotkeyStyle.Render("f"), helpTextStyle.Render("filter"),
		hotkeyStyle.Render("tab"), helpTextStyle.Render("switch"),
		hotkeyStyle.Render("c"), helpTextStyle.Render("comment"),
		hotkeyStyle.Render("x"), helpTextStyle.Render("close/reopen"),
		hotkeyStyle.Render("q"), helpTextStyle.Render("quit"),
	)
	if m.showDetail {
		help = fmt.Sprintf(
			"%s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s",
			hotkeyStyle.Render("esc"), helpTextStyle.Render("back"),
			hotkeyStyle.Render("o"), helpTextStyle.Render("open"),
			hotkeyStyle.Render("r"), helpTextStyle.Render("refresh"),
			hotkeyStyle.Render("c"), helpTextStyle.Render("comment"),
			hotkeyStyle.Render("x"), helpTextStyle.Render("close/reopen"),
			hotkeyStyle.Render("n/p"), helpTextStyle.Render("comments"),
			hotkeyStyle.Render("q"), helpTextStyle.Render("quit"),
		)
	}
	if m.commentMode {
		help = fmt.Sprintf(
			"%s %s  %s %s",
			hotkeyStyle.Render("ctrl+g"), helpTextStyle.Render("send"),
			hotkeyStyle.Render("esc"), helpTextStyle.Render("cancel"),
		)
	}
	if m.confirmMode {
		help = fmt.Sprintf(
			"%s %s  %s %s",
			hotkeyStyle.Render("y"), helpTextStyle.Render("confirm"),
			hotkeyStyle.Render("n/esc"), helpTextStyle.Render("cancel"),
		)
	}
	status := m.styles.Status.Render(m.status)
	if strings.HasPrefix(m.status, "Error:") {
		status = m.styles.StatusErr.Render(m.status)
	}
	if m.loading || m.detailLoading || m.actionLoading {
		status = fmt.Sprintf("%s %s", m.spinner.View(), status)
	}
	if !m.statusOverride && !m.lastUpdated.IsZero() && !m.loading && m.err == nil {
		status = m.styles.Status.Render(fmt.Sprintf("Loaded %d items • updated %s", len(m.list.Items()), humanizeSince(m.lastUpdated)))
	}
	footer := fmt.Sprintf("%s\n%s", help, status)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		tabsLine,
		body,
		footer,
	)
}

func (m model) detailView() string {
	if m.detailLoading {
		return fmt.Sprintf("%s %s", m.spinner.View(), m.styles.Status.Render("Loading details..."))
	}
	if m.detailErr != nil {
		return m.styles.StatusErr.Render(fmt.Sprintf("Error loading details: %s", m.detailErr.Error()))
	}
	if m.detailItem.Title == "" {
		return m.styles.MutedText.Render("No details loaded.")
	}

	info := fmt.Sprintf(
		"%s • %s • #%d • %s • %d comments • updated %s",
		m.detailItem.Repo,
		m.detailItem.Kind,
		m.detailItem.Number,
		m.detailItem.State,
		m.detailItem.Comments,
		humanizeSince(m.detailItem.Updated),
	)
	extra := ""
	switch m.detailItem.Kind {
	case "PR":
		draft := "ready"
		if m.detailItem.Draft {
			draft = "draft"
		}
		mergeable := formatMergeable(m.detailItem.Mergeable)
		reviews := fmt.Sprintf("reviews: +%d / -%d / %d",
			m.detailItem.ReviewApprovals,
			m.detailItem.ReviewChanges,
			m.detailItem.ReviewComments,
		)
		extra = fmt.Sprintf("state: %s • mergeable: %s • %s • +%d/-%d • files %d • commits %d",
			draft,
			mergeable,
			reviews,
			m.detailItem.Additions,
			m.detailItem.Deletions,
			m.detailItem.ChangedFiles,
			m.detailItem.Commits,
		)
	case "Issue":
		extra = fmt.Sprintf("labels: %s • assignees: %s",
			formatList(m.detailItem.Labels),
			formatList(m.detailItem.Assignees),
		)
	}

	titleStyle := m.styles.AccentText
	bodyStyle := m.styles.BodyText.Copy().Width(m.width - 2)
	comments := renderComments(m.detailItem.CommentList, m.detailItem.CommentPage, m.detailItem.HasNextComments, m.detailItem.HasPrevComments, m.width-2, m.styles)
	metaLine := m.styles.MetaText.Render(info)
	if extra != "" {
		metaLine = metaLine + "\n" + m.styles.MutedText.Render(extra)
	}
	content := fmt.Sprintf("%s\n%s\n\n%s\n\n%s",
		titleStyle.Render(m.detailItem.Title),
		metaLine,
		bodyStyle.Render(strings.TrimSpace(m.detailItem.Body)),
		comments,
	)

	return content
}

func (m model) commentView() string {
	title := m.styles.AccentText.Render("New Comment")
	info := m.styles.MetaText.Render(fmt.Sprintf("%s • #%d", m.actionItem.Repo, m.actionItem.Number))
	return fmt.Sprintf("%s\n%s\n\n%s", title, info, m.textarea.View())
}

func (m model) confirmView() string {
	actionText := actionLabel(m.confirmTargetState)
	target := fmt.Sprintf("%s • #%d", m.actionItem.Repo, m.actionItem.Number)
	prompt := fmt.Sprintf("%s this %s?", actionText, strings.ToLower(m.actionItem.Kind))
	content := fmt.Sprintf("%s\n%s", prompt, target)
	return m.styles.Confirm.Render(content)
}
