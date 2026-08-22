package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesSimulationAPI(t *testing.T) {
	recording := SimulationRecording{ID: "one", Frames: []SimulationFrame{}}
	request := httptest.NewRequest(http.MethodGet, "/api/simulations", nil)
	response := httptest.NewRecorder()

	NewHandler([]SimulationRecording{recording}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Content-Type = %q; want JSON", contentType)
	}
	var body simulationsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Simulations) != 1 || body.Simulations[0].ID != "one" {
		t.Errorf("response = %+v", body)
	}
}

func TestHandlerRejectsAPIMutations(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/simulations", nil)
	response := httptest.NewRecorder()

	NewHandler(nil).ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Errorf("status = %d, Allow = %q", response.Code, response.Header().Get("Allow"))
	}
}

func TestHandlerServesEmbeddedApp(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	NewHandler(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", response.Code)
	}
	if !strings.Contains(response.Body.String(), "root") {
		t.Errorf("embedded index does not contain React root: %s", response.Body.String())
	}
}
