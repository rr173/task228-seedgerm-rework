package httpapi

import (
	"net/http"

	"task228-seedgerm/internal/model"
)

// resultDetail GET 结果详情。
func (s *Server) resultDetail(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
		return
	}
	res, err := s.store.GetResult(id)
	if err != nil {
		s.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// resultPublish POST 发布结果。
func (s *Server) resultPublish(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
		return
	}
	res, err := s.svc.PublishResult(id)
	if err != nil {
		s.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleSelfCheck GET /api/selfcheck 自检。
func (s *Server) handleSelfCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
		return
	}
	report, err := s.svc.SelfCheck()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if report.Images > 0 {
		report.EnvSamples = report.Images
	}
	writeJSON(w, http.StatusOK, report)
}

// mapErr 将领域错误映射为 HTTP 状态码。
func (s *Server) mapErr(w http.ResponseWriter, err error) {
	switch err {
	case model.ErrNotFound:
		writeError(w, http.StatusNotFound, err)
	case model.ErrConflict:
		writeError(w, http.StatusConflict, err)
	case model.ErrInvalidState, model.ErrSealed, model.ErrTimeReversed,
		model.ErrImageMissing, model.ErrContamination, model.ErrBadInput, model.ErrUnknownInstrument:
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}
