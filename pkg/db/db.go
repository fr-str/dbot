package db

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"dbot/pkg/store"

	"github.com/fr-str/log"
	"modernc.org/sqlite"
	_ "modernc.org/sqlite"
)

func (s db) configure() {
	s.w.SetMaxOpenConns(1)
	s.w.SetConnMaxLifetime(0)
	s.w.SetConnMaxIdleTime(0)

	s.r.SetMaxOpenConns(100)
	s.r.SetConnMaxLifetime(5 * time.Minute)
	s.r.SetMaxIdleConns(2)
}

func Connect(ctx context.Context, filename string, schema string) (*store.Queries, error) {
	w, err := sql.Open("sqlite", filename)
	if err != nil {
		return nil, err
	}
	r, err := sql.Open("sqlite", filename)
	if err != nil {
		return nil, err
	}

	// create tables
	if _, err := w.ExecContext(ctx, schema); err != nil {
		return nil, err
	}

	d := db{
		w: w,
		r: r,
	}
	d.configure()
	return store.New(d), nil
}

type db struct {
	w *sql.DB
	r *sql.DB
}

func (s db) ExecContext(ctx context.Context, sql string, args ...any) (sql.Result, error) {
	ts := time.Now()
	res, err := s.w.ExecContext(ctx, sql, args...)
	logger(ctx, "ExecContext", sql, ts, args, res, err)
	return res, err
}

func (s db) PrepareContext(ctx context.Context, sql string) (*sql.Stmt, error) {
	ts := time.Now()
	stmt, err := s.w.PrepareContext(ctx, sql)
	logger(ctx, "PrepareContext", sql, ts, sql, nil, err)
	return stmt, err
}

func (s db) QueryContext(ctx context.Context, sql string, args ...any) (*sql.Rows, error) {
	ts := time.Now()
	rows, err := s.w.QueryContext(ctx, sql, args...)
	logger(ctx, "QueryContext", sql, ts, args, nil, err)
	return rows, err
}

func (s db) QueryRowContext(ctx context.Context, sql string, args ...any) *sql.Row {
	ts := time.Now()
	row := s.w.QueryRowContext(ctx, sql, args...)
	logger(ctx, "QueryRowContext", sql, ts, args, nil, nil)
	return row
}

// relaceConsecutiveSpaces replaces consecutive spaces with a single space
func relaceConsecutiveSpaces(s string) string {
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

func logger(ctx context.Context, info string, query string, ts time.Time, args any, res sql.Result, err error) {
	timeSince := time.Since(ts).String()
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.ReplaceAll(query, "\t", " ")
	query = relaceConsecutiveSpaces(query)
	meta := []any{
		log.String("query", query),
		log.Any("args", args),
		log.String("duration", timeSince),
	}

	if res != nil {
		rows, err := res.RowsAffected()
		if err != nil {
			meta = append(meta, log.String("rows_error", err.Error()))
			log.Error("Rows affected failed", meta...)
		}
		meta = append(meta, log.Int("rows", rows))
	}
	if err != nil {
		meta = append(meta, log.Err(err))
		e := &sqlite.Error{}
		if !errors.As(err, &e) {
			log.Error(fmt.Sprintf("%s failed", info), meta...)
			return
		}
	}
	log.InfoCtx(ctx, fmt.Sprintf("%s executed", info), meta...)
}