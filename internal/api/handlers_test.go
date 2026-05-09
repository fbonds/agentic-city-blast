package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mferree/agent-city/internal/model"
)

type fakeState struct {
	state model.CityState
}

func (f *fakeState) GetState() model.CityState { return f.state }

func TestHandleGetBuilding(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"slashed ID", "internal/hub/hub.go"},
		{"simple ID", "main.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(&fakeState{
				state: model.CityState{
					Buildings: []model.Building{
						{ID: tt.id, Label: tt.id},
					},
				},
			})
			mux := http.NewServeMux()
			srv.Register(mux)

			req := httptest.NewRequest("GET", "/api/buildings/"+tt.id, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			var b model.Building
			if err := json.NewDecoder(w.Body).Decode(&b); err != nil {
				t.Fatalf("decoding response: %v", err)
			}
			if b.ID != tt.id {
				t.Fatalf("expected ID %q, got %q", tt.id, b.ID)
			}
		})
	}
}
