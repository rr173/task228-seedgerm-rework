// Package service 编排层：把采集/阶段/环境/复核/结果各业务包串成业务闭环，
// 并在入口处把关并发与错误边界（封存不可改、种子编号冲突、污染不可删证据等）。
package service

import (
	"fmt"
	"sync"
	"time"

	"task228-seedgerm/internal/enviro"
	"task228-seedgerm/internal/ingest"
	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/result"
	"task228-seedgerm/internal/review"
	"task228-seedgerm/internal/stage"
	"task228-seedgerm/internal/store"
)

// Service 业务编排服务。
type Service struct {
	store  *store.Store
	Ingest *ingest.Service
	Stage  *stage.Detector
	Enviro *enviro.Service
	Review *review.Service
	Result *result.Service

	mu sync.Mutex // 保护同试验的串行判定（如结果合并）
}

// New 构造编排服务。
func New(s *store.Store) *Service {
	return &Service{
		store:  s,
		Ingest: ingest.New(s),
		Stage:  stage.New(s),
		Enviro: enviro.New(s),
		Review: review.New(s),
		Result: result.New(s),
	}
}

// CreateTrial 创建试验。
func (svc *Service) CreateTrial(code, name, species string) (model.Trial, error) {
	return svc.store.CreateTrial(code, name, species, model.TrialPlanned)
}

// TransitionTrial 流转试验状态。
func (svc *Service) TransitionTrial(id int64, to model.TrialState) (model.Trial, error) {
	return svc.store.UpdateTrialState(id, to)
}

// CreateSeed 创建种子。
func (svc *Service) CreateSeed(trialID int64, seedNo string) (model.Seed, error) {
	// 封存试验禁止修改
	t, err := svc.store.GetTrial(trialID)
	if err != nil {
		return model.Seed{}, err
	}
	if t.State == model.TrialSealed {
		return model.Seed{}, model.ErrSealed
	}
	return svc.store.CreateSeed(trialID, seedNo, model.SeedPending)
}

// IngestImage 录入图像（委托采集模块）。
func (svc *Service) IngestImage(seedID int64, hash string, capturedAt time.Time, w, h int, note string) (model.SeedImage, bool, error) {
	// 封存试验禁止修改
	seed, err := svc.store.GetSeed(seedID)
	if err != nil {
		return model.SeedImage{}, false, err
	}
	t, err := svc.store.GetTrial(seed.TrialID)
	if err != nil {
		return model.SeedImage{}, false, err
	}
	if t.State == model.TrialSealed {
		return model.SeedImage{}, false, model.ErrSealed
	}
	capturedAt = capturedAt.Truncate(time.Second)
	return svc.Ingest.IngestImage(ingest.ImageInput{SeedID: seedID, Hash: hash, CapturedAt: capturedAt, Width: w, Height: h, Note: note})
}

// DetectStage 检测阶段（委托阶段模块）。
func (svc *Service) DetectStage(seedID int64, capturedAt time.Time, radicle bool, coleoptileLen, contamScore float64, stallHours float64) (model.StageEvent, error) {
	seed, err := svc.store.GetSeed(seedID)
	if err != nil {
		return model.StageEvent{}, err
	}
	t, err := svc.store.GetTrial(seed.TrialID)
	if err != nil {
		return model.StageEvent{}, err
	}
	if t.State == model.TrialSealed {
		return model.StageEvent{}, model.ErrSealed
	}
	return svc.Stage.Detect(stage.DetectInput{
		SeedID:         seedID,
		CapturedAt:     capturedAt,
		RadicleVisible: radicle,
		ColeoptileLen:  coleoptileLen,
		ContamScore:    contamScore,
	}, stallHours)
}

