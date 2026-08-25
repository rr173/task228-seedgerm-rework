package store

import (
	"database/sql"
	"fmt"

	"task228-seedgerm/internal/model"
)

// CreateSeed 在试验下创建种子。seed_no 试验内唯一，冲突返回 ErrConflict。
func (s *Store) CreateSeed(trialID int64, seedNo string, state model.SeedState) (model.Seed, error) {
	if !model.ValidSeedState(string(state)) {
		return model.Seed{}, model.ErrBadInput
	}
	now := nowUnix()
	res, err := s.db.Exec(
		`INSERT INTO seeds(trial_id,seed_no,state,contam,created_at,updated_at) VALUES(?,?,?,0,?,?)`,
		trialID, seedNo, string(state), now, now)
	if err != nil {
		if isUniqueErr(err) {
			return model.Seed{}, model.ErrConflict
		}
		return model.Seed{}, fmt.Errorf("insert seed: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetSeed(id)
}

// GetSeed 按 ID 获取种子。
func (s *Store) GetSeed(id int64) (model.Seed, error) {
	row := s.db.QueryRow(
		`SELECT id,trial_id,seed_no,state,contam,created_at,updated_at FROM seeds WHERE id=?`, id)
	seed, err := scanSeed(row.Scan)
	if err == sql.ErrNoRows {
		return model.Seed{}, model.ErrNotFound
	}
	return seed, err
}

// ListSeeds 列出试验下全部种子。
func (s *Store) ListSeeds(trialID int64) ([]model.Seed, error) {
	rows, err := s.db.Query(
		`SELECT id,trial_id,seed_no,state,contam,created_at,updated_at FROM seeds WHERE trial_id=? ORDER BY id`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Seed
	for rows.Next() {
		seed, err := scanSeed(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, seed)
	}
	return out, rows.Err()
}

// UpdateSeedState 流转种子状态。污染态不可回退到观测中（不可删证据约束由 service 把关）。
func (s *Store) UpdateSeedState(id int64, to model.SeedState) (model.Seed, error) {
	seed, err := s.GetSeed(id)
	if err != nil {
		return seed, err
	}
	if !model.ValidSeedState(string(to)) {
		return model.Seed{}, model.ErrBadInput
	}
	now := nowUnix()
	if _, err := s.db.Exec(`UPDATE seeds SET state=?,updated_at=? WHERE id=?`, string(to), now, id); err != nil {
		return model.Seed{}, fmt.Errorf("update seed state: %w", err)
	}
	return s.GetSeed(id)
}

// MarkSeedContam 设置污染标记（不可清除图像证据，由调用方约束）。
func (s *Store) MarkSeedContam(id int64, contam bool) error {
	now := nowUnix()
	_, err := s.db.Exec(`UPDATE seeds SET contam=?,updated_at=? WHERE id=?`, boolInt(contam), now, id)
	return err
}

// boolInt 布尔转整数。
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
