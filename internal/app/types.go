package app

import (
	"strconv"
	"time"
)

type filter struct {
	Name  string
	Query string
}

type tab struct {
	Name string
	Kind string
}

var filters = []filter{
	{Name: "Open", Query: "is:open archived:false involves:@me"},
	{Name: "Review requested", Query: "review-requested:@me"},
	{Name: "Assigned", Query: "assignee:@me"},
	{Name: "Mentions", Query: "mentions:@me"},
	{Name: "Authored", Query: "author:@me"},
}

var tabs = []tab{
	{Name: "PRs", Kind: "pr"},
	{Name: "Issues", Kind: "issue"},
}

const (
	maxItems    = 50
	maxComments = 10
)

type issueItem struct {
	TitleText string
	Repo      string
	Number    int
	URL       string
	Kind      string
}

func (i issueItem) Title() string { return i.TitleText }
func (i issueItem) Description() string {
	return i.Repo + " • #" + strconv.Itoa(i.Number) + " • " + i.Kind
}
func (i issueItem) FilterValue() string { return i.TitleText }

type fetchResult struct {
	items []issueItem
	err   error
}

type detail struct {
	Title           string
	Body            string
	State           string
	Author          string
	Updated         time.Time
	Comments        int
	URL             string
	Repo            string
	Number          int
	Kind            string
	Labels          []string
	Assignees       []string
	Draft           bool
	Mergeable       *bool
	Additions       int
	Deletions       int
	ChangedFiles    int
	Commits         int
	ReviewApprovals int
	ReviewChanges   int
	ReviewComments  int
	CommentList     []issueComment
	CommentPage     int
	HasNextComments bool
	HasPrevComments bool
}

type detailResult struct {
	item detail
	err  error
}

type issueComment struct {
	Author  string
	Body    string
	Updated time.Time
}

type commentResult struct {
	err error
}

type stateResult struct {
	state string
	err   error
}

type commentPageInfo struct {
	Page    int
	HasNext bool
	HasPrev bool
}
