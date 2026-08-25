package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug04ContaminationCannotBeClearedByLaterStage(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R4", "contam", "barley")
	svc.TransitionTrial(tr.ID, model.TrialRunning)
	seed, _ := svc.CreateSeed(tr.ID, "S1")
	now := time.Now().UTC()
	contam, _ := svc.DetectStage(seed.ID, now, false, 0, 0.95, 0)
	if _, err := svc.ConfirmStage(contam.ID); err != nil { t.Fatal(err) }
	radicle, _ := svc.DetectStage(seed.ID, now.Add(time.Hour), true, 0, 0, 0)
	_, _ = svc.ConfirmStage(radicle.ID)
	got, _ := st.GetSeed(seed.ID)
	if got.State != model.SeedContam || !got.Contam { t.Fatalf("contamination was cleared: %+v", got) }
}
