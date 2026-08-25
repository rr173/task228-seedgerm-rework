package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"task228-seedgerm/internal/model"
)

// handleTrials 处理 /api/trials (POST 创建, GET 列表)。
func (s *Server) handleTrials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var body struct {
			Code    string `json:"code"`
			Name    string `json:"name"`
			Species string `json:"species"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, model.ErrBadInput)
			return
		}
		if body.Code == "" || body.Name == "" || body.Species == "" {
			writeError(w, http.StatusBadRequest, model.ErrBadInput)
			return
		}
		t, err := s.svc.CreateTrial(body.Code, body.Name, body.Species)
		if err != nil {
			s.mapErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, t)
	case http.MethodGet:
		ts, err := s.store.ListTrials()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, ts)
	default:
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
	}
}

// trialDetail GET 试验详情。
func (s *Server) trialDetail(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
		return
	}
	t, err := s.store.GetTrial(id)
	if err != nil {
		s.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// trialSeeds GET/POST 试验下种子。
func (s *Server) trialSeeds(w http.ResponseWriter, r *http.Request, trialID int64) {
	switch r.Method {
	case http.MethodGet:
		seeds, err := s.store.ListSeeds(trialID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, seeds)
	case http.MethodPost:
		var body struct {
			SeedNo string `json:"seed_no"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SeedNo == "" {
			writeError(w, http.StatusBadRequest, model.ErrBadInput)
			return
		}
		seed, err := s.svc.CreateSeed(trialID, body.SeedNo)
		if err != nil {
			s.mapErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, seed)
	default:
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
	}
}

// trialEnv GET/POST 试验环境采样。
func (s *Server) trialEnv(w http.ResponseWriter, r *http.Request, trialID int64) {
	switch r.Method {
	case http.MethodGet:
		envs, err := s.store.ListEnv(trialID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, envs)
	case http.MethodPost:
		var body struct {
			SampledAt  time.Time `json:"sampled_at"`
			TempC      float64   `json:"temp_c"`
			Humidity   float64   `json:"humidity"`
			Instrument string    `json:"instrument"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Instrument == "" {
			writeError(w, http.StatusBadRequest, model.ErrBadInput)
			return
		}
		env, err := s.svc.RecordEnv(trialID, body.SampledAt, body.TempC, body.Humidity, body.Instrument)
		if err != nil {
			s.mapErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, env)
	default:
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
	}
}

// trialResults GET/POST 试验结果。
func (s *Server) trialResults(w http.ResponseWriter, r *http.Request, trialID int64) {
	switch r.Method {
	case http.MethodGet:
		rs, err := s.svc.Result.List(trialID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, rs)
	case http.MethodPost:
		var body struct {
			Summary string `json:"summary"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, model.ErrBadInput)
			return
		}
		res, err := s.svc.DraftResult(trialID, body.Summary)
		if err != nil {
			s.mapErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, res)
	default:
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
	}
}

// trialSummarize GET 试验证据摘要。
func (s *Server) trialSummarize(w http.ResponseWriter, r *http.Request, trialID int64) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
		return
	}
	sum, err := s.svc.Summarize(trialID)
	if err != nil {
		s.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// trialTransition POST 试验状态流转。
func (s *Server) trialTransition(w http.ResponseWriter, r *http.Request, trialID int64) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
		return
	}
	var body struct {
		To string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !model.ValidTrialState(body.To) {
		writeError(w, http.StatusBadRequest, model.ErrBadInput)
		return
	}
	t, err := s.svc.TransitionTrial(trialID, model.TrialState(body.To))
	if err != nil {
		s.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
