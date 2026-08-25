package store

import (
	"database/sql"
	"fmt"

	"task228-seedgerm/internal/model"
)

// NextResultVersion 计算试验下一个结果版本号（自增）。
func (s *Store) NextResultVersion(trialID int64) (int, error) {
	row := s.db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM trial_results WHERE trial_id=?`, trialID)
	var max int
	if err := row.Scan(&max); err != nil {
		return 0, fmt.Errorf("max version: %w", err)
	}
	return max + 1, nil
}

// CreateResult 创建试验结果版本（draft 态）。prevVersion 指向被替代版本（0 表示首个）。
func (s *Store) CreateResult(trialID int64, version int, summary string, prevVersion int) (model.TrialResult, error) {
	now := nowUnix()
	res, err := s.db.Exec(
		`INSERT INTO trial_results(trial_id,version,state,summary,prev_version,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?)`,
		trialID, version, string(model.ResultDraft), summary, prevVersion, now, now)
	if err != nil {
		if isUniqueErr(err) {
			return model.TrialResult{}, model.ErrConflict
		}
		return model.TrialResult{}, fmt.Errorf("insert result: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetResult(id)
}

// GetResult 按 ID 获取结果。
func (s *Store) GetResult(id int64) (model.TrialResult, error) {
	row := s.db.QueryRow(
		`SELECT id,trial_id,version,state,summary,prev_version,created_at,updated_at FROM trial_results WHERE id=?`, id)
	r, err := scanResult(row.Scan)
	if err == sql.ErrNoRows {
		return model.TrialResult{}, model.ErrNotFound
	}
	return r, err
}

// ListResults 列出试验全部结果版本（按版本升序）。
func (s *Store) ListResults(trialID int64) ([]model.TrialResult, error) {
	rows, err := s.db.Query(
		`SELECT id,trial_id,version,state,summary,prev_version,created_at,updated_at FROM trial_results
		 WHERE trial_id=? ORDER BY version ASC`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.TrialResult
	for rows.Next() {
		r, err := scanResult(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateResultState 流转结果状态。
func (s *Store) UpdateResultState(id int64, to model.ResultState) (model.TrialResult, error) {
	r, err := s.GetResult(id)
	if err != nil {
		return r, err
	}
	if !model.CanTransitionResult(r.State, to) {
		return r, model.ErrInvalidState
	}
	now := nowUnix()
	if _, err := s.db.Exec(`UPDATE trial_results SET state=?,updated_at=? WHERE id=?`, string(to), now, id); err != nil {
		return model.TrialResult{}, fmt.Errorf("update result state: %w", err)
	}
	return s.GetResult(id)
}

// PublishResult 原子发布结果：将目标版本置 published，并把同试验下其它已发布版本
// 置 superseded，保证同一试验同一时刻至多一个已发布版本。
// 支持从 draft 或 pending 发布；按状态机 draft→pending→published 两步流转。
func (s *Store) PublishResult(id int64) (model.TrialResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return model.TrialResult{}, err
	}
	defer tx.Rollback()

	var trialID int64
	var state string
	if err := tx.QueryRow(
		`SELECT trial_id,state FROM trial_results WHERE id=?`, id).
		Scan(&trialID, &state); err != nil {
		if err == sql.ErrNoRows {
			return model.TrialResult{}, model.ErrNotFound
		}
		return model.TrialResult{}, err
	}
	from := model.ResultState(state)
	// 仅 draft/pending 可发布
	if from != model.ResultDraft && from != model.ResultPending {
		return model.TrialResult{}, model.ErrInvalidState
	}
	now := nowUnix()
	// draft 先流转到 pending（状态机：draft→pending）
	if from == model.ResultDraft {
		if _, err := tx.Exec(
			`UPDATE trial_results SET state=?,updated_at=? WHERE id=?`,
			string(model.ResultPending), now, id); err != nil {
			return model.TrialResult{}, fmt.Errorf("to pending: %w", err)
		}
	}
	// 把同试验其它已发布版本置 superseded（排除当前版本）
	if _, err := tx.Exec(
		`UPDATE trial_results SET state=?,updated_at=?
		 WHERE trial_id=? AND state=? AND id<>?`,
		string(model.ResultSuperseded), now, trialID, string(model.ResultPublished), id); err != nil {
		return model.TrialResult{}, fmt.Errorf("supersede published results: %w", err)
	}
	// 再把目标版本置 published（状态机：pending→published）
	if _, err := tx.Exec(
		`UPDATE trial_results SET state=?,updated_at=? WHERE id=?`,
		string(model.ResultPublished), now, id); err != nil {
		return model.TrialResult{}, fmt.Errorf("publish result: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return model.TrialResult{}, err
	}
	return s.GetResult(id)
}
