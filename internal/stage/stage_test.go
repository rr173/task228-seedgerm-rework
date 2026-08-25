package stage_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/stage"
	"task228-seedgerm/internal/store"
)

func TestDetectorConfirmsContaminationAndProtectsEvidence(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stage.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	trial, err := st.CreateTrial("STAGE-001", "阶段测试", "玉米", model.TrialRunning)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := st.CreateSeed(trial.ID, "C1", model.SeedPending)
	if err != nil {
		t.Fatal(err)
	}
	detector := stage.New(st)
	ev, err := detector.Detect(stage.DetectInput{SeedID: seed.ID, CapturedAt: time.Now().UTC(), ContamScore: 0.9}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Stage != model.StagePolluted || ev.State != model.StageCandidate {
		t.Fatalf("unexpected contamination event: %+v", ev)
	}
	if _, err := detector.Confirm(ev.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := st.GetSeed(seed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.State != model.SeedContam || !updated.Contam {
		t.Fatalf("contamination state was not linked: %+v", updated)
	}
}
