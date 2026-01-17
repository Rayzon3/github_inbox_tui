package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

func fetchIssuesWithFallback(ctx context.Context, token, query string) ([]issueItem, error) {
	normalized := strings.ToLower(query)
	if strings.Contains(normalized, "is:issue") || strings.Contains(normalized, "is:pr") || strings.Contains(normalized, "is:pull-request") {
		return fetchIssues(ctx, token, query)
	}

	issueItems, issueErr := fetchIssues(ctx, token, query+" is:issue")
	if issueErr != nil {
		return nil, issueErr
	}
	prItems, prErr := fetchIssues(ctx, token, query+" is:pr")
	if prErr != nil {
		return nil, prErr
	}

	combined := make([]issueItem, 0, len(issueItems)+len(prItems))
	seen := make(map[string]struct{}, len(issueItems)+len(prItems))
	for _, item := range issueItems {
		combined = append(combined, item)
		seen[item.URL] = struct{}{}
	}
	for _, item := range prItems {
		if _, ok := seen[item.URL]; ok {
			continue
		}
		combined = append(combined, item)
	}

	return combined, nil
}

func applyTabQuery(query, kind string) string {
	kind = strings.ToLower(kind)
	if kind != "pr" && kind != "issue" {
		return query
	}
	tokens := strings.Fields(query)
	filtered := make([]string, 0, len(tokens)+1)
	for _, token := range tokens {
		lt := strings.ToLower(token)
		switch lt {
		case "is:pr", "is:issue", "is:pull-request", "is:pullrequest":
			continue
		default:
			filtered = append(filtered, token)
		}
	}
	if kind == "pr" {
		filtered = append(filtered, "is:pr")
	} else {
		filtered = append(filtered, "is:issue")
	}
	return strings.Join(filtered, " ")
}

