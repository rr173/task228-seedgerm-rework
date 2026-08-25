package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug07ConflictResolutionIsAtomicOnInvalidPreferredEvent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R7", "atomic", "millet")
	svc.TransitionTrial(tr.ID, model.TrialRunning)
	seed, _ := svc.CreateSeed(tr.ID, "S1")
	at := time.Now().UTC()
	conflict, _ := st.CreateStage(seed.ID, model.StageRadicle, at, "auto", .8)
	preferred, _ := st.CreateStage(seed.ID, model.StageColeoptile, at.Add(time.Minute), "auto", .8)
	if _, err := svc.ConfirmStage(preferred.ID); err != nil { t.Fatal(err) }
	if _, err := svc.ResolveStageConflict(conflict.ID, preferred.ID); err == nil { t.Fatal("invalid preferred stage unexpectedly resolved") }
	afterConflict, _ := st.GetStage(conflict.ID)
	if afterConflict.State != model.StageCandidate { t.Fatalf("partial resolution changed conflict: %+v", afterConflict) }
}
