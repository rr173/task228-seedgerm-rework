package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug01ImageRetryWithOlderTimestampRemainsIdempotent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R1", "retry", "wheat")
	_, _ = svc.TransitionTrial(tr.ID, model.TrialRunning)
	seed, _ := svc.CreateSeed(tr.ID, "S1")
	at := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	first, created, err := svc.IngestImage(seed.ID, "same-hash", at, 640, 480, "first")
	if err != nil || !created { t.Fatalf("first ingest: created=%v err=%v", created, err) }
	retry, created, err := svc.IngestImage(seed.ID, "same-hash", at.Add(-time.Hour), 640, 480, "revised note")
	if err != nil || created { t.Fatalf("retry should be idempotent: created=%v err=%v", created, err) }
	if retry.ID != first.ID || retry.Note != "first" { t.Fatalf("identity changed: first=%+v retry=%+v", first, retry) }
}
