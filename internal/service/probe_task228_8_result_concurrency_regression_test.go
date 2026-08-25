package service_test

import (
	"path/filepath"
	"sync"
	"testing"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug08ConcurrentDraftsKeepResultVersionChain(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	svc := service.New(st)
	tr, _ := svc.CreateTrial("R8", "versions", "wheat")
	if _, err := svc.TransitionTrial(tr.ID, model.TrialRunning); err != nil { t.Fatal(err) }
	if _, err := svc.TransitionTrial(tr.ID, model.TrialReviewing); err != nil { t.Fatal(err) }
	const n = 24
	results := make(chan model.TrialResult, n)
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, err := svc.DraftResult(tr.ID, string(rune('a'+i)))
			if err != nil { errs <- err; return }
			results <- r
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs { t.Fatalf("concurrent draft failed: %v", err) }
	seen := map[int]bool{}
	for r := range results {
		if seen[r.Version] { t.Fatalf("duplicate version %d", r.Version) }
		seen[r.Version] = true
	}
	if len(seen) != n { t.Fatalf("missing versions: got %d want %d", len(seen), n) }
}
