package service_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug02EnvironmentSamplesKeepSubsecondNaturalKey(t *testing.T) {
	_ = model.TrialRunning
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R2", "precision", "rice")
	base := time.Date(2026, 8, 25, 10, 0, 0, 123000000, time.UTC)
	if _, err := svc.RecordEnv(tr.ID, base, 25, 60, "thermo-1"); err != nil { t.Fatal(err) }
	if _, err := svc.RecordEnv(tr.ID, base.Add(700*time.Millisecond), 21, 62, "thermo-1"); err != nil { t.Fatal(err) }
	items, err := svc.Enviro.List(tr.ID)
	if err != nil { t.Fatal(err) }
	if len(items) != 2 { t.Fatalf("subsecond observations were merged: %+v", items) }
}
