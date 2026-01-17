package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type uiStyles struct {
	Title      lipgloss.Style
	Filter     lipgloss.Style
	HelpKey    lipgloss.Style
	HelpText   lipgloss.Style
	Status     lipgloss.Style
	StatusErr  lipgloss.Style
	BodyText   lipgloss.Style
	MetaText   lipgloss.Style
	MutedText  lipgloss.Style
	AccentText lipgloss.Style
	TreeLine   lipgloss.Style
	Panel      lipgloss.Style
	Confirm    lipgloss.Style
}

func newStyles() uiStyles {
	return uiStyles{
		Title:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1e66f5")),
		Filter:     lipgloss.NewStyle().Foreground(lipgloss.Color("#8839ef")).Bold(true),
		HelpKey:    lipgloss.NewStyle().Foreground(lipgloss.Color("#1e66f5")).Bold(true),
		HelpText:   lipgloss.NewStyle().Foreground(lipgloss.Color("#6c6f85")),
		Status:     lipgloss.NewStyle().Foreground(lipgloss.Color("#4c4f69")),
		StatusErr:  lipgloss.NewStyle().Foreground(lipgloss.Color("#d20f39")).Bold(true),
		BodyText:   lipgloss.NewStyle().Foreground(lipgloss.Color("#4c4f69")),
		MetaText:   lipgloss.NewStyle().Foreground(lipgloss.Color("#179299")).Bold(true),
		MutedText:  lipgloss.NewStyle().Foreground(lipgloss.Color("#8c8fa1")),
		AccentText: lipgloss.NewStyle().Foreground(lipgloss.Color("#ea76cb")).Bold(true),
		TreeLine:   lipgloss.NewStyle().Foreground(lipgloss.Color("#bcc0cc")),
		Panel:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#bcc0cc")).Padding(0, 1),
		Confirm:    lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.Color("#df8e1d")).Padding(1, 2),
	}
}

func initList() list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#1e66f5")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#5c5f77"))
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("#4c4f69"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(lipgloss.Color("#8c8fa1"))
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.Foreground(lipgloss.Color("#9ca0b0"))
	delegate.Styles.DimmedDesc = delegate.Styles.DimmedDesc.Foreground(lipgloss.Color("#9ca0b0"))

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowPagination(true)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c6f85"))
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca0b0"))
	return l
}

func renderTabs(tabList []tab, active int, styles uiStyles) string {
	parts := make([]string, 0, len(tabList))
	for i, t := range tabList {
		label := " " + t.Name + " "
		if i == active {
			parts = append(parts, styles.AccentText.Render(label))
		} else {
			parts = append(parts, styles.MutedText.Render(label))
		}
	}
	return strings.Join(parts, " ")
}

func renderComments(comments []issueComment, page int, hasNext, hasPrev bool, width int, styles uiStyles) string {
	if len(comments) == 0 {
		return fmt.Sprintf("%s\n  %s", styles.AccentText.Render("Comments"), styles.MutedText.Render("(no comments)"))
	}
	builder := strings.Builder{}
	bodyStyle := styles.BodyText.Copy().Width(width - 4)
	builder.WriteString(styles.AccentText.Render(fmt.Sprintf("Comments (page %d)", page)))
	for _, c := range comments {
		builder.WriteString("\n")
		builder.WriteString(styles.TreeLine.Render("|- "))
		builder.WriteString(styles.MetaText.Render(c.Author))
		builder.WriteString(" ")
		builder.WriteString(styles.MutedText.Render("• " + humanizeSince(c.Updated)))
		body := strings.TrimSpace(c.Body)
		if body == "" {
			body = "(empty)"
		}
		bodyRendered := bodyStyle.Render(body)
		builder.WriteString("\n")
		builder.WriteString(prefixLines(bodyRendered, styles.TreeLine.Render("|  ")))
	}
	if hasPrev || hasNext {
		builder.WriteString("\n|")
		if hasPrev {
			builder.WriteString(styles.MutedText.Render("  prev: p"))
		}
		if hasNext {
			if hasPrev {
				builder.WriteString("  ")
			}
			builder.WriteString(styles.MutedText.Render("next: n"))
		}
	}
	return builder.String()
}

func prefixLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func hasLinkRel(linkHeader, rel string) bool {
	if linkHeader == "" {
		return false
	}
	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		if strings.Contains(part, "rel=\""+rel+"\"") {
			return true
		}
	}
	return false
}

func humanizeSince(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func actionLabel(targetState string) string {
	if targetState == "closed" {
		return "Close"
	}
	return "Reopen"
}

func actionProgress(targetState string) string {
	if targetState == "closed" {
		return "Closing..."
	}
	return "Reopening..."
}

func formatList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func formatMergeable(val *bool) string {
	if val == nil {
		return "unknown"
	}
	if *val {
		return "yes"
	}
	return "no"
}
