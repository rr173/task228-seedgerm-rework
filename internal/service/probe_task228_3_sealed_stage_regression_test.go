package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug03SealedTrialRejectsStageConfirmation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R3", "sealed", "corn")
	svc.TransitionTrial(tr.ID, model.TrialRunning)
	seed, _ := svc.CreateSeed(tr.ID, "S1")
	ev, err := svc.DetectStage(seed.ID, time.Now().UTC(), true, 0, 0, 0)
	if err != nil { t.Fatal(err) }
	svc.TransitionTrial(tr.ID, model.TrialReviewing)
	svc.TransitionTrial(tr.ID, model.TrialCompleted)
	svc.TransitionTrial(tr.ID, model.TrialSealed)
	if _, err := svc.ConfirmStage(ev.ID); err != model.ErrSealed { t.Fatalf("sealed stage mutation accepted: %v", err) }
	got, _ := st.GetStage(ev.ID)
	if got.State != model.StageCandidate { t.Fatalf("stage was mutated: %+v", got) }
}
