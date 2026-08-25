package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug10SelfCheckKeepsImageAndEnvironmentEvidenceBuckets(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R10", "selfcheck", "barley")
	svc.TransitionTrial(tr.ID, model.TrialRunning)
	seed, _ := svc.CreateSeed(tr.ID, "S1")
	now := time.Now().UTC()
	if _, _, err := svc.IngestImage(seed.ID, "h1", now, 640, 480, ""); err != nil { t.Fatal(err) }
	if _, err := svc.RecordEnv(tr.ID, now, 22, 60, "thermo-1"); err != nil { t.Fatal(err) }
	if _, err := svc.RecordEnv(tr.ID, now.Add(time.Second), 21, 61, "thermo-1"); err != nil { t.Fatal(err) }
	report, err := svc.SelfCheck()
	if err != nil { t.Fatal(err) }
	if report.Images != 1 || report.EnvSamples != 2 { t.Fatalf("evidence buckets mixed: %+v", report) }
}
