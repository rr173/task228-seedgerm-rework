package store

import (
	"database/sql"
	"fmt"
	"time"

	"task228-seedgerm/internal/model"
)

// CreateStage 创建阶段事件（候选态）。
func (s *Store) CreateStage(seedID int64, stage model.GermStage, occurredAt time.Time, source string, confidence float64) (model.StageEvent, error) {
	var latest int64
	err := s.db.QueryRow(`SELECT COALESCE(MAX(occurred_at),0) FROM stage_events WHERE seed_id=?`, seedID).Scan(&latest)
	if err != nil {
		return model.StageEvent{}, fmt.Errorf("latest stage time: %w", err)
	}
	if latest > 0 && occurredAt.UnixMilli() < latest {
		return model.StageEvent{}, model.ErrTimeReversed
	}
	now := nowUnix()
	res, err := s.db.Exec(
		`INSERT INTO stage_events(seed_id,stage,state,occurred_at,source,confidence,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?,?)`,
		seedID, string(stage), string(model.StageCandidate), occurredAt.UnixMilli(), source, confidence, now, now)
	if err != nil {
		return model.StageEvent{}, fmt.Errorf("insert stage: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetStage(id)
}

// GetStage 按 ID 获取阶段事件。
func (s *Store) GetStage(id int64) (model.StageEvent, error) {
	row := s.db.QueryRow(
		`SELECT id,seed_id,stage,state,occurred_at,source,confidence,created_at,updated_at FROM stage_events WHERE id=?`, id)
	st, err := scanStage(row.Scan)
	if err == sql.ErrNoRows {
		return model.StageEvent{}, model.ErrNotFound
	}
	return st, err
}

// ListStages 列出种子全部阶段事件（按发生时间升序）。
func (s *Store) ListStages(seedID int64) ([]model.StageEvent, error) {
	rows, err := s.db.Query(
		`SELECT id,seed_id,stage,state,occurred_at,source,confidence,created_at,updated_at FROM stage_events
		 WHERE seed_id=? ORDER BY occurred_at ASC`, seedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.StageEvent
	for rows.Next() {
		st, err := scanStage(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// UpdateStageState 流转阶段事件状态。
func (s *Store) UpdateStageState(id int64, to model.StageEventState) (model.StageEvent, error) {
	st, err := s.GetStage(id)
	if err != nil {
		return st, err
	}
	if !model.CanTransitionStage(st.State, to) {
		return st, model.ErrInvalidState
	}
	now := nowUnix()
	if _, err := s.db.Exec(`UPDATE stage_events SET state=?,updated_at=? WHERE id=?`, string(to), now, id); err != nil {
		return model.StageEvent{}, fmt.Errorf("update stage state: %w", err)
	}
	return s.GetStage(id)
}

func (s *Store) EnsureStageBelongsToSeed(stageID, seedID int64) error {
	var actual int64
	if err := s.db.QueryRow(`SELECT seed_id FROM stage_events WHERE id=?`, stageID).Scan(&actual); err != nil {
		if err == sql.ErrNoRows {
			return model.ErrNotFound
		}
		return err
	}
	if actual != seedID {
		return model.ErrConflict
	}
	return nil
}

func (s *Store) ResolveStageConflict(conflictID, preferID int64) (model.StageEvent, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return model.StageEvent{}, err
	}
	defer tx.Rollback()
	var conflictSeed, preferSeed int64
	var conflictState, preferState string
	if err := tx.QueryRow(`SELECT seed_id,state FROM stage_events WHERE id=?`, conflictID).Scan(&conflictSeed, &conflictState); err != nil {
		if err == sql.ErrNoRows {
			return model.StageEvent{}, model.ErrNotFound
		}
		return model.StageEvent{}, err
	}
	if err := tx.QueryRow(`SELECT seed_id,state FROM stage_events WHERE id=?`, preferID).Scan(&preferSeed, &preferState); err != nil {
		if err == sql.ErrNoRows {
			return model.StageEvent{}, model.ErrNotFound
		}
		return model.StageEvent{}, err
	}
	if conflictSeed != preferSeed {
		return model.StageEvent{}, model.ErrConflict
	}
	if !model.CanTransitionStage(model.StageEventState(conflictState), model.StageRevoked) || !model.CanTransitionStage(model.StageEventState(preferState), model.StageConfirmed) {
		return model.StageEvent{}, model.ErrInvalidState
	}
	now := nowUnix()
	if _, err := tx.Exec(`UPDATE stage_events SET state=?,updated_at=? WHERE id=?`, string(model.StageRevoked), now, conflictID); err != nil {
		return model.StageEvent{}, err
	}
	if _, err := tx.Exec(`UPDATE stage_events SET state=?,updated_at=? WHERE id=?`, string(model.StageConfirmed), now, preferID); err != nil {
		return model.StageEvent{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.StageEvent{}, err
	}
	return s.GetStage(preferID)
}

// LatestConfirmedStage 返回种子最近一个 confirmed 阶段（用于阶段边修订）。
func (s *Store) LatestConfirmedStage(seedID int64) (model.StageEvent, error) {
	row := s.db.QueryRow(
		`SELECT id,seed_id,stage,state,occurred_at,source,confidence,created_at,updated_at FROM stage_events
		 WHERE seed_id=? AND state=? ORDER BY occurred_at DESC LIMIT 1`,
		seedID, string(model.StageConfirmed))
	st, err := scanStage(row.Scan)
	if err == sql.ErrNoRows {
		return model.StageEvent{}, model.ErrNotFound
	}
	return st, err
}
