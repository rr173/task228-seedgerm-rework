package enviro_test

import (
	"path/filepath"
	"testing"
	"time"

	"task228-seedgerm/internal/enviro"
	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/store"
)

func TestRecordAndInterveneAroundSummarizeCooling(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "env.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	trial, err := st.CreateTrial("ENV-001", "环境测试", "水稻", model.TrialRunning)
	if err != nil {
		t.Fatal(err)
	}
	svc := enviro.New(st)
	at := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	if _, err := svc.Record(enviro.SampleInput{TrialID: trial.ID, SampledAt: at.Add(-time.Hour), TempC: 25, Humidity: 60, Instrument: "thermo-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Record(enviro.SampleInput{TrialID: trial.ID, SampledAt: at, TempC: 21, Humidity: 63, Instrument: "thermo-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Record(enviro.SampleInput{TrialID: trial.ID, SampledAt: at, TempC: 21, Humidity: 63, Instrument: ""}); err != model.ErrUnknownInstrument {
		t.Fatalf("empty instrument should fail, got %v", err)
	}
	iv, err := svc.InterveneAround(trial.ID, at, 2*time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !iv.Cooling || iv.TempDrop != 4 {
		t.Fatalf("unexpected intervention summary: %+v", iv)
	}
}
