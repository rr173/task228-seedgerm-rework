package store

import (
	"database/sql"
	"fmt"

	"task228-seedgerm/internal/model"
)

// AddObservation 为种子添加人工观察（复核意见）。
func (s *Store) AddObservation(seedID int64, author, note string) (model.Observation, error) {
	if author == "" {
		return model.Observation{}, model.ErrBadInput
	}
	now := nowUnix()
	res, err := s.db.Exec(
		`INSERT INTO observations(seed_id,author,note,created_at) VALUES(?,?,?,?)`,
		seedID, author, note, now)
	if err != nil {
		return model.Observation{}, fmt.Errorf("insert observation: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetObservation(id)
}

// GetObservation 按 ID 获取观察。
func (s *Store) GetObservation(id int64) (model.Observation, error) {
	row := s.db.QueryRow(
		`SELECT id,seed_id,author,note,created_at FROM observations WHERE id=?`, id)
	o, err := scanObservation(row.Scan)
	if err == sql.ErrNoRows {
		return model.Observation{}, model.ErrNotFound
	}
	return o, err
}

// ListObservations 列出种子全部观察（按时间升序）。
func (s *Store) ListObservations(seedID int64) ([]model.Observation, error) {
	rows, err := s.db.Query(
		`SELECT id,seed_id,author,note,created_at FROM observations WHERE seed_id=? ORDER BY created_at ASC`, seedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Observation
	for rows.Next() {
		o, err := scanObservation(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
