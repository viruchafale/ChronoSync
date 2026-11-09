package exchange

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/governance"
)

type ReportStore interface {
	ExportCanonical(context.Context, string) ([]byte, error)
	ImportCanonical(context.Context, []byte) (analysis.Report, bool, error)
}

type BenchmarkStore interface {
	ExportDataset(context.Context, string, bool) ([]byte, governance.DatasetRecord, error)
	ImportDataset(context.Context, []byte) (governance.DatasetRecord, error)
	GetDataset(context.Context, string) (governance.Dataset, error)
	ListDatasets(context.Context) ([]governance.DatasetRecord, error)
}

type Service struct {
	root                *os.Root
	reports             ReportStore
	benchmarks          BenchmarkStore
	gate                chan struct{}
	sessions            map[string]exportSession
	migrationChallenges map[string][]byte
}

type exportSession struct {
	keyID       string
	identity    Identity
	manifestRaw []byte
	datasetRaw  []byte
	reports     map[string][]byte
	payload     []byte
	expires     time.Time
}

func New(base string, reports ReportStore, benchmarks BenchmarkStore) (*Service, error) {
	if !filepath.IsAbs(base) || reports == nil || benchmarks == nil {
		return nil, &Error{"INVALID_ARGUMENT", "exchange root or repositories"}
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "exchange root"}
	}
	directories := []string{"exchange", "exchange/v1", "exchange/v1/identities", "exchange/v1/identities/public", "exchange/v1/identities/private", "exchange/v1/trust", "exchange/v1/custody", "exchange/v1/migration"}
	for _, directory := range directories {
		if err = root.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			root.Close()
			return nil, &Error{"STORAGE_ERROR", "exchange directory"}
		}
		info, inspectErr := root.Lstat(directory)
		if inspectErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			root.Close()
			return nil, &Error{"UNSAFE_STORAGE", "exchange directory"}
		}
	}
	service := &Service{root: root, reports: reports, benchmarks: benchmarks, gate: make(chan struct{}, 1), sessions: map[string]exportSession{}, migrationChallenges: map[string][]byte{}}
	for _, directory := range []string{"exchange/v1/identities/public", "exchange/v1/identities/private", "exchange/v1/trust", "exchange/v1/custody", "exchange/v1/migration"} {
		if err = service.sync(directory); err != nil {
			root.Close()
			return nil, err
		}
	}
	return service, nil
}

func (s *Service) Close() error { return s.root.Close() }

func (s *Service) enter() error {
	select {
	case s.gate <- struct{}{}:
		return nil
	default:
		return &Error{"BUSY", "exchange writer"}
	}
}

func (s *Service) leave() { <-s.gate }
