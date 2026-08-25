package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"task228-seedgerm/internal/model"
)

// seedDetail GET 种子详情。
func (s *Server) seedDetail(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
		return
	}
	seed, err := s.store.GetSeed(id)
	if err != nil {
		s.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, seed)
}

// seedImages GET/POST 种子图像。
func (s *Server) seedImages(w http.ResponseWriter, r *http.Request, seedID int64) {
	switch r.Method {
	case http.MethodGet:
		imgs, err := s.store.ListImages(seedID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, imgs)
	case http.MethodPost:
		var body struct {
			Hash       string    `json:"hash"`
			CapturedAt time.Time `json:"captured_at"`
			Width      int       `json:"width"`
			Height     int       `json:"height"`
			Note       string    `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Hash == "" {
			writeError(w, http.StatusBadRequest, model.ErrBadInput)
			return
		}
		img, created, err := s.svc.IngestImage(seedID, body.Hash, body.CapturedAt, body.Width, body.Height, body.Note)
		if err != nil {
			s.mapErr(w, err)
			return
		}
		code := http.StatusCreated
		if !created {
			code = http.StatusOK
		}
		writeJSON(w, code, img)
	default:
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
	}
}

// seedObservations GET/POST 人工观察。
func (s *Server) seedObservations(w http.ResponseWriter, r *http.Request, seedID int64) {
	switch r.Method {
	case http.MethodGet:
		obs, err := s.store.ListObservations(seedID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, obs)
	case http.MethodPost:
		var body struct {
			Author string `json:"author"`
			Note   string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Author == "" {
			writeError(w, http.StatusBadRequest, model.ErrBadInput)
			return
		}
		o, err := s.svc.AddObservation(seedID, body.Author, body.Note)
		if err != nil {
			s.mapErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, o)
	default:
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
	}
}

// seedDetect POST 阶段检测。
func (s *Server) seedDetect(w http.ResponseWriter, r *http.Request, seedID int64) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, model.ErrBadInput)
		return
	}
	var body struct {
		CapturedAt    time.Time `json:"captured_at"`
		Radicle       bool      `json:"radicle_visible"`
		ColeoptileLen float64   `json:"coleoptile_len"`
		ContamScore   float64   `json:"contam_score"`
		StallHours    float64   `json:"stall_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrBadInput)
		return
	}
	ev, err := s.svc.DetectStage(seedID, body.CapturedAt, body.Radicle, body.ColeoptileLen, body.ContamScore, body.StallHours)
	if err != nil {
		s.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}
