package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

const writeStoryStmt = `
insert into stories (story_id, api_version, queue_name, fetched_at, raw_document)
values ($1, $2, $3, $4, $5)
on conflict do nothing
returning id
`

const writeCommentStmt = `
insert into comments (internal_story_id, comment_id, raw_document)
values ($1, $2, $3)
`

type CommentModel struct {
	CommentID   int64
	RawDocument string
}

type StoryModel struct {
	StoryID     int64
	APIVersion  string
	QueueName   string
	FetchedAt   time.Time
	RawDocument string
	Comments    []CommentModel
}

// Repo provides access to a persistent data store for News stories and
// comments.
type Repo struct {
	conn *pgx.Conn
}

func NewRepo(conn *pgx.Conn) *Repo {
	return &Repo{conn: conn}
}

// WriteStory writes a story and its comments.
func (r *Repo) WriteStory(ctx context.Context, story StoryModel) error {
	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var id int32
	row := tx.QueryRow(ctx, writeStoryStmt, story.StoryID, story.APIVersion, story.QueueName, story.FetchedAt, story.RawDocument)
	err = row.Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		slog.Error("Skipping insert of duplicate story", "story_id", story.StoryID, "error", err)
		return nil
	}
	if err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for _, comment := range story.Comments {
		batch.Queue(writeCommentStmt, id, comment.CommentID, comment.RawDocument)
	}

	err = r.conn.SendBatch(ctx, batch).Close()
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
