package store

import (
	"database/sql"
	"fmt"

	"task228-seedgerm/internal/model"
)

// NextResultVersion 计算试验下一个结果版本号（自增）。
//
// 注意：该读操作自身不持锁。若调用方需要在并发下保证版本号连续且唯一，
// 应使用 CreateResultNext，它在一个事务内原子地完成版本分配与插入，
// 避免读-改-写之间的竞争窗口。
func (s *Store) NextResultVersion(trialID int64) (int, error) {
	row := s.db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM trial_results WHERE trial_id=?`, trialID)
	var max int
	if err := row.Scan(&max); err != nil {
		return 0, fmt.Errorf("max version: %w", err)
	}
	return max + 1, nil
}

// CreateResult 创建试验结果版本（draft 态）。prevVersion 指向被替代版本（0 表示首个）。
// 版本号由调用方指定，trial_id+version 唯一约束冲突时返回 ErrConflict。
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

// CreateResultNext 在单个事务内原子地分配下一个连续版本号并插入结果版本。
// 由于 SetMaxOpenConns(1) 使写事务串行化，版本号分配（MAX+1）与插入之间不存在
// 竞争窗口，从而保证并发起草同一试验时版本号连续且唯一、所有请求都成功。
// prev_version 自动指向上一版本（首个版本为 0），形成完整版本链。
func (s *Store) CreateResultNext(trialID int64, summary string) (model.TrialResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return model.TrialResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var max int
	if err := tx.QueryRow(
		`SELECT COALESCE(MAX(version),0) FROM trial_results WHERE trial_id=?`, trialID,
	).Scan(&max); err != nil {
		return model.TrialResult{}, fmt.Errorf("max version: %w", err)
	}
	version := max + 1
	prev := 0
	if version > 1 {
		prev = version - 1
	}
	now := nowUnix()
	res, err := tx.Exec(
		`INSERT INTO trial_results(trial_id,version,state,summary,prev_version,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?)`,
		trialID, version, string(model.ResultDraft), summary, prev, now, now)
	if err != nil {
		if isUniqueErr(err) {
			return model.TrialResult{}, model.ErrConflict
		}
		return model.TrialResult{}, fmt.Errorf("insert result: %w", err)
	}
	id, _ := res.LastInsertId()
	if err := tx.Commit(); err != nil {
		return model.TrialResult{}, fmt.Errorf("commit tx: %w", err)
	}
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

// SupersedeResult 将某版本标记为 superseded。
func (s *Store) SupersedeResult(id int64) error {
	_, err := s.db.Exec(`UPDATE trial_results SET state=? WHERE id=?`, string(model.ResultSuperseded), id)
	return err
}