func fetchIssues(ctx context.Context, token, query string) ([]issueItem, error) {
	endpoint, err := url.Parse("https://api.github.com/search/issues")
	if err != nil {
		return nil, err
	}

	params := endpoint.Query()
	params.Set("q", query)
	params.Set("per_page", fmt.Sprintf("%d", maxItems))
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	addJSONHeaders(req, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}

	var payload struct {
		Items []struct {
			Title         string    `json:"title"`
			Number        int       `json:"number"`
			HTMLURL       string    `json:"html_url"`
			RepositoryURL string    `json:"repository_url"`
			PullRequest   *struct{} `json:"pull_request"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	items := make([]issueItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		repo := repoNameFromAPIURL(item.RepositoryURL)
		kind := "Issue"
		if item.PullRequest != nil {
			kind = "PR"
		}
		items = append(items, issueItem{
			TitleText: item.Title,
			Repo:      repo,
			Number:    item.Number,
			URL:       item.HTMLURL,
			Kind:      kind,
		})
	}

	return items, nil
}

func fetchIssueDetail(ctx context.Context, token string, item issueItem, commentPage int) (detail, error) {
	if item.Repo == "" || item.Number == 0 {
		return detail{}, errors.New("missing repo or number")
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", item.Repo, item.Number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return detail{}, err
	}
	addJSONHeaders(req, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return detail{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return detail{}, readAPIError(resp)
	}

	var payload struct {
		Title     string    `json:"title"`
		Body      string    `json:"body"`
		State     string    `json:"state"`
		HTMLURL   string    `json:"html_url"`
		UpdatedAt time.Time `json:"updated_at"`
		Comments  int       `json:"comments"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Assignees []struct {
			Login string `json:"login"`
		} `json:"assignees"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		PullRequest *struct{} `json:"pull_request"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return detail{}, err
	}

	kind := "Issue"
	if payload.PullRequest != nil {
		kind = "PR"
	}

	comments, pageInfo, err := fetchIssueComments(ctx, token, item, commentPage)
	if err != nil {
		return detail{}, err
	}

	labels := make([]string, 0, len(payload.Labels))
	for _, l := range payload.Labels {
		if l.Name != "" {
			labels = append(labels, l.Name)
		}
	}
	assignees := make([]string, 0, len(payload.Assignees))
	for _, a := range payload.Assignees {
		if a.Login != "" {
			assignees = append(assignees, a.Login)
		}
	}

	var prMeta prDetail
	if kind == "PR" {
		prMeta, err = fetchPullRequestDetail(ctx, token, item)
		if err != nil {
			return detail{}, err
		}
	}

	return detail{
		Title:           payload.Title,
		Body:            payload.Body,
		State:           payload.State,
		Author:          payload.User.Login,
		Updated:         payload.UpdatedAt,
		Comments:        payload.Comments,
		URL:             payload.HTMLURL,
		Repo:            item.Repo,
		Number:          item.Number,
		Kind:            kind,
		Labels:          labels,
		Assignees:       assignees,
		Draft:           prMeta.Draft,
		Mergeable:       prMeta.Mergeable,
		Additions:       prMeta.Additions,
		Deletions:       prMeta.Deletions,
		ChangedFiles:    prMeta.ChangedFiles,
		Commits:         prMeta.Commits,
		ReviewApprovals: prMeta.ReviewApprovals,
		ReviewChanges:   prMeta.ReviewChanges,
		ReviewComments:  prMeta.ReviewComments,
		CommentList:     comments,
		CommentPage:     pageInfo.Page,
		HasNextComments: pageInfo.HasNext,
		HasPrevComments: pageInfo.HasPrev,
	}, nil
}

type prDetail struct {
	Draft           bool
	Mergeable       *bool
	Additions       int
	Deletions       int
	ChangedFiles    int
	Commits         int
	ReviewApprovals int
	ReviewChanges   int
	ReviewComments  int
}

func fetchPullRequestDetail(ctx context.Context, token string, item issueItem) (prDetail, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", item.Repo, item.Number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return prDetail{}, err
	}
	addJSONHeaders(req, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return prDetail{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return prDetail{}, readAPIError(resp)
	}

	var payload struct {
		Draft        bool  `json:"draft"`
		Mergeable    *bool `json:"mergeable"`
		Additions    int   `json:"additions"`
		Deletions    int   `json:"deletions"`
		ChangedFiles int   `json:"changed_files"`
		Commits      int   `json:"commits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return prDetail{}, err
	}

	reviews, err := fetchPullRequestReviews(ctx, token, item)
	if err != nil {
		return prDetail{}, err
	}

	return prDetail{
		Draft:           payload.Draft,
		Mergeable:       payload.Mergeable,
		Additions:       payload.Additions,
		Deletions:       payload.Deletions,
		ChangedFiles:    payload.ChangedFiles,
		Commits:         payload.Commits,
		ReviewApprovals: reviews.approvals,
		ReviewChanges:   reviews.changesRequested,
		ReviewComments:  reviews.commented,
	}, nil
}

type reviewSummary struct {
	approvals        int
	changesRequested int
	commented        int
}

func fetchPullRequestReviews(ctx context.Context, token string, item issueItem) (reviewSummary, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d/reviews", item.Repo, item.Number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return reviewSummary{}, err
	}
	addJSONHeaders(req, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return reviewSummary{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return reviewSummary{}, readAPIError(resp)
	}

	var payload []struct {
		State string `json:"state"`
		User  struct {
			Login string `json:"login"`
		} `json:"user"`
		SubmittedAt time.Time `json:"submitted_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return reviewSummary{}, err
	}

	latestByUser := make(map[string]string)
	for _, r := range payload {
		user := r.User.Login
		if user == "" {
			continue
		}
		latestByUser[user] = r.State
	}

	var summary reviewSummary
	for _, state := range latestByUser {
		switch strings.ToUpper(state) {
		case "APPROVED":
			summary.approvals++
		case "CHANGES_REQUESTED":
			summary.changesRequested++
		case "COMMENTED":
			summary.commented++
		}
	}

	return summary, nil
}

func fetchIssueComments(ctx context.Context, token string, item issueItem, page int) ([]issueComment, commentPageInfo, error) {
	if item.Repo == "" || item.Number == 0 {
		return nil, commentPageInfo{}, errors.New("missing repo or number")
	}
	endpoint, err := url.Parse(fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", item.Repo, item.Number))
	if err != nil {
		return nil, commentPageInfo{}, err
	}
	params := endpoint.Query()
	params.Set("per_page", fmt.Sprintf("%d", maxComments))
	params.Set("page", fmt.Sprintf("%d", page))
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, commentPageInfo{}, err
	}
	addJSONHeaders(req, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, commentPageInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, commentPageInfo{}, readAPIError(resp)
	}

	var payload []struct {
		Body      string    `json:"body"`
		UpdatedAt time.Time `json:"updated_at"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, commentPageInfo{}, err
	}

	comments := make([]issueComment, 0, len(payload))
	for _, c := range payload {
		comments = append(comments, issueComment{
			Author:  c.User.Login,
			Body:    c.Body,
			Updated: c.UpdatedAt,
		})
	}

	pageInfo := commentPageInfo{
		Page:    page,
		HasNext: hasLinkRel(resp.Header.Get("Link"), "next"),
		HasPrev: hasLinkRel(resp.Header.Get("Link"), "prev"),
	}

	return comments, pageInfo, nil
}

func repoNameFromAPIURL(apiURL string) string {
	if apiURL == "" {
		return "unknown/repo"
	}
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return "unknown/repo"
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return "unknown/repo"
	}
	return path.Join(parts[len(parts)-2], parts[len(parts)-1])
}

func addJSONHeaders(req *http.Request, token string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
}

func readAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	detail := strings.TrimSpace(string(body))
	if detail != "" {
		return fmt.Errorf("github api error: %s: %s", resp.Status, detail)
	}
	return fmt.Errorf("github api error: %s", resp.Status)
}
