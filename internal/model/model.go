// Package model 定义种子萌发证据台的核心实体、状态枚举与领域错误。
package model

import (
	"errors"
	"time"
)

// 领域错误。
var (
	ErrNotFound        = errors.New("resource not found")
	ErrConflict        = errors.New("resource conflict (duplicate key)")
	ErrInvalidState    = errors.New("invalid state transition")
	ErrSealed          = errors.New("trial is sealed, mutation rejected")
	ErrBadInput        = errors.New("invalid input")
	ErrTimeReversed    = errors.New("timestamp must not go backwards")
	ErrImageMissing    = errors.New("seed image evidence missing")
	ErrContamination   = errors.New("contamination marker present, image evidence immutable")
	ErrUnknownInstrument = errors.New("unknown instrument")
)

// TrialState 萌发试验状态。
type TrialState string

const (
	TrialPlanned    TrialState = "planned"
	TrialRunning    TrialState = "running"
	TrialReviewing  TrialState = "reviewing"
	TrialCompleted  TrialState = "completed"
	TrialSealed     TrialState = "sealed"
)

// ValidTrialState 校验状态合法。
func ValidTrialState(s string) bool {
	switch TrialState(s) {
	case TrialPlanned, TrialRunning, TrialReviewing, TrialCompleted, TrialSealed:
		return true
	}
	return false
}

// TrialStateTransitions 定义允许的状态流转。
var TrialStateTransitions = map[TrialState][]TrialState{
	TrialPlanned:   {TrialRunning},
	TrialRunning:   {TrialReviewing},
	TrialReviewing: {TrialCompleted},
	TrialCompleted: {TrialSealed},
	TrialSealed:    {},
}

// CanTransitionTrial 判断从 from 到 to 是否合法。
func CanTransitionTrial(from, to TrialState) bool {
	for _, n := range TrialStateTransitions[from] {
		if n == to {
			return true
		}
	}
	return false
}

// SeedState 单粒种子状态。
type SeedState string

const (
	SeedPending    SeedState = "pending"
	SeedObserving  SeedState = "observing"
	SeedGerminated SeedState = "germinated"
	SeedStalled    SeedState = "stalled"
	SeedContam     SeedState = "contaminated"
)

// ValidSeedState 校验种子状态合法。
func ValidSeedState(s string) bool {
	switch SeedState(s) {
	case SeedPending, SeedObserving, SeedGerminated, SeedStalled, SeedContam:
		return true
	}
	return false
}

// StageEventState 阶段事件状态。
type StageEventState string

const (
	StageCandidate StageEventState = "candidate"
	StageConfirmed StageEventState = "confirmed"
	StageConflict  StageEventState = "conflict"
	StageRevoked   StageEventState = "revoked"
)

// ValidStageEventState 校验阶段事件状态合法。
func ValidStageEventState(s string) bool {
	switch StageEventState(s) {
	case StageCandidate, StageConfirmed, StageConflict, StageRevoked:
		return true
	}
	return false
}

// StageEventTransitions 阶段事件状态流转。
var StageEventTransitions = map[StageEventState][]StageEventState{
	StageCandidate: {StageConfirmed, StageConflict, StageRevoked},
	StageConfirmed: {StageRevoked},
	StageConflict:  {StageConfirmed, StageRevoked},
	StageRevoked:   {},
}

// CanTransitionStage 判断阶段事件状态流转合法性。
func CanTransitionStage(from, to StageEventState) bool {
	for _, n := range StageEventTransitions[from] {
		if n == to {
			return true
		}
	}
	return false
}

// ResultState 试验结果版本状态。
type ResultState string

const (
	ResultDraft     ResultState = "draft"
	ResultPending   ResultState = "pending"
	ResultPublished ResultState = "published"
	ResultSuperseded ResultState = "superseded"
)

// ValidResultState 校验结果状态合法。
func ValidResultState(s string) bool {
	switch ResultState(s) {
	case ResultDraft, ResultPending, ResultPublished, ResultSuperseded:
		return true
	}
	return false
}

// ResultStateTransitions 结果版本流转。
var ResultStateTransitions = map[ResultState][]ResultState{
	ResultDraft:     {ResultPending},
	ResultPending:   {ResultPublished, ResultSuperseded},
	ResultPublished: {ResultSuperseded},
	ResultSuperseded: {},
}

// CanTransitionResult 判断结果状态流转合法性。
func CanTransitionResult(from, to ResultState) bool {
	for _, n := range ResultStateTransitions[from] {
		if n == to {
			return true
		}
	}
	return false
}

// GermStage 萌发生理阶段标签。
type GermStage string

const (
	StageImbibition   GermStage = "imbibition"   // 吸胀
	StageRadicle      GermStage = "radicle"      // 胚根出现
	StageCotyledon    GermStage = "cotyledon"    // 子叶突破
	StageColeoptile   GermStage = "coleoptile"   // 叶鞘展开
	StageStagnant     GermStage = "stagnant"     // 停滞
	StagePolluted     GermStage = "polluted"     // 污染
)

// TimeNow 返回当前 UTC 时间（集中封装便于测试替换语义一致）。
func TimeNow() time.Time {
	return time.Now().UTC()
}
