package ipc

import (
	"bufio"
	"chronosync.local/engine/internal/contract"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/recorder"
	"chronosync.local/engine/internal/snapshot"
	"chronosync.local/engine/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
)

const MaxFrame = 1024 * 1024

type Request struct {
	ProtocolVersion int             `json:"protocol_version"`
	ID              string          `json:"id"`
	Method          string          `json:"method"`
	Params          json.RawMessage `json:"params"`
}
type RPCError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}
type Response struct {
	ProtocolVersion int       `json:"protocol_version"`
	ID              string    `json:"id"`
	Result          any       `json:"result,omitempty"`
	Error           *RPCError `json:"error,omitempty"`
}
type Page struct {
	Run      model.Run     `json:"run"`
	Events   []model.Event `json:"events"`
	Warnings []string      `json:"warnings"`
	Next     string        `json:"next"`
}

func Serve(in io.Reader, out io.Writer, r *recorder.Recorder, s storage.Inspector, snapshots ...*snapshot.Service) error {
	defer r.Close()
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), MaxFrame)
	enc := json.NewEncoder(out)
	hello := false
	for scan.Scan() {
		var req Request
		resp := Response{ProtocolVersion: 1}
		if json.Unmarshal(scan.Bytes(), &req) != nil || req.ID == "" || len(req.ID) > 128 || req.ProtocolVersion != 1 {
			resp.Error = &RPCError{"PROTOCOL_ERROR", "Invalid v1 request", false}
		} else {
			resp.ID = req.ID
			var err error
			if !hello && req.Method != "hello" {
				err = errors.New("hello required before requests")
			} else {
				resp.Result, err = dispatch(req, r, s, snapshots...)
				if err == nil && req.Method == "hello" {
					hello = true
				}
			}
			if err != nil {
				resp.Error = &RPCError{"REQUEST_FAILED", err.Error(), false}
				var snapshotError *snapshot.Error
				if errors.As(err, &snapshotError) {
					resp.Error.Code = snapshotError.Code
					resp.Error.Retryable = snapshotError.Code == "BUSY"
				}
				var storageError *storage.Error
				if errors.As(err, &storageError) {
					resp.Error.Code = storageError.Code
					resp.Error.Retryable = storageError.Code == "STORE_BUSY"
				}
			}
		}
		b, e := json.Marshal(resp)
		if e != nil {
			return e
		}
		if len(b) > MaxFrame {
			return errors.New("response exceeded frame limit")
		}
		if e = enc.Encode(resp); e != nil {
			return e
		}
	}
	return scan.Err()
}
func dispatch(req Request, r *recorder.Recorder, s storage.Inspector, snapshots ...*snapshot.Service) (any, error) {
	if strings.HasPrefix(req.Method, "snapshot.") || strings.HasPrefix(req.Method, "branch.") || req.Method == "object.inventory" {
		if len(snapshots) == 0 {
			return nil, errors.New("snapshot service unavailable")
		}
		return snapshotDispatch(req, r, snapshots[0])
	}
	switch req.Method {
	case "hello":
		return contract.CurrentVersion(), nil
	case "run.start":
		var spec model.StartSpec
		if json.Unmarshal(req.Params, &spec) != nil {
			return nil, errors.New("invalid run specification")
		}
		return r.Start(spec)
	case "run.list":
		runs, err := s.List()
		if len(runs) > 100 {
			runs = runs[:100]
		}
		return runs, err
	case "run.query":
		var q storage.RunQuery
		if json.Unmarshal(req.Params, &q) != nil {
			return nil, errors.New("invalid run query")
		}
		return s.QueryRuns(context.Background(), q)
	case "event.query":
		var q storage.EventQuery
		if json.Unmarshal(req.Params, &q) != nil {
			return nil, errors.New("invalid event query")
		}
		return s.QueryEvents(context.Background(), q)
	case "event.get", "run.get":
		var q struct {
			ID      string `json:"id"`
			EventID string `json:"event_id"`
		}
		if json.Unmarshal(req.Params, &q) != nil {
			return nil, errors.New("invalid identity")
		}
		if req.Method == "run.get" {
			return s.GetRun(context.Background(), q.ID)
		}
		return s.GetEvent(context.Background(), q.ID, q.EventID)
	case "run.cancel":
		var p struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(req.Params, &p) != nil {
			return nil, errors.New("invalid cancellation")
		}
		err := r.Cancel(p.ID)
		return map[string]bool{"accepted": err == nil}, err
	case "event.list":
		var p struct {
			ID    string `json:"id"`
			After string `json:"after"`
		}
		if json.Unmarshal(req.Params, &p) != nil {
			return nil, errors.New("invalid page request")
		}
		after, err := strconv.ParseInt(p.After, 10, 64)
		if err != nil || after < 0 {
			return nil, errors.New("invalid cursor")
		}
		run, err := s.GetRun(context.Background(), p.ID)
		if err != nil {
			return nil, err
		}
		page := Page{Run: run, Events: []model.Event{}, Warnings: []string{}, Next: p.After}
		if run.History != "complete" {
			page.Warnings = append(page.Warnings, run.History+": "+run.Diagnostic)
		}
		summaries, err := s.QueryEvents(context.Background(), storage.EventQuery{ID: p.ID, Min: strconv.FormatInt(after+1, 10), Limit: 100})
		if err != nil {
			return nil, err
		}
		bytes := 0
		for _, summary := range summaries.Events {
			event, e := s.GetEvent(context.Background(), p.ID, summary.EventID)
			if e != nil {
				page.Warnings = append(page.Warnings, e.Error())
				page.Next = summary.Sequence
				continue
			}
			b, _ := json.Marshal(event)
			if bytes+len(b) > 512*1024 {
				break
			}
			bytes += len(b)
			page.Events = append(page.Events, event)
			page.Next = event.Sequence
		}
		if err = r.Error(); err != nil {
			page.Warnings = append(page.Warnings, err.Error())
		}
		return page, nil
	default:
		return nil, errors.New("unsupported method")
	}
}
