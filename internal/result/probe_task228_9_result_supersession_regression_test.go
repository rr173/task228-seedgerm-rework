package result_test

import (
	"path/filepath"
	"testing"

	"task228-seedgerm/internal/model"
	"task228-seedgerm/internal/result"
	"task228-seedgerm/internal/store"
)

func TestTask228Bug09PublishingRevisionSupersedesPublishedPredecessor(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	tr, _ := st.CreateTrial("R9", "publish", "rice", model.TrialReviewing)
	svc := result.New(st)
	first, _ := svc.Draft(tr.ID, "first")
	if _, err := svc.Publish(first.ID); err != nil { t.Fatal(err) }
	second, _ := svc.Draft(tr.ID, "second")
	if _, err := svc.Publish(second.ID); err != nil { t.Fatal(err) }
	items, _ := svc.List(tr.ID)
	if items[0].State != model.ResultSuperseded || items[1].State != model.ResultPublished { t.Fatalf("broken predecessor chain: %+v", items) }
}
