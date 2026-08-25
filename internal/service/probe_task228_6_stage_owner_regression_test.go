package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug06ConfirmStallRejectsStageFromAnotherSeed(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R6", "owner", "oat")
	svc.TransitionTrial(tr.ID, model.TrialRunning)
	a, _ := svc.CreateSeed(tr.ID, "A")
	b, _ := svc.CreateSeed(tr.ID, "B")
	ev, err := st.CreateStage(b.ID, model.StageStagnant, time.Now().UTC(), "manual", 0.5)
	if err != nil { t.Fatal(err) }
	if _, err := svc.ConfirmStall(a.ID, ev.ID, "reviewer", "wrong seed"); err != model.ErrConflict { t.Fatalf("cross-seed confirmation accepted: %v", err) }
	obs, _ := svc.Review.ListObservations(a.ID)
	if len(obs) != 0 { t.Fatalf("observation was recorded for wrong seed: %+v", obs) }
}
