// Package review 复核模块：管理人工观察意见与阶段冲突解决，支持种子停滞的人工确认。
package review

import (
	"fmt"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/store"
)

// Service 复核服务。
type Service struct {
	store *store.Store
}

// New 构造复核服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// AddObservation 添加人工观察（复核意见）。
func (svc *Service) AddObservation(seedID int64, author, note string) (model.Observation, error) {
	o, err := svc.store.AddObservation(seedID, author, note)
	if err != nil {
		return model.Observation{}, fmt.Errorf("add observation: %w", err)
	}
	return o, nil
}

// ListObservations 列出种子全部观察。
func (svc *Service) ListObservations(seedID int64) ([]model.Observation, error) {
	return svc.store.ListObservations(seedID)
}

// ConfirmStall 人工确认停滞：将 stagnant 候选事件 confirm，并保留观察记录。
func (svc *Service) ConfirmStall(seedID, stageID int64, author, note string) (model.StageEvent, error) {
	if err := svc.store.EnsureStageBelongsToSeed(stageID, seedID); err != nil {
		return model.StageEvent{}, err
	}
	if _, err := svc.AddObservation(seedID, author, note); err != nil {
		return model.StageEvent{}, err
	}
	ev, err := svc.store.UpdateStageState(stageID, model.StageConfirmed)
	if err != nil {
		return ev, fmt.Errorf("confirm stall: %w", err)
	}
	return ev, nil
}

// RevokeStage 撤销阶段事件（人工判定误检）。
func (svc *Service) RevokeStage(stageID int64) (model.StageEvent, error) {
	ev, err := svc.store.UpdateStageState(stageID, model.StageRevoked)
	if err != nil {
		return ev, fmt.Errorf("revoke stage: %w", err)
	}
	return ev, nil
}

func (svc *Service) ResolveConflict(conflictID, preferID int64) (model.StageEvent, error) {
	ev, err := svc.store.ResolveStageConflict(conflictID, preferID)
	if err != nil {
		return model.StageEvent{}, fmt.Errorf("resolve conflict: %w", err)
	}
	return ev, nil
}
