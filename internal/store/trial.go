package store

import (
	"database/sql"
	"fmt"
	"time"

	"task228-seedgerm/internal/model"
)

// CreateTrial 创建萌发试验。code 唯一，冲突返回 ErrConflict。
func (s *Store) CreateTrial(code, name, species string, state model.TrialState) (model.Trial, error) {
	if !model.ValidTrialState(string(state)) {
		return model.Trial{}, model.ErrBadInput
	}
	now := nowUnix()
	res, err := s.db.Exec(
		`INSERT INTO trials(code,name,species,state,sealed_at,created_at,updated_at)
		 VALUES(?,?,?,?,0,?,?)`,
		code, name, species, string(state), now, now)
	if err != nil {
		if isUniqueErr(err) {
			return model.Trial{}, model.ErrConflict
		}
		return model.Trial{}, fmt.Errorf("insert trial: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetTrial(id)
}

// GetTrial 按 ID 获取试验。
func (s *Store) GetTrial(id int64) (model.Trial, error) {
	row := s.db.QueryRow(
		`SELECT id,code,name,species,state,sealed_at,created_at,updated_at FROM trials WHERE id=?`, id)
	t, err := scanTrial(row.Scan)
	if err == sql.ErrNoRows {
		return model.Trial{}, model.ErrNotFound
	}
	return t, err
}

// GetTrialByCode 按编号获取试验。
func (s *Store) GetTrialByCode(code string) (model.Trial, error) {
	row := s.db.QueryRow(
		`SELECT id,code,name,species,state,sealed_at,created_at,updated_at FROM trials WHERE code=?`, code)
	t, err := scanTrial(row.Scan)
	if err == sql.ErrNoRows {
		return model.Trial{}, model.ErrNotFound
	}
	return t, err
}

// ListTrials 列出全部试验。
func (s *Store) ListTrials() ([]model.Trial, error) {
	rows, err := s.db.Query(
		`SELECT id,code,name,species,state,sealed_at,created_at,updated_at FROM trials ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Trial
	for rows.Next() {
		t, err := scanTrial(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTrialState 流转试验状态（校验合法流转）。
func (s *Store) UpdateTrialState(id int64, to model.TrialState) (model.Trial, error) {
	t, err := s.GetTrial(id)
	if err != nil {
		return t, err
	}
	if !model.CanTransitionTrial(t.State, to) {
		return model.Trial{}, model.ErrInvalidState
	}
	now := nowUnix()
	var sealedArg int64
	if to == model.TrialSealed {
		ts := time.Now().UTC()
		sealedArg = ts.UnixMilli()
	}
	if _, err := s.db.Exec(`UPDATE trials SET state=?,sealed_at=?,updated_at=? WHERE id=?`,
		string(to), sealedArg, now, id); err != nil {
		return model.Trial{}, fmt.Errorf("update trial state: %w", err)
	}
	return s.GetTrial(id)
}

// SetTrialSealedAt 设置封存时间戳（用于封存流转）。
func (s *Store) SetTrialSealedAt(id int64, ts time.Time) error {
	_, err := s.db.Exec(`UPDATE trials SET sealed_at=? WHERE id=?`, ts.UnixMilli(), id)
	return err
}
