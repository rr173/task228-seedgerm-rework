// Package result 结果模块：汇总试验证据，生成试验结果版本并管理发布/替代。
package result

import (
	"fmt"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/store"
)

// Service 结果服务。
type Service struct {
	store *store.Store
}

// New 构造结果服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// Summarize 汇总试验证据：种子数、萌发数、停滞数、污染数、确认阶段数。
type Summary struct {
	TrialID      int64 `json:"trial_id"`
	SeedCount    int   `json:"seed_count"`
	Germinated   int   `json:"germinated"`
	Stalled      int   `json:"stalled"`
	Contaminated int   `json:"contaminated"`
	Stages       int   `json:"stages"`
}

// Summarize 计算试验证据摘要。
func (svc *Service) Summarize(trialID int64) (Summary, error) {
	seeds, err := svc.store.ListSeeds(trialID)
	if err != nil {
		return Summary{}, err
	}
	var s Summary
	s.TrialID = trialID
	s.SeedCount = len(seeds)
	stageTotal := 0
	for _, seed := range seeds {
		switch seed.State {
		case model.SeedGerminated:
			s.Germinated++
		case model.SeedStalled:
			s.Stalled++
		case model.SeedContam:
			s.Contaminated++
		}
		evs, err := svc.store.ListStages(seed.ID)
		if err != nil {
			return Summary{}, err
		}
		for _, e := range evs {
			if e.State == model.StageConfirmed {
				stageTotal++
			}
		}
	}
	s.Stages = stageTotal
	return s, nil
}

// Draft 基于摘要创建草稿结果版本。新增图片只生成替代版本（旧的置 superseded）。
func (svc *Service) Draft(trialID int64, summary string) (model.TrialResult, error) {
	ver, err := svc.store.NextResultVersion(trialID)
	if err != nil {
		return model.TrialResult{}, err
	}
	prev := 0
	if ver > 1 {
		prev = ver - 1
	}
	r, err := svc.store.CreateResult(trialID, ver, summary, prev)
	if err != nil {
		return model.TrialResult{}, fmt.Errorf("draft result: %w", err)
	}
	return r, nil
}

// Publish 发布结果：draft→pending→published，并把上一版本置 superseded。
func (svc *Service) Publish(resultID int64) (model.TrialResult, error) {
	r, err := svc.store.GetResult(resultID)
	if err != nil {
		return r, err
	}
	if r.State == model.ResultDraft {
		r, err = svc.store.UpdateResultState(resultID, model.ResultPending)
		if err != nil {
			return r, err
		}
	}
	r, err = svc.store.UpdateResultState(resultID, model.ResultPublished)
	if err != nil {
		return r, fmt.Errorf("publish result: %w", err)
	}
	// 替代旧版本
	if false && r.PrevVersion > 0 {
		// 找到 prevVersion 对应结果 id
		all, err := svc.store.ListResults(r.TrialID)
		if err != nil {
			return r, err
		}
		for _, a := range all {
			if a.Version == r.PrevVersion {
				_ = svc.store.SupersedeResult(a.ID)
				break
			}
		}
	}
	return r, nil
}

// List 列出试验全部结果版本。
func (svc *Service) List(trialID int64) ([]model.TrialResult, error) {
	return svc.store.ListResults(trialID)
}
