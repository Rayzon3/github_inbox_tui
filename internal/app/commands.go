package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func fetchCmd(f filter, kind string) tea.Cmd {
	return func() tea.Msg {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fetchResult{err: errors.New("GITHUB_TOKEN is required")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		query := applyTabQuery(f.Query, kind)
		items, err := fetchIssuesWithFallback(ctx, token, query)
		return fetchResult{items: items, err: err}
	}
}

func fetchDetailCmd(item issueItem, commentPage int) tea.Cmd {
	return func() tea.Msg {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return detailResult{err: errors.New("GITHUB_TOKEN is required")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		result, err := fetchIssueDetail(ctx, token, item, commentPage)
		return detailResult{item: result, err: err}
	}
}

func postCommentCmd(item issueItem, body string) tea.Cmd {
	return func() tea.Msg {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return commentResult{err: errors.New("GITHUB_TOKEN is required")}
		}
		if item.Repo == "" || item.Number == 0 {
			return commentResult{err: errors.New("missing repo or number")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		payload := map[string]string{"body": body}
		raw, err := json.Marshal(payload)
		if err != nil {
			return commentResult{err: err}
		}

		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", item.Repo, item.Number)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(raw)))
		if err != nil {
			return commentResult{err: err}
		}
		addJSONHeaders(req, token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return commentResult{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return commentResult{err: readAPIError(resp)}
		}

		return commentResult{}
	}
}

func updateIssueStateCmd(item issueItem, state string) tea.Cmd {
	return func() tea.Msg {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return stateResult{err: errors.New("GITHUB_TOKEN is required")}
		}
		if item.Repo == "" || item.Number == 0 {
			return stateResult{err: errors.New("missing repo or number")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		payload := map[string]string{"state": state}
		raw, err := json.Marshal(payload)
		if err != nil {
			return stateResult{err: err}
		}

		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", item.Repo, item.Number)
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, apiURL, strings.NewReader(string(raw)))
		if err != nil {
			return stateResult{err: err}
		}
		addJSONHeaders(req, token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return stateResult{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return stateResult{err: readAPIError(resp)}
		}

		return stateResult{state: state}
	}
}

func openURLCmd(target string) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return nil
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", target)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
		default:
			cmd = exec.Command("xdg-open", target)
		}
		_ = cmd.Start()
		return nil
	}
}
