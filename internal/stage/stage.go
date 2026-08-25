// Package stage 阶段模块：检测胚根出现、叶鞘展开等萌发生理阶段事件，识别停滞与污染。
package stage

import (
	"fmt"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/store"
)

// Detector 阶段检测服务。
type Detector struct {
	store *store.Store
}

// New 构造阶段检测服务。
func New(s *store.Store) *Detector { return &Detector{store: s} }

// DetectInput 检测输入。
type DetectInput struct {
	SeedID     int64
	CapturedAt time.Time
	// 图像特征（由上游视觉模块产出，这里用简化指标）。
	RadicleVisible bool    // 胚根可见
	ColeoptileLen  float64 // 叶鞘长度(mm)
	ContamScore    float64 // 污染指数 0~1
}

// Detect 依据图像特征与既有阶段序列创建候选阶段事件。
// 规则：
//   - ContamScore >= 0.7 → polluted（污染）
//   - RadicleVisible 且尚无 radicle → radicle（胚根出现）
//   - ColeoptileLen >= 2.0 且尚无 coleoptile → coleoptile（叶鞘展开）
//   - 距上一 confirmed 阶段超过 stallHours 且无新进展 → stagnant（停滞，仅作候选标记）
func (d *Detector) Detect(in DetectInput, stallHours float64) (model.StageEvent, error) {
	prev, err := d.store.ListStages(in.SeedID)
	if err != nil {
		return model.StageEvent{}, err
	}
	for _, p := range prev {
		if in.CapturedAt.Before(p.OccurredAt) {
			return model.StageEvent{}, model.ErrTimeReversed
		}
	}
	hasStage := func(s model.GermStage) bool {
		for _, p := range prev {
			if p.Stage == s && p.State != model.StageRevoked {
				return true
			}
		}
		return false
	}

	var stage model.GermStage
	var conf float64
	switch {
	case in.ContamScore >= 0.7:
		stage = model.StagePolluted
		conf = in.ContamScore
	case in.RadicleVisible && !hasStage(model.StageRadicle):
		stage = model.StageRadicle
		conf = 0.9
	case in.ColeoptileLen >= 2.0 && !hasStage(model.StageColeoptile):
		stage = model.StageColeoptile
		conf = 0.85
	case d.stalled(prev, in.CapturedAt, stallHours):
		stage = model.StageStagnant
		conf = 0.6
	default:
		// 无新事件
		return model.StageEvent{}, model.ErrNotFound
	}

	ev, err := d.store.CreateStage(in.SeedID, stage, in.CapturedAt, "auto", conf)
	if err != nil {
		return model.StageEvent{}, fmt.Errorf("detect stage: %w", err)
	}
	return ev, nil
}

// stalled 判断在 lastConfirmed 之后 window 内无任何新确认阶段。
func (d *Detector) stalled(prev []model.StageEvent, now time.Time, stallHours float64) bool {
	if stallHours <= 0 {
		return false
	}
	var last time.Time
	for _, p := range prev {
		if p.State == model.StageConfirmed && p.OccurredAt.After(last) {
			last = p.OccurredAt
		}
	}
	if last.IsZero() {
		return false
	}
	return now.Sub(last).Hours() >= stallHours
}

// Confirm 确认候选阶段事件（状态 candidate→confirmed），并联动种子状态。
// 污染不可逆：种子一旦确认污染（contam 标记为真），后续任何阶段确认都不得
// 清除污染标记或把种子状态从 contaminated 回退到萌发/停滞/观测状态。
func (d *Detector) Confirm(stageID int64) (model.StageEvent, error) {
	ev, err := d.store.UpdateStageState(stageID, model.StageConfirmed)
	if err != nil {
		return ev, err
	}
	// 污染证据不可逆：已污染种子不再因新阶段确认而联动状态或清除污染标记，
	// 仅保留阶段事件自身的确认，种子保持 contaminated。
	seed, err := d.store.GetSeed(ev.SeedID)
	if err != nil {
		return ev, err
	}
	if seed.Contam {
		return ev, nil
	}
	// 联动种子状态（仅未污染种子）
	var seedState model.SeedState
	switch ev.Stage {
	case model.StageRadicle, model.StageColeoptile:
		seedState = model.SeedGerminated
	case model.StageStagnant:
		seedState = model.SeedStalled
	case model.StagePolluted:
		seedState = model.SeedContam
	default:
		seedState = model.SeedObserving
	}
	if _, err := d.store.UpdateSeedState(ev.SeedID, seedState); err != nil {
		return ev, err
	}
	if seedState == model.SeedContam {
		if err := d.store.MarkSeedContam(ev.SeedID, true); err != nil {
			return ev, err
		}
	}
	return ev, nil
}

// ResolveConflict 处理冲突：将冲突事件置 revoked 并把用户指定事件 confirm。
func (d *Detector) ResolveConflict(conflictID, preferID int64) error {
	if _, err := d.store.UpdateStageState(conflictID, model.StageRevoked); err != nil {
		return fmt.Errorf("revoke conflict: %w", err)
	}
	if preferID != conflictID {
		if _, err := d.store.UpdateStageState(preferID, model.StageConfirmed); err != nil {
			return fmt.Errorf("confirm preferred: %w", err)
		}
	}
	return nil
}
