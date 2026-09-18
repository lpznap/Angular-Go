package domain

import (
	"errors"
	"github.com/microcosm-cc/bluemonday"
	"strings"
	"time"
)

var ErrConflict = errors.New("This note changed elsewhere. Reload it before saving; your draft is preserved.")
var ErrNotFound = errors.New("Resource not found")

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type Task struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}
type Note struct {
	ID          string    `json:"id"`
	WorkDate    string    `json:"workDate"`
	Title       string    `json:"title"`
	Project     string    `json:"project"`
	Description string    `json:"description"`
	Tasks       []Task    `json:"tasks"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	Tags        []string  `json:"tags"`
	Minutes     int32     `json:"minutes"`
	Blockers    string    `json:"blockers"`
	NextSteps   string    `json:"nextSteps"`
	Version     int32     `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type Attachment struct {
	ID          string `json:"id"`
	NoteID      string `json:"noteId"`
	Name        string `json:"name"`
	MIME        string `json:"mime"`
	Size        int64  `json:"size"`
	StorageKey  string `json:"-"`
	ArchivePath string `json:"archivePath,omitempty"`
}
type Filter struct {
	Search, From, To, Project, Status, Priority, Tag, Sort string
	Page, Size, Offset                                     int32
}

var richPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "em", "u", "s", "ul", "ol", "li", "blockquote", "h2", "h3", "pre", "code")
	return p
}()

func Sanitize(s string) string { return richPolicy.Sanitize(s) }
func Plain(s string) string    { return bluemonday.StrictPolicy().Sanitize(s) }
func (n *Note) Validate() error {
	n.Title = strings.TrimSpace(n.Title)
	if _, err := time.Parse("2006-01-02", n.WorkDate); err != nil {
		return ValidationError{"A valid work date is required"}
	}
	if len(n.Title) == 0 || len(n.Title) > 300 {
		return ValidationError{"Title must contain 1–300 bytes"}
	}
	if !strings.Contains("|todo|progress|completed|blocked|", "|"+n.Status+"|") || n.Status == "" {
		return ValidationError{"Invalid status"}
	}
	if n.Priority != "low" && n.Priority != "medium" && n.Priority != "high" {
		return ValidationError{"Invalid priority"}
	}
	if n.Minutes < 0 || n.Minutes > 1440 {
		return ValidationError{"Time must be between 0 and 24 hours"}
	}
	if len(n.Description) > 100000 || len(n.Blockers) > 10000 || len(n.NextSteps) > 10000 || len(n.Project) > 200 || len(n.Tasks) > 100 || len(n.Tags) > 30 {
		return ValidationError{"Note content exceeds limits"}
	}
	for _, t := range n.Tasks {
		if len(t.Text) > 1000 {
			return ValidationError{"Task exceeds 1000 bytes"}
		}
	}
	for _, t := range n.Tags {
		if len(t) > 80 {
			return ValidationError{"Tag exceeds 80 bytes"}
		}
	}
	n.Description = Sanitize(n.Description)
	if n.Tasks == nil {
		n.Tasks = []Task{}
	}
	if n.Tags == nil {
		n.Tags = []string{}
	}
	return nil
}
