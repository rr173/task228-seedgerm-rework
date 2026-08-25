package model_test

import (
	"testing"

	"task228-seedgerm/internal/model"
)

func TestTrialStateMachineOnlyAllowsForwardReview(t *testing.T) {
	if !model.CanTransitionTrial(model.TrialPlanned, model.TrialRunning) {
		t.Fatal("planned trial should be startable")
	}
	if model.CanTransitionTrial(model.TrialRunning, model.TrialCompleted) {
		t.Fatal("running trial must pass through reviewing")
	}
	if model.CanTransitionTrial(model.TrialSealed, model.TrialRunning) {
		t.Fatal("sealed trial must be immutable")
	}
}

func TestStageAndResultStatesRejectUnknownValues(t *testing.T) {
	if model.ValidStageEventState("lost") {
		t.Fatal("unknown stage state accepted")
	}
	if model.ValidResultState("released") {
		t.Fatal("unknown result state accepted")
	}
	if !model.CanTransitionResult(model.ResultPublished, model.ResultSuperseded) {
		t.Fatal("published result should be replaceable")
	}
}
