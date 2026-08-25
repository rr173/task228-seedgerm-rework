package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task228-seedgerm/internal/httpapi"
	"task228-seedgerm/internal/service"
	"task228-seedgerm/internal/store"
)

func TestHandlerServesWebWorkbenchAndResourceRoutes(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/http.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(httpapi.New(service.New(st), st, ":0", "").Handler())
	defer srv.Close()

	page, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer page.Body.Close()
	if page.StatusCode != http.StatusOK {
		t.Fatalf("web page status=%d", page.StatusCode)
	}

	body := `{"code":"HTTP-001","name":"接口试验","species":"小麦"}`
	resp, err := http.Post(srv.URL+"/api/trials", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create trial status=%d", resp.StatusCode)
	}
	var trial struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&trial); err != nil {
		t.Fatal(err)
	}
	if trial.ID == 0 {
		t.Fatal("created trial has no id")
	}

	list, err := http.Get(srv.URL + "/api/trials")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Body.Close()
	if list.StatusCode != http.StatusOK {
		t.Fatalf("list trial status=%d", list.StatusCode)
	}
}
