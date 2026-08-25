package ingest_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/ingest"
	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/store"
)

func TestIngestImageEnforcesTimelineAndHashIdempotency(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "ingest.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	trial, err := st.CreateTrial("ING-001", "采集测试", "小麦", model.TrialRunning)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := st.CreateSeed(trial.ID, "S1", model.SeedPending)
	if err != nil {
		t.Fatal(err)
	}
	svc := ingest.New(st)
	first := time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)
	if _, created, err := svc.IngestImage(ingest.ImageInput{SeedID: seed.ID, Hash: "frame-1", CapturedAt: first}); err != nil || !created {
		t.Fatalf("first image should be created: created=%v err=%v", created, err)
	}
	if _, created, err := svc.IngestImage(ingest.ImageInput{SeedID: seed.ID, Hash: "frame-1", CapturedAt: first.Add(time.Hour)}); err != nil || created {
		t.Fatalf("duplicate image should be idempotent: created=%v err=%v", created, err)
	}
	if _, _, err := svc.IngestImage(ingest.ImageInput{SeedID: seed.ID, Hash: "frame-0", CapturedAt: first.Add(-time.Minute)}); err != model.ErrTimeReversed {
		t.Fatalf("out-of-order image should be rejected, got %v", err)
	}
}
