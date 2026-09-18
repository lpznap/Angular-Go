package service

import (
	"context"
	"crypto/rand"
	"dailyworknotes/internal/domain"
	"dailyworknotes/internal/repository"
	"dailyworknotes/internal/repository/db"
	"encoding/hex"
)

func ID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}

type Notes struct{ Repo *repository.Repository }

func (s *Notes) Save(ctx context.Context, user string, n domain.Note, create bool) (domain.Note, error) {
	if e := n.Validate(); e != nil {
		return n, e
	}
	if create {
		n.ID = ID()
		return repository.Create(ctx, s.Repo.Q, user, n)
	}
	if _, e := s.Repo.Get(ctx, user, n.ID); e != nil {
		return n, e
	}
	return repository.Update(ctx, s.Repo.Q, user, n)
}
func (s *Notes) Delete(ctx context.Context, user, id string, version int32) error {
	tx, e := s.Repo.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, "INSERT INTO file_deletions(storage_key) SELECT storage_key FROM attachments WHERE note_id=$1 AND user_id=$2 ON CONFLICT DO NOTHING", id, user)
	if e != nil {
		return e
	}
	count, e := db.New(tx).DeleteNote(ctx, db.DeleteNoteParams{ID: id, UserID: user, Version: version})
	if e != nil {
		return e
	}
	if count == 0 {
		return domain.ErrConflict
	}
	return tx.Commit(ctx)
}
