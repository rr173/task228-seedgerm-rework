package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug05StageTimelineRejectsOlderObservation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R5", "timeline", "soy")
	svc.TransitionTrial(tr.ID, model.TrialRunning)
	seed, _ := svc.CreateSeed(tr.ID, "S1")
	base := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	if _, err := svc.DetectStage(seed.ID, base.Add(time.Hour), true, 0, 0, 0); err != nil { t.Fatal(err) }
	if _, err := svc.DetectStage(seed.ID, base, false, 2.5, 0, 0); err != model.ErrTimeReversed { t.Fatalf("older stage accepted: %v", err) }
}
