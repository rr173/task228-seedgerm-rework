// Command seedgerm 是农业种子萌发时序证据台的服务入口。
// 支持 --addr :端口 --db 路径 启动长驻服务，以及 --smoke-test 自检闭环。
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"task228-seedgerm/internal/httpapi"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP 监听地址")
	dbPath := flag.String("db", "./seedgerm.db", "SQLite 数据库路径")
	smoke := flag.Bool("smoke-test", false, "运行自检闭环（创建数据→关闭重开 DB 验证持久化→退出 0）")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(); err != nil {
			log.Fatalf("smoke-test failed: %v", err)
		}
		fmt.Println("smoke-test OK")
		return
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	svc := service.New(st)
	srv := httpapi.New(svc, st, *addr, *dbPath)
	log.Printf("seedgerm listening on %s (db=%s)", *addr, *dbPath)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// runSmokeTest 自检闭环：创建试验/种子/图像/环境/阶段/观察/结果，关闭重开 DB 验证持久化与重启恢复。
func runSmokeTest() error {
	tmp, err := os.MkdirTemp("", "seedgerm-smoke-")
	if err != nil {
		return fmt.Errorf("mkdir temp: %w", err)
	}
	defer os.RemoveAll(tmp)
	dbFile := tmp + "/seedgerm.db"

	// 第一轮：写入
	st, err := store.Open(dbFile)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	svc := service.New(st)

	trial, err := svc.CreateTrial("SMOKE-001", "冒烟试验", "小麦")
	if err != nil {
		return fmt.Errorf("create trial: %w", err)
	}
	if _, err := svc.TransitionTrial(trial.ID, "running"); err != nil {
		return fmt.Errorf("transition running: %w", err)
	}
	seed, err := svc.CreateSeed(trial.ID, "S1")
	if err != nil {
		return fmt.Errorf("create seed: %w", err)
	}
	now := time.Now().UTC()
	if _, _, err := svc.IngestImage(seed.ID, "hash-aaa", now.Add(-2*time.Hour), 640, 480, "frame1"); err != nil {
		return fmt.Errorf("ingest image: %w", err)
	}
	// 幂等：同 hash 重复录入不应新增
	if _, created, err := svc.IngestImage(seed.ID, "hash-aaa", now.Add(-1*time.Hour), 640, 480, "dup"); err != nil || created {
		return fmt.Errorf("idempotent ingest failed: created=%v err=%v", created, err)
	}
	if _, err := svc.RecordEnv(trial.ID, now.Add(-2*time.Hour), 22.0, 60.0, "thermo-1"); err != nil {
		return fmt.Errorf("record env: %w", err)
	}
	// 检测胚根
	ev, err := svc.DetectStage(seed.ID, now, true, 0, 0.1, 0)
	if err != nil {
		return fmt.Errorf("detect stage: %w", err)
	}
	if _, err := svc.ConfirmStage(ev.ID); err != nil {
		return fmt.Errorf("confirm stage: %w", err)
	}
	if _, err := svc.AddObservation(seed.ID, "reviewer", "胚根明显"); err != nil {
		return fmt.Errorf("add observation: %w", err)
	}
	res, err := svc.DraftResult(trial.ID, "萌发率 100%")
	if err != nil {
		return fmt.Errorf("draft result: %w", err)
	}
	if _, err := svc.PublishResult(res.ID); err != nil {
		return fmt.Errorf("publish result: %w", err)
	}
	if _, err := svc.TransitionTrial(trial.ID, "reviewing"); err != nil {
		return fmt.Errorf("transition reviewing: %w", err)
	}

	// 关闭 DB
	if err := st.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	// 第二轮：重开 DB，验证持久化与重启恢复
	st2, err := store.Open(dbFile)
	if err != nil {
		return fmt.Errorf("reopen: %w", err)
	}
	defer st2.Close()
	svc2 := service.New(st2)

	t2, err := st2.GetTrial(trial.ID)
	if err != nil {
		return fmt.Errorf("reopen get trial: %w", err)
	}
	if t2.State != "reviewing" {
		return fmt.Errorf("trial state not persisted: got %s", t2.State)
	}
	seeds, err := st2.ListSeeds(trial.ID)
	if err != nil || len(seeds) != 1 {
		return fmt.Errorf("seeds not persisted: len=%d err=%v", len(seeds), err)
	}
	imgs, err := st2.ListImages(seed.ID)
	if err != nil || len(imgs) != 1 {
		return fmt.Errorf("images not persisted: len=%d err=%v", len(imgs), err)
	}
	stages, err := st2.ListStages(seed.ID)
	if err != nil || len(stages) != 1 {
		return fmt.Errorf("stages not persisted: len=%d err=%v", len(stages), err)
	}
	if stages[0].State != "confirmed" {
		return fmt.Errorf("stage state not persisted: %s", stages[0].State)
	}
	results, err := svc2.Result.List(trial.ID)
	if err != nil || len(results) != 1 {
		return fmt.Errorf("results not persisted: len=%d err=%v", len(results), err)
	}
	if results[0].State != "published" {
		return fmt.Errorf("result state not persisted: %s", results[0].State)
	}
	report, err := svc2.SelfCheck()
	if err != nil {
		return fmt.Errorf("selfcheck: %w", err)
	}
	if report.Seeds != 1 || report.Images != 1 || report.EnvSamples != 1 {
		return fmt.Errorf("selfcheck counts wrong: %+v", report)
	}
	return nil
}
