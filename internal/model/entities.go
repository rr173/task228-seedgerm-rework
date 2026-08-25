package model

import "time"

// Trial 萌发试验。
type Trial struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`        // 试验编号（唯一）
	Name        string     `json:"name"`
	Species     string     `json:"species"`     // 物种
	State       TrialState `json:"state"`
	SealedAt    *time.Time `json:"sealed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Seed 单粒种子。
type Seed struct {
	ID          int64     `json:"id"`
	TrialID     int64     `json:"trial_id"`
	SeedNo      string    `json:"seed_no"` // 种子编号（试验内唯一）
	State       SeedState `json:"state"`
	Contam      bool      `json:"contam"` // 污染标记（污染后图像证据不可删）
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SeedImage 单粒种子按时间采集的图像摘要。
type SeedImage struct {
	ID          int64     `json:"id"`
	SeedID      int64     `json:"seed_id"`
	Hash        string    `json:"hash"` // 图像内容哈希（幂等键）
	CapturedAt  time.Time `json:"captured_at"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

// EnvSample 环境采样（温湿度曲线）。
type EnvSample struct {
	ID          int64     `json:"id"`
	TrialID     int64     `json:"trial_id"`
	SampledAt   time.Time `json:"sampled_at"`
	TempC       float64   `json:"temp_c"`
	Humidity    float64   `json:"humidity"`
	Instrument  string    `json:"instrument"`
	CreatedAt   time.Time `json:"created_at"`
}

// StageEvent 萌发生理阶段事件。
type StageEvent struct {
	ID          int64           `json:"id"`
	SeedID      int64           `json:"seed_id"`
	Stage       GermStage       `json:"stage"`
	State       StageEventState `json:"state"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Source      string          `json:"source"` // auto | manual
	Confidence  float64         `json:"confidence"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Observation 人工观察（复核意见）。
type Observation struct {
	ID          int64     `json:"id"`
	SeedID      int64     `json:"seed_id"`
	Author      string    `json:"author"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

// TrialResult 试验结果版本。
type TrialResult struct {
	ID          int64       `json:"id"`
	TrialID     int64       `json:"trial_id"`
	Version     int         `json:"version"` // 自增版本号
	State       ResultState `json:"state"`
	Summary     string      `json:"summary"`
	PrevVersion int         `json:"prev_version"` // 0 表示首个版本
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
