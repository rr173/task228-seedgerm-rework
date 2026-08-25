// Package enviro 环境模块：关联温湿度干预与阶段事件，识别降温/波动对萌发的影响。
package enviro

import (
	"fmt"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/store"
)

// Service 环境服务。
type Service struct {
	store *store.Store
}

// New 构造环境服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// SampleInput 环境采样输入。
type SampleInput struct {
	TrialID    int64
	SampledAt  time.Time
	TempC      float64
	Humidity   float64
	Instrument string
}

// Record 记录环境采样（幂等，同 trial+sampled_at+instrument 覆盖）。
func (svc *Service) Record(in SampleInput) (model.EnvSample, error) {
	if in.Instrument == "" {
		return model.EnvSample{}, model.ErrUnknownInstrument
	}
	env, err := svc.store.UpsertEnv(in.TrialID, in.SampledAt, in.TempC, in.Humidity, in.Instrument)
	if err != nil {
		return model.EnvSample{}, fmt.Errorf("record env: %w", err)
	}
	return env, nil
}

// InterveneAround 返回种子所属试验在某阶段发生时刻前后的环境干预（温湿度变化）。
func (svc *Service) InterveneAround(trialID int64, at time.Time, before, after time.Duration) (Intervention, error) {
	samples, err := svc.store.EnvAround(trialID, at, before, after)
	if err != nil {
		return Intervention{}, err
	}
	iv := Intervention{At: at, Samples: samples}
	if len(samples) == 0 {
		return iv, nil
	}
	minT, maxT := samples[0].TempC, samples[0].TempC
	for _, s := range samples {
		if s.TempC < minT {
			minT = s.TempC
		}
		if s.TempC > maxT {
			maxT = s.TempC
		}
	}
	iv.TempDrop = maxT - minT
	iv.Cooling = minT < samples[0].TempC-0.5
	return iv, nil
}

// Intervention 环境干预摘要。
type Intervention struct {
	At       time.Time
	Samples  []model.EnvSample
	TempDrop float64 // 窗口内最大温差
	Cooling  bool    // 是否出现降温
}

// List 列出试验全部环境采样。
func (svc *Service) List(trialID int64) ([]model.EnvSample, error) {
	return svc.store.ListEnv(trialID)
}
