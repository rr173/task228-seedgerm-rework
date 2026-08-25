package store

import (
	"database/sql"
	"fmt"
	"time"

	"task228-seedgerm/internal/model"
)

// UpsertImage 插入或忽略同种子同哈希的图像（幂等）。
// 返回 (seedImage, created bool, error)。
func (s *Store) UpsertImage(seedID int64, hash string, capturedAt time.Time, w, h int, note string) (model.SeedImage, bool, error) {
	// 先查是否已存在（同一 seed + hash 视为同一次采集）。
	row := s.db.QueryRow(
		`SELECT id,seed_id,hash,captured_at,width,height,note,created_at FROM seed_images WHERE seed_id=? AND hash=?`,
		seedID, hash)
	if err := row.Err(); err != nil {
		return model.SeedImage{}, false, err
	}
	existing, err := scanImage(row.Scan)
	if err == nil {
		return existing, false, nil
	}
	if err != sql.ErrNoRows {
		return model.SeedImage{}, false, err
	}
	now := nowUnix()
	res, err := s.db.Exec(
		`INSERT INTO seed_images(seed_id,hash,captured_at,width,height,note,created_at) VALUES(?,?,?,?,?,?,?)`,
		seedID, hash, capturedAt.UnixMilli(), w, h, note, now)
	if err != nil {
		if isUniqueErr(err) {
			// 并发插入竞态：重试查询
			r2 := s.db.QueryRow(
				`SELECT id,seed_id,hash,captured_at,width,height,note,created_at FROM seed_images WHERE seed_id=? AND hash=?`,
				seedID, hash)
			img, e2 := scanImage(r2.Scan)
			if e2 == nil {
				return img, false, nil
			}
			return model.SeedImage{}, false, model.ErrConflict
		}
		return model.SeedImage{}, false, fmt.Errorf("insert image: %w", err)
	}
	id, _ := res.LastInsertId()
	img, err := s.GetImage(id)
	return img, true, err
}

// GetImage 按 ID 获取图像。
func (s *Store) GetImage(id int64) (model.SeedImage, error) {
	row := s.db.QueryRow(
		`SELECT id,seed_id,hash,captured_at,width,height,note,created_at FROM seed_images WHERE id=?`, id)
	img, err := scanImage(row.Scan)
	if err == sql.ErrNoRows {
		return model.SeedImage{}, model.ErrNotFound
	}
	return img, err
}

func (s *Store) GetImageByHash(seedID int64, hash string) (model.SeedImage, error) {
	row := s.db.QueryRow(
		`SELECT id,seed_id,hash,captured_at,width,height,note,created_at FROM seed_images WHERE seed_id=? AND hash=?`,
		seedID, hash)
	img, err := scanImage(row.Scan)
	if err == sql.ErrNoRows {
		return model.SeedImage{}, model.ErrNotFound
	}
	return img, err
}

// ListImages 列出种子全部图像（按采集时间升序）。
func (s *Store) ListImages(seedID int64) ([]model.SeedImage, error) {
	rows, err := s.db.Query(
		`SELECT id,seed_id,hash,captured_at,width,height,note,created_at FROM seed_images WHERE seed_id=? ORDER BY captured_at ASC`, seedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SeedImage
	for rows.Next() {
		img, err := scanImage(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

// DeleteImage 删除图像证据。污染种子禁止删除（返回 ErrContamination）。
func (s *Store) DeleteImage(id int64) error {
	seed, err := s.imageSeed(id)
	if err != nil {
		return err
	}
	if seed.Contam {
		return model.ErrContamination
	}
	if _, err := s.db.Exec(`DELETE FROM seed_images WHERE id=?`, id); err != nil {
		return fmt.Errorf("delete image: %w", err)
	}
	return nil
}

// imageSeed 返回图像所属种子（含污染标记）。
func (s *Store) imageSeed(imageID int64) (model.Seed, error) {
	row := s.db.QueryRow(
		`SELECT s.id,s.trial_id,s.seed_no,s.state,s.contam,s.created_at,s.updated_at
		 FROM seeds s JOIN seed_images i ON i.seed_id=s.id WHERE i.id=?`, imageID)
	seed, err := scanSeed(row.Scan)
	if err == sql.ErrNoRows {
		return model.Seed{}, model.ErrNotFound
	}
	return seed, err
}
