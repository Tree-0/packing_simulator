package frontend

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
)

//go:embed web/dist
var embeddedWeb embed.FS

type simulationsResponse struct {
	Simulations []SimulationRecording `json:"simulations"`
}

// NewHandler returns the read-only API and embedded React application.
func NewHandler(recordings []SimulationRecording) http.Handler {
	response := simulationsResponse{
		Simulations: append([]SimulationRecording(nil), recordings...),
	}

	dist, err := fs.Sub(embeddedWeb, "web/dist")
	if err != nil {
		panic(err)
	}
	assets := http.FileServer(http.FS(dist))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/simulations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "encode simulations", http.StatusInternalServerError)
		}
	})
	mux.Handle("/", methodHandler(assets))
	return mux
}

func methodHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}
