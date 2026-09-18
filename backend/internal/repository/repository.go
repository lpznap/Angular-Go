package repository

import (
	"context"
	"dailyworknotes/internal/domain"
	"dailyworknotes/internal/repository/db"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Repository struct {
	Pool *pgxpool.Pool
	Q    *db.Queries
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool, db.New(pool)} }
func Date(s string) pgtype.Date {
	t, _ := time.Parse("2006-01-02", s)
	return pgtype.Date{Time: t, Valid: true}
}
func Convert(n db.Note) domain.Note {
	tasks := []domain.Task{}
	_ = json.Unmarshal(n.Tasks, &tasks)
	return domain.Note{ID: n.ID, WorkDate: n.WorkDate.Time.Format("2006-01-02"), Title: n.Title, Project: n.Project, Description: n.Description, Tasks: tasks, Status: n.Status, Priority: n.Priority, Tags: n.Tags, Minutes: n.Minutes, Blockers: n.Blockers, NextSteps: n.NextSteps, Version: n.Version, CreatedAt: n.CreatedAt.Time, UpdatedAt: n.UpdatedAt.Time}
}
func (r *Repository) Get(ctx context.Context, user, id string) (domain.Note, error) {
	n, e := r.Q.GetNote(ctx, db.GetNoteParams{ID: id, UserID: user})
	if errors.Is(e, pgx.ErrNoRows) {
		e = domain.ErrNotFound
	}
	return Convert(n), e
}
func Create(ctx context.Context, q *db.Queries, user string, n domain.Note) (domain.Note, error) {
	tasks, _ := json.Marshal(n.Tasks)
	row, e := q.CreateNote(ctx, db.CreateNoteParams{ID: n.ID, UserID: user, WorkDate: Date(n.WorkDate), Title: n.Title, Project: n.Project, Description: n.Description, Tasks: tasks, Status: n.Status, Priority: n.Priority, Tags: n.Tags, Minutes: n.Minutes, Blockers: n.Blockers, NextSteps: n.NextSteps})
	return Convert(row), e
}
func Update(ctx context.Context, q *db.Queries, user string, n domain.Note) (domain.Note, error) {
	tasks, _ := json.Marshal(n.Tasks)
	row, e := q.UpdateNote(ctx, db.UpdateNoteParams{ID: n.ID, UserID: user, WorkDate: Date(n.WorkDate), Title: n.Title, Project: n.Project, Description: n.Description, Tasks: tasks, Status: n.Status, Priority: n.Priority, Tags: n.Tags, Minutes: n.Minutes, Blockers: n.Blockers, NextSteps: n.NextSteps, Version: n.Version})
	if errors.Is(e, pgx.ErrNoRows) {
		e = domain.ErrConflict
	}
	return Convert(row), e
}
func (r *Repository) List(ctx context.Context, user string, f domain.Filter) ([]domain.Note, error) {
	offset := (f.Page - 1) * f.Size
	if f.Offset > 0 { offset = f.Offset }
	rows, e := r.Q.ListNotes(ctx, db.ListNotesParams{Owner: user, Search: f.Search, FromDate: f.From, ToDate: f.To, ProjectFilter: f.Project, StatusFilter: f.Status, PriorityFilter: f.Priority, TagFilter: f.Tag, SortBy: f.Sort, PageSize: f.Size, PageOffset: offset})
	out := []domain.Note{}
	for _, n := range rows {
		out = append(out, Convert(n))
	}
	return out, e
}
func (r *Repository) Attachments(ctx context.Context, user, note string) ([]domain.Attachment, error) {
	rows, e := r.Pool.Query(ctx, "SELECT id,note_id,name,mime,size,storage_key FROM attachments WHERE user_id=$1 AND ($2='' OR note_id=$2) ORDER BY created_at", user, note)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Attachment{}
	for rows.Next() {
		var a domain.Attachment
		if e = rows.Scan(&a.ID, &a.NoteID, &a.Name, &a.MIME, &a.Size, &a.StorageKey); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
