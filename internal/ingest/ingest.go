// Package ingest 采集模块：接收按时间采集的种子图像摘要，做幂等去重与时间倒序检测。
package ingest

import (
	"fmt"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/store"
)

// Service 采集服务。
type Service struct {
	store *store.Store
}

// New 构造采集服务。
func New(s *store.Store) *Service { return &Service{store: s} }

// ImageInput 图像录入输入。
type ImageInput struct {
	SeedID     int64
	Hash       string
	CapturedAt time.Time
	Width      int
	Height     int
	Note       string
}

// IngestImage 录入图像；幂等（同 seed+hash 返回原证据且不新增记录，即使重试时间戳早于已保存帧）；
// 时间倒序（真正不同摘要且早于上次采集）报错。
// 返回 (image, created, error)。
func (svc *Service) IngestImage(in ImageInput) (model.SeedImage, bool, error) {
	if in.Hash == "" {
		return model.SeedImage{}, false, model.ErrBadInput
	}
	// 幂等优先：同 seed+hash 视为同一帧的重试，直接返回原证据，不因时间戳早于已保存帧而拒绝。
	if existing, err := svc.store.GetImageByHash(in.SeedID, in.Hash); err == nil {
		return existing, false, nil
	} else if err != model.ErrNotFound {
		return model.SeedImage{}, false, err
	}
	// 真正不同的新摘要：查该种子上一帧采集时间，禁止时间倒序。
	imgs, err := svc.store.ListImages(in.SeedID)
	if err != nil {
		return model.SeedImage{}, false, err
	}
	for _, prev := range imgs {
		if prev.CapturedAt.After(in.CapturedAt) {
			return model.SeedImage{}, false, model.ErrTimeReversed
		}
	}
	img, created, err := svc.store.UpsertImage(in.SeedID, in.Hash, in.CapturedAt, in.Width, in.Height, in.Note)
	if err != nil {
		return model.SeedImage{}, false, fmt.Errorf("ingest image: %w", err)
	}
	return img, created, nil
}

// BulkIngest 批量录入（并发安全：不同种子可并行，同种子串行由调用方保证）。
func (svc *Service) BulkIngest(items []ImageInput) ([]model.SeedImage, []error) {
	out := make([]model.SeedImage, 0, len(items))
	errs := make([]error, 0)
	for _, it := range items {
		img, _, err := svc.IngestImage(it)
		if err != nil {
			errs = append(errs, fmt.Errorf("seed %d: %w", it.SeedID, err))
			continue
		}
		out = append(out, img)
	}
	return out, errs
}