// ConfirmStage 确认阶段（联动种子状态）。
func (svc *Service) ConfirmStage(stageID int64) (model.StageEvent, error) {
	ev, err := svc.store.GetStage(stageID)
	if err != nil {
		return model.StageEvent{}, err
	}
	seed, err := svc.store.GetSeed(ev.SeedID)
	if err != nil {
		return model.StageEvent{}, err
	}
	trial, err := svc.store.GetTrial(seed.TrialID)
	if err != nil {
		return model.StageEvent{}, err
	}
	if trial.State == model.TrialSealed {
		return model.StageEvent{}, model.ErrSealed
	}
	return svc.Stage.Confirm(stageID)
}

func (svc *Service) EnsureStageBelongsToSeed(stageID, seedID int64) error {
	return svc.store.EnsureStageBelongsToSeed(stageID, seedID)
}

func (svc *Service) ResolveStageConflict(conflictID, preferID int64) (model.StageEvent, error) {
	return svc.Review.ResolveConflict(conflictID, preferID)
}

// RecordEnv 记录环境采样。
func (svc *Service) RecordEnv(trialID int64, sampledAt time.Time, tempC, humidity float64, instrument string) (model.EnvSample, error) {
	t, err := svc.store.GetTrial(trialID)
	if err != nil {
		return model.EnvSample{}, err
	}
	if t.State == model.TrialSealed {
		return model.EnvSample{}, model.ErrSealed
	}
	return svc.Enviro.Record(enviro.SampleInput{TrialID: trialID, SampledAt: sampledAt, TempC: tempC, Humidity: humidity, Instrument: instrument})
}

// AddObservation 添加人工观察。
func (svc *Service) AddObservation(seedID int64, author, note string) (model.Observation, error) {
	return svc.Review.AddObservation(seedID, author, note)
}

// ConfirmStall 人工确认停滞。
func (svc *Service) ConfirmStall(seedID, stageID int64, author, note string) (model.StageEvent, error) {
	return svc.Review.ConfirmStall(seedID, stageID, author, note)
}

// DraftResult 起草结果（串行合并保护）。
func (svc *Service) DraftResult(trialID int64, summary string) (model.TrialResult, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	t, err := svc.store.GetTrial(trialID)
	if err != nil {
		return model.TrialResult{}, err
	}
	if t.State == model.TrialSealed {
		return model.TrialResult{}, model.ErrSealed
	}
	return svc.Result.Draft(trialID, summary)
}

// PublishResult 发布结果。
func (svc *Service) PublishResult(resultID int64) (model.TrialResult, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	return svc.Result.Publish(resultID)
}

// Summarize 汇总试验证据。
func (svc *Service) Summarize(trialID int64) (result.Summary, error) {
	return svc.Result.Summarize(trialID)
}

// DeleteImage 删除图像证据（污染种子拒绝，由 store 把关）。
func (svc *Service) DeleteImage(imageID int64) error {
	return svc.store.DeleteImage(imageID)
}

// SelfCheck 自检：统计各表行数，验证不变量（封存试验未修改、污染种子图像未被删）。
func (svc *Service) SelfCheck() (SelfCheckReport, error) {
	report := SelfCheckReport{}
	trials, err := svc.store.ListTrials()
	if err != nil {
		return report, err
	}
	report.Trials = len(trials)
	for _, t := range trials {
		seeds, err := svc.store.ListSeeds(t.ID)
		if err != nil {
			return report, err
		}
		report.Seeds += len(seeds)
		for _, s := range seeds {
			imgs, err := svc.store.ListImages(s.ID)
			if err != nil {
				return report, err
			}
			report.Images += len(imgs)
			if s.Contam && len(imgs) == 0 {
				// 污染种子却无图像证据：违反不变量
				report.Violations = append(report.Violations,
					fmt.Sprintf("contaminated seed %d has no image evidence", s.ID))
			}
		}
		envs, err := svc.store.ListEnv(t.ID)
		if err != nil {
			return report, err
		}
		report.EnvSamples += len(envs)
	}
	return report, nil
}

// SelfCheckReport 自检报告。
type SelfCheckReport struct {
	Trials     int      `json:"trials"`
	Seeds      int      `json:"seeds"`
	Images     int      `json:"images"`
	EnvSamples int      `json:"env_samples"`
	Violations []string `json:"violations,omitempty"`
}
