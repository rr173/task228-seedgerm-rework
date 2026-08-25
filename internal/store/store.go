// Package store 提供 SQLite 持久化：建表迁移与所有实体的 CRUD。
// 使用纯 Go 驱动 modernc.org/sqlite，CGO 无关，离线可构建。
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"task228-seedgerm/internal/model"
)

// Store 封装数据库连接与事务。
type Store struct {
	db *sql.DB
}

// Open 打开（必要时创建）SQLite 数据库并完成建表迁移。
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // WAL + 单写者，避免并发写锁
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭数据库。
func (s *Store) Close() error {
	return s.db.Close()
}

// DB 暴露底层 *sql.DB（供事务测试）。
func (s *Store) DB() *sql.DB { return s.db }

// migrate 执行建表（幂等）。
func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS trials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			species TEXT NOT NULL,
			state TEXT NOT NULL,
			sealed_at INTEGER,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS seeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL REFERENCES trials(id),
			seed_no TEXT NOT NULL,
			state TEXT NOT NULL,
			contam INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(trial_id, seed_no)
		)`,
		`CREATE TABLE IF NOT EXISTS seed_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			seed_id INTEGER NOT NULL REFERENCES seeds(id),
			hash TEXT NOT NULL,
			captured_at INTEGER NOT NULL,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			UNIQUE(seed_id, hash)
		)`,
		`CREATE TABLE IF NOT EXISTS env_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL REFERENCES trials(id),
			sampled_at INTEGER NOT NULL,
			temp_c REAL NOT NULL,
			humidity REAL NOT NULL,
			instrument TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			UNIQUE(trial_id, sampled_at, instrument)
		)`,
		`CREATE TABLE IF NOT EXISTS stage_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			seed_id INTEGER NOT NULL REFERENCES seeds(id),
			stage TEXT NOT NULL,
			state TEXT NOT NULL,
			occurred_at INTEGER NOT NULL,
			source TEXT NOT NULL,
			confidence REAL NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS observations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			seed_id INTEGER NOT NULL REFERENCES seeds(id),
			author TEXT NOT NULL,
			note TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS trial_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL REFERENCES trials(id),
			version INTEGER NOT NULL,
			state TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			prev_version INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(trial_id, version)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_seeds_trial ON seeds(trial_id)`,
		`CREATE INDEX IF NOT EXISTS idx_images_seed ON seed_images(seed_id)`,
		`CREATE INDEX IF NOT EXISTS idx_env_trial ON env_samples(trial_id)`,
		`CREATE INDEX IF NOT EXISTS idx_stage_seed ON stage_events(seed_id)`,
		`CREATE INDEX IF NOT EXISTS idx_obs_seed ON observations(seed_id)`,
		`CREATE INDEX IF NOT EXISTS idx_result_trial ON trial_results(trial_id)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return fmt.Errorf("migrate exec: %w", err)
		}
	}
	return nil
}

// nowUnix 返回当前 Unix 毫秒。
func nowUnix() int64 { return time.Now().UTC().UnixMilli() }

// scanTrial 从行读取 Trial。
func scanTrial(scan func(...interface{}) error) (model.Trial, error) {
	var t model.Trial
	var sealed int64
	var createdAt, updatedAt int64
	if err := scan(&t.ID, &t.Code, &t.Name, &t.Species, &t.State, &sealed, &createdAt, &updatedAt); err != nil {
		return t, err
	}
	t.CreatedAt = time.UnixMilli(createdAt).UTC()
	t.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	if sealed > 0 {
		ts := time.UnixMilli(sealed).UTC()
		t.SealedAt = &ts
	}
	return t, nil
}

// scanSeed 从行读取 Seed。
func scanSeed(scan func(...interface{}) error) (model.Seed, error) {
	var s model.Seed
	var contam int
	var createdAt, updatedAt int64
	if err := scan(&s.ID, &s.TrialID, &s.SeedNo, &s.State, &contam, &createdAt, &updatedAt); err != nil {
		return s, err
	}
	s.Contam = contam != 0
	s.CreatedAt = time.UnixMilli(createdAt).UTC()
	s.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return s, nil
}

// scanImage 从行读取 SeedImage。
func scanImage(scan func(...interface{}) error) (model.SeedImage, error) {
	var img model.SeedImage
	var capturedAt, createdAt int64
	if err := scan(&img.ID, &img.SeedID, &img.Hash, &capturedAt, &img.Width, &img.Height, &img.Note, &createdAt); err != nil {
		return img, err
	}
	img.CapturedAt = time.UnixMilli(capturedAt).UTC()
	img.CreatedAt = time.UnixMilli(createdAt).UTC()
	return img, nil
}

// scanEnv 从行读取 EnvSample。
func scanEnv(scan func(...interface{}) error) (model.EnvSample, error) {
	var e model.EnvSample
	var sampledAt, createdAt int64
	if err := scan(&e.ID, &e.TrialID, &sampledAt, &e.TempC, &e.Humidity, &e.Instrument, &createdAt); err != nil {
		return e, err
	}
	e.SampledAt = time.UnixMilli(sampledAt).UTC()
	e.CreatedAt = time.UnixMilli(createdAt).UTC()
	return e, nil
}

// scanStage 从行读取 StageEvent。
func scanStage(scan func(...interface{}) error) (model.StageEvent, error) {
	var st model.StageEvent
	var occurredAt, createdAt, updatedAt int64
	if err := scan(&st.ID, &st.SeedID, &st.Stage, &st.State, &occurredAt, &st.Source, &st.Confidence, &createdAt, &updatedAt); err != nil {
		return st, err
	}
	st.OccurredAt = time.UnixMilli(occurredAt).UTC()
	st.CreatedAt = time.UnixMilli(createdAt).UTC()
	st.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return st, nil
}

// scanObservation 从行读取 Observation。
func scanObservation(scan func(...interface{}) error) (model.Observation, error) {
	var o model.Observation
	var createdAt int64
	if err := scan(&o.ID, &o.SeedID, &o.Author, &o.Note, &createdAt); err != nil {
		return o, err
	}
	o.CreatedAt = time.UnixMilli(createdAt).UTC()
	return o, nil
}

// scanResult 从行读取 TrialResult。
func scanResult(scan func(...interface{}) error) (model.TrialResult, error) {
	var r model.TrialResult
	var createdAt, updatedAt int64
	if err := scan(&r.ID, &r.TrialID, &r.Version, &r.State, &r.Summary, &r.PrevVersion, &createdAt, &updatedAt); err != nil {
		return r, err
	}
	r.CreatedAt = time.UnixMilli(createdAt).UTC()
	r.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return r, nil
}
