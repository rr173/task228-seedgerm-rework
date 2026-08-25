package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func newSvc(t *testing.T) (*service.Service, func()) {
	t.Helper()
	db := filepath.Join(t.TempDir(), "seedgerm.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	svc := service.New(st)
	return svc, func() { _ = st.Close() }
}

// TestEndToEnd 端到端：试验→种子→图像→环境→阶段→确认→观察→结果→发布。
func TestEndToEnd(t *testing.T) {
	svc, cleanup := newSvc(t)
	defer cleanup()

	trial, err := svc.CreateTrial("T-001", "小麦萌发", "小麦")
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if _, err := svc.TransitionTrial(trial.ID, model.TrialRunning); err != nil {
		t.Fatalf("transition: %v", err)
	}
	seed, err := svc.CreateSeed(trial.ID, "S1")
	if err != nil {
		t.Fatalf("create seed: %v", err)
	}
	now := time.Now().UTC()
	if _, _, err := svc.IngestImage(seed.ID, "h1", now.Add(-2*time.Hour), 640, 480, ""); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	// 幂等
	if _, created, err := svc.IngestImage(seed.ID, "h1", now.Add(-1*time.Hour), 640, 480, "dup"); err != nil || created {
		t.Fatalf("idempotent failed: created=%v err=%v", created, err)
	}
	if _, err := svc.RecordEnv(trial.ID, now.Add(-2*time.Hour), 22, 60, "th1"); err != nil {
		t.Fatalf("env: %v", err)
	}
	ev, err := svc.DetectStage(seed.ID, now, true, 0, 0.1, 0)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if ev.Stage != model.StageRadicle {
		t.Fatalf("stage want radicle got %s", ev.Stage)
	}
	if _, err := svc.ConfirmStage(ev.ID); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if _, err := svc.AddObservation(seed.ID, "r", "胚根明显"); err != nil {
		t.Fatalf("observation: %v", err)
	}
	// 状态机：running 不能跳到 completed
	if _, err := svc.TransitionTrial(trial.ID, model.TrialCompleted); err != model.ErrInvalidState {
		t.Fatalf("invalid transition should fail, got %v", err)
	}
	if _, err := svc.TransitionTrial(trial.ID, model.TrialReviewing); err != nil {
		t.Fatalf("to reviewing: %v", err)
	}
	res, err := svc.DraftResult(trial.ID, "萌发率100%")
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if _, err := svc.PublishResult(res.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	report, err := svc.SelfCheck()
	if err != nil {
		t.Fatalf("selfcheck: %v", err)
	}
	if report.Seeds != 1 || report.Images != 1 || report.EnvSamples != 1 {
		t.Fatalf("selfcheck counts: %+v", report)
	}
}

// TestContamImageImmutable 污染种子禁止删除图像证据。
func TestContamImageImmutable(t *testing.T) {
	svc, cleanup := newSvc(t)
	defer cleanup()
	trial, _ := svc.CreateTrial("T-002", "玉米", "玉米")
	svc.TransitionTrial(trial.ID, model.TrialRunning)
	seed, _ := svc.CreateSeed(trial.ID, "C1")
	now := time.Now().UTC()
	img, _, err := svc.IngestImage(seed.ID, "hc", now, 640, 480, "")
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	// 标记污染（通过检测污染阶段）
	ev, err := svc.DetectStage(seed.ID, now.Add(time.Hour), false, 0, 0.9, 0)
	if err != nil {
		t.Fatalf("detect contam: %v", err)
	}
	if _, err := svc.ConfirmStage(ev.ID); err != nil {
		t.Fatalf("confirm contam: %v", err)
	}
	// 删除图像应被拒
	if err := svc.DeleteImage(img.ID); err != model.ErrContamination {
		t.Fatalf("delete image should be rejected, got %v", err)
	}
}

// TestSealedTrialImmutable 封存试验禁止修改。
func TestSealedTrialImmutable(t *testing.T) {
	svc, cleanup := newSvc(t)
	defer cleanup()
	trial, _ := svc.CreateTrial("T-003", "水稻", "水稻")
	svc.TransitionTrial(trial.ID, model.TrialRunning)
	svc.TransitionTrial(trial.ID, model.TrialReviewing)
	svc.TransitionTrial(trial.ID, model.TrialCompleted)
	svc.TransitionTrial(trial.ID, model.TrialSealed)
	if _, err := svc.CreateSeed(trial.ID, "X1"); err != model.ErrSealed {
		t.Fatalf("sealed trial should reject seed, got %v", err)
	}
}
