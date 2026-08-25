// Package httpapi HTTP 层：以 /api 前缀暴露试验/种子/图像/环境/阶段/复核/结果/自检 API。
package httpapi

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

// webAssets 是随服务一起发布的最小实验工作台，避免运行时依赖外部静态文件目录。
//
//go:embed web/*
var webAssets embed.FS

// Server HTTP 服务。
type Server struct {
	svc    *service.Service
	store  *store.Store
	addr   string
	dbPath string
}

// New 构造 HTTP 服务。
func New(svc *service.Service, st *store.Store, addr, dbPath string) *Server {
	return &Server{svc: svc, store: st, addr: addr, dbPath: dbPath}
}

// Handler 返回路由 mux。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// 每个业务入口都显式注册，既让 HTTP 契约可发现，也让网页和外部采集端可以
	// 独立验证单个资源动作，而不是依赖一个“大而全”的路径分发器。
	mux.HandleFunc("POST /api/trials", s.handleTrials)
	mux.HandleFunc("GET /api/trials", s.handleTrials)
	mux.HandleFunc("GET /api/trials/{id}", withID("/api/trials/", s.trialDetail))
	mux.HandleFunc("POST /api/trials/{id}/transition", withID("/api/trials/", s.trialTransition))
	mux.HandleFunc("GET /api/trials/{id}/seeds", withID("/api/trials/", s.trialSeeds))
	mux.HandleFunc("POST /api/trials/{id}/seeds", withID("/api/trials/", s.trialSeeds))
	mux.HandleFunc("GET /api/trials/{id}/env", withID("/api/trials/", s.trialEnv))
	mux.HandleFunc("POST /api/trials/{id}/env", withID("/api/trials/", s.trialEnv))
	mux.HandleFunc("GET /api/trials/{id}/results", withID("/api/trials/", s.trialResults))
	mux.HandleFunc("POST /api/trials/{id}/results", withID("/api/trials/", s.trialResults))
	mux.HandleFunc("GET /api/trials/{id}/summarize", withID("/api/trials/", s.trialSummarize))
	mux.HandleFunc("GET /api/seeds/{id}", withID("/api/seeds/", s.seedDetail))
	mux.HandleFunc("GET /api/seeds/{id}/images", withID("/api/seeds/", s.seedImages))
	mux.HandleFunc("POST /api/seeds/{id}/images", withID("/api/seeds/", s.seedImages))
	mux.HandleFunc("GET /api/seeds/{id}/observations", withID("/api/seeds/", s.seedObservations))
	mux.HandleFunc("POST /api/seeds/{id}/observations", withID("/api/seeds/", s.seedObservations))
	mux.HandleFunc("POST /api/seeds/{id}/detect", withID("/api/seeds/", s.seedDetect))
	mux.HandleFunc("GET /api/stages/{id}", withID("/api/stages/", s.stageDetail))
	mux.HandleFunc("POST /api/stages/{id}/confirm", withID("/api/stages/", s.stageConfirm))
	mux.HandleFunc("POST /api/stages/{id}/resolve", withID("/api/stages/", s.stageResolve))
	mux.HandleFunc("GET /api/results/{id}", withID("/api/results/", s.resultDetail))
	mux.HandleFunc("POST /api/results/{id}/publish", withID("/api/results/", s.resultPublish))
	mux.HandleFunc("GET /api/selfcheck", s.handleSelfCheck)

	webRoot, err := fs.Sub(webAssets, "web")
	if err == nil {
		mux.Handle("GET /", http.FileServer(http.FS(webRoot)))
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

// withID 将 Go 1.22+ ServeMux 的路径变量适配为现有领域 handler 的整数 ID。
// 适配器只负责路径解析，业务 handler 仍分别负责自己的输入、状态和持久化规则。
func withID(prefix string, fn func(http.ResponseWriter, *http.Request, int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(r.URL.Path, prefix)
		if !ok {
			writeError(w, http.StatusBadRequest, model.ErrBadInput)
			return
		}
		fn(w, r, id)
	}
}

// Start 启动服务（长驻）。
func (s *Server) Start() error {
	srv := &http.Server{Addr: s.addr, Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	return srv.ListenAndServe()
}

// writeJSON 写 JSON 响应。
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 写错误响应。
func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

// parseID 从路径段取 ID。
func parseID(path, prefix string) (int64, bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return 0, false
	}
	// 取第一段
	seg := rest
	if idx := strings.Index(rest, "/"); idx >= 0 {
		seg = rest[:idx]
	}
	id, err := strconv.ParseInt(seg, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
