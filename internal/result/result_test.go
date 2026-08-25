package result_test

import (
	"path/filepath"
	"sync"
	"testing"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/result"
	"task228-seedgerm/internal/store"
)

func TestPublishedResultVersionSupersedesPreviousVersion(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "result.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	trial, err := st.CreateTrial("RESULT-001", "结果测试", "小麦", model.TrialReviewing)
	if err != nil {
		t.Fatal(err)
	}
	svc := result.New(st)
	first, err := svc.Draft(trial.ID, "首版")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Publish(first.ID); err != nil {
		t.Fatal(err)
	}
	second, err := svc.Draft(trial.ID, "修订版")
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 || second.PrevVersion != 1 {
		t.Fatalf("unexpected version chain: %+v", second)
	}
	if _, err := svc.Publish(second.ID); err != nil {
		t.Fatal(err)
	}
	items, err := svc.List(trial.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].State != model.ResultSuperseded || items[1].State != model.ResultPublished {
		t.Fatalf("previous result was not superseded: %+v", items)
	}
}

// TestConcurrentDraftProducesConsecutiveUniqueVersions 并发起草同一试验：
// 所有请求都必须成功，版本号连续且唯一（1..N），prev_version 指向上一版本形成完整版本链。
func TestConcurrentDraftProducesConsecutiveUniqueVersions(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "result_concurrent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	trial, err := st.CreateTrial("CONC-001", "并发结果", "小麦", model.TrialReviewing)
	if err != nil {
		t.Fatal(err)
	}
	svc := result.New(st)

	const n = 16
	var wg sync.WaitGroup
	results := make([]model.TrialResult, n)
	errs := make([]error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			r, err := svc.Draft(trial.ID, "并发草稿")
			results[i] = r
			errs[i] = err
		}(i)
	}
	wg.Wait()

	// 所有请求都应成功
	for i, err := range errs {
		if err != nil {
			t.Fatalf("draft %d failed: %v", i, err)
		}
	}
	// 版本号唯一且落在 [1, n]
	versions := make(map[int]bool, n)
	for i, r := range results {
		if r.Version < 1 || r.Version > n {
			t.Fatalf("draft %d version out of range: %d", i, r.Version)
		}
		if versions[r.Version] {
			t.Fatalf("duplicate version %d", r.Version)
		}
		versions[r.Version] = true
	}
	// 版本号连续：1..N 全部出现
	if len(versions) != n {
		t.Fatalf("expected %d unique versions, got %d", n, len(versions))
	}
	for v := 1; v <= n; v++ {
		if !versions[v] {
			t.Fatalf("missing version %d in chain", v)
		}
	}
	// 完整版本链：每个版本的 prev_version 指向上一版本（首版本为 0）
	all, err := svc.List(trial.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != n {
		t.Fatalf("expected %d persisted results, got %d", n, len(all))
	}
	versionToPrev := make(map[int]int, n)
	for _, r := range all {
		versionToPrev[r.Version] = r.PrevVersion
	}
	for v := 1; v <= n; v++ {
		wantPrev := 0
		if v > 1 {
			wantPrev = v - 1
		}
		if versionToPrev[v] != wantPrev {
			t.Fatalf("version %d prev_version want %d, got %d", v, wantPrev, versionToPrev[v])
		}
	}
}
