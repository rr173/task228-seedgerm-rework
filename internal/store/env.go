package store

import (
	"database/sql"
	"fmt"
	"time"

	"task228-seedgerm/internal/model"
)

// UpsertEnv 插入或更新某试验某时刻某仪器的环境采样（幂等：同一 trial+sampled_at+instrument 唯一）。
func (s *Store) UpsertEnv(trialID int64, sampledAt time.Time, tempC, humidity float64, instrument string) (model.EnvSample, error) {
	row := s.db.QueryRow(
		`SELECT id,trial_id,sampled_at,temp_c,humidity,instrument,created_at FROM env_samples
		 WHERE trial_id=? AND sampled_at=? AND instrument=?`,
		trialID, sampledAt.UnixMilli(), instrument)
	if err := row.Err(); err != nil {
		return model.EnvSample{}, err
	}
	existing, err := scanEnv(row.Scan)
	if err == nil {
		// 已存在：更新数值
		if _, err := s.db.Exec(
			`UPDATE env_samples SET temp_c=?,humidity=? WHERE id=?`, tempC, humidity, existing.ID); err != nil {
			return model.EnvSample{}, fmt.Errorf("update env: %w", err)
		}
		return s.GetEnv(existing.ID)
	}
	if err != sql.ErrNoRows {
		return model.EnvSample{}, err
	}
	now := nowUnix()
	res, err := s.db.Exec(
		`INSERT INTO env_samples(trial_id,sampled_at,temp_c,humidity,instrument,created_at) VALUES(?,?,?,?,?,?)`,
		trialID, sampledAt.UnixMilli(), tempC, humidity, instrument, now)
	if err != nil {
		if isUniqueErr(err) {
			r2 := s.db.QueryRow(
				`SELECT id,trial_id,sampled_at,temp_c,humidity,instrument,created_at FROM env_samples
				 WHERE trial_id=? AND sampled_at=? AND instrument=?`,
				trialID, sampledAt.UnixMilli(), instrument)
			env, e2 := scanEnv(r2.Scan)
			if e2 == nil {
				return env, nil
			}
			return model.EnvSample{}, model.ErrConflict
		}
		return model.EnvSample{}, fmt.Errorf("insert env: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetEnv(id)
}

// GetEnv 按 ID 获取环境采样。
func (s *Store) GetEnv(id int64) (model.EnvSample, error) {
	row := s.db.QueryRow(
		`SELECT id,trial_id,sampled_at,temp_c,humidity,instrument,created_at FROM env_samples WHERE id=?`, id)
	env, err := scanEnv(row.Scan)
	if err == sql.ErrNoRows {
		return model.EnvSample{}, model.ErrNotFound
	}
	return env, err
}

// ListEnv 列出试验全部环境采样（按时间升序）。
func (s *Store) ListEnv(trialID int64) ([]model.EnvSample, error) {
	rows, err := s.db.Query(
		`SELECT id,trial_id,sampled_at,temp_c,humidity,instrument,created_at FROM env_samples
		 WHERE trial_id=? GROUP BY instrument ORDER BY sampled_at ASC`, trialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.EnvSample
	for rows.Next() {
		env, err := scanEnv(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, env)
	}
	return out, rows.Err()
}

// EnvAround 返回 trial 在 t 前后窗口内的环境采样（关联温湿度干预）。
func (s *Store) EnvAround(trialID int64, t time.Time, before, after time.Duration) ([]model.EnvSample, error) {
	lo := t.Add(-before).UnixMilli()
	hi := t.Add(after).UnixMilli()
	rows, err := s.db.Query(
		`SELECT id,trial_id,sampled_at,temp_c,humidity,instrument,created_at FROM env_samples
		 WHERE trial_id=? AND sampled_at>=? AND sampled_at<=? ORDER BY sampled_at ASC`, trialID, lo, hi)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.EnvSample
	for rows.Next() {
		env, err := scanEnv(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, env)
	}
	return out, rows.Err()
}
