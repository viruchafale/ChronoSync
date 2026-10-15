package storage

import (
	"chronosync.local/engine/internal/model"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type EventQuery struct {
	ID     string `json:"id"`
	Cursor string `json:"cursor"`
	Kind   string `json:"kind"`
	Search string `json:"search"`
	Min    string `json:"min"`
	Max    string `json:"max"`
	Limit  int    `json:"limit"`
}
type EventSummary struct {
	EventID    string `json:"event_id"`
	Sequence   string `json:"sequence"`
	Kind       string `json:"kind"`
	ObservedAt string `json:"observed_at"`
	Status     string `json:"status"`
	Truncated  bool   `json:"truncated"`
}
type EventPage struct {
	Run       model.Run      `json:"run"`
	Events    []EventSummary `json:"events"`
	Next      string         `json:"next"`
	HighWater string         `json:"high_water"`
	Count     int64          `json:"count"`
}
type eventCursor struct {
	Version int
	ID      string
	After   int64
	High    int64
	Filter  string
}

func sortTime(value string) string {
	t, e := time.Parse(time.RFC3339Nano, value)
	if e != nil {
		return value
	}
	return t.UTC().Format("2006-01-02T15:04:05.000000000Z")
}
func encode(v any) string { b, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(b) }
func decode(s string, v any) error {
	if len(s) > 2048 {
		return invalid("cursor too long")
	}
	b, e := base64.RawURLEncoding.DecodeString(s)
	if e != nil || json.Unmarshal(b, v) != nil {
		return invalid("malformed cursor")
	}
	return nil
}
func seq(s string, defaultValue int64) (int64, error) {
	if s == "" {
		return defaultValue, nil
	}
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || n < 0 || strconv.FormatInt(n, 10) != s {
		return 0, invalid("sequence range")
	}
	return n, nil
}
func pageLimit(n int) (int, error) {
	if n == 0 {
		return 100, nil
	}
	if n < 1 || n > 100 {
		return 0, invalid("page limit 1..100")
	}
	return n, nil
}
func (s *Store) QueryEvents(ctx context.Context, q EventQuery) (EventPage, error) {
	p := EventPage{Events: []EventSummary{}}
	limit, err := pageLimit(q.Limit)
	if err != nil {
		return p, err
	}
	if q.Kind != "" && !model.Kinds[q.Kind] || len(q.Search) > 256 || strings.IndexByte(q.Search, 0) >= 0 {
		return p, invalid("event filter")
	}
	min, err := seq(q.Min, 1)
	if err != nil {
		return p, err
	}
	max, err := seq(q.Max, 9223372036854775807)
	if err != nil || min > max {
		return p, invalid("sequence range")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	p.Run, err = s.GetRun(ctx, q.ID)
	if err != nil {
		return p, err
	}
	high, _ := strconv.ParseInt(p.Run.LastSequence, 10, 64)
	hash := sha256.Sum256([]byte(q.Kind + "\x00" + q.Search + "\x00" + strconv.FormatInt(min, 10) + "\x00" + strconv.FormatInt(max, 10)))
	filter := hex.EncodeToString(hash[:])
	c := eventCursor{Version: 1, ID: q.ID, High: high, Filter: filter}
	if q.Cursor != "" {
		if err = decode(q.Cursor, &c); err != nil {
			return p, err
		}
		if c.Version != 1 || c.ID != q.ID || c.Filter != filter || c.After < 0 || c.High < 0 || c.After > c.High || c.High > high {
			return p, invalid("cursor scope")
		}
	}
	p.HighWater = strconv.FormatInt(c.High, 10)
	where := ` FROM events WHERE run_id=? AND sequence>? AND sequence<=? AND sequence>=? AND sequence<=?`
	args := []any{q.ID, c.After, c.High, min, max}
	if q.Kind != "" {
		where += ` AND kind=?`
		args = append(args, q.Kind)
	}
	if q.Search != "" {
		where += ` AND instr(lower(search),lower(?))>0`
		args = append(args, q.Search)
	}
	// Total recorded count comes from the indexed run relationship, not payloads.
	if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM events WHERE run_id=? AND sequence<=?`, q.ID, c.High).Scan(&p.Count); err != nil {
		return p, failure("event count", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,sequence,kind,observed,status,truncated`+where+` ORDER BY sequence LIMIT ?`, append(args, limit+1)...)
	if err != nil {
		return p, failure("event query", err)
	}
	defer rows.Close()
	for rows.Next() {
		var e EventSummary
		if err = rows.Scan(&e.EventID, &e.Sequence, &e.Kind, &e.ObservedAt, &e.Status, &e.Truncated); err != nil {
			return p, failure("event row", err)
		}
		if !model.ValidID(e.EventID) || !model.Kinds[e.Kind] {
			return p, &Error{"CORRUPT_METADATA", "event index"}
		}
		if _, err := seq(e.Sequence, -1); err != nil {
			return p, &Error{"CORRUPT_METADATA", "event sequence"}
		}
		if _, err := time.Parse(time.RFC3339Nano, e.ObservedAt); err != nil {
			return p, &Error{"CORRUPT_METADATA", "event timestamp"}
		}
		if len(p.Events) == limit {
			last, _ := strconv.ParseInt(p.Events[len(p.Events)-1].Sequence, 10, 64)
			c.After = last
			p.Next = encode(c)
			break
		}
		p.Events = append(p.Events, e)
	}
	return p, failure("event query", rows.Err())
}

type RunQuery struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}
type RunPage struct {
	Runs []model.Run `json:"runs"`
	Next string      `json:"next"`
}
type runCursor struct {
	Version int
	Created string
	ID      string
}

func (s *Store) QueryRuns(ctx context.Context, q RunQuery) (RunPage, error) {
	p := RunPage{Runs: []model.Run{}}
	limit, err := pageLimit(q.Limit)
	if err != nil {
		return p, err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	query := `SELECT id FROM runs`
	args := []any{}
	if q.Cursor != "" {
		var c runCursor
		if err = decode(q.Cursor, &c); err != nil {
			return p, err
		}
		if c.Version != 1 || !model.ValidID(c.ID) {
			return p, invalid("run cursor")
		}
		query += ` WHERE (created,id)<(?,?)`
		args = append(args, c.Created, c.ID)
	}
	query += ` ORDER BY created DESC,id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return p, failure("run query", err)
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return p, failure("run row", err)
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return p, failure("run query", err)
	}
	for i, id := range ids {
		if i == limit {
			last := p.Runs[len(p.Runs)-1]
			p.Next = encode(runCursor{1, sortTime(last.CreatedAt), last.ID})
			break
		}
		r, e := s.GetRun(ctx, id)
		if e != nil {
			return p, e
		}
		p.Runs = append(p.Runs, r)
	}
	return p, nil
}
