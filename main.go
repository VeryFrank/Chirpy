package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

func main() {
	serveMux := http.NewServeMux()
	server := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	apiConfig := apiConfig{}

	serveMux.Handle("/app/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	registerApiHanlders(serveMux, &apiConfig)
	registerAdminHandlers(serveMux, &apiConfig)

	server.ListenAndServe()
}

func registerApiHanlders(serveMux *http.ServeMux, apiConfig *apiConfig) {
	serveMux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	serveMux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		jsonBody := jsonBody{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&jsonBody)
		if err != nil {
			log.Printf("Error decoding parameters: %s", err)
			w.WriteHeader(500)

			jsonErrorResp := jsonErrorResp{
				Error: "Something went wrong",
			}

			jsonBytes, err := json.Marshal(jsonErrorResp)
			if err != nil {
				log.Printf("Error marshaling json: %s", err)
				return
			}

			w.Write(jsonBytes)
			return
		}

		if len(jsonBody.Body) <= 140 {
			w.WriteHeader(200)
			w.Header().Add("content-type", "application/json")
			w.Write([]byte("{\"valid\":true}"))
		} else {
			jsonErrorResp := jsonErrorResp{
				Error: "Chirp is too long",
			}

			jsonBytes, err := json.Marshal(jsonErrorResp)
			if err != nil {
				log.Printf("Error marshaling json %s", err)
				return
			}

			w.WriteHeader(400)
			w.Header().Add("content-type", "application/json")
			w.Write(jsonBytes)
		}
	})

}

func registerAdminHandlers(serveMux *http.ServeMux, apiConfig *apiConfig) {
	serveMux.HandleFunc("GET /admin/metrics", apiConfig.showMetricsHandler)

	serveMux.HandleFunc("POST /admin/reset", apiConfig.resetMetricsHandler)
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) showMetricsHandler(respWriter http.ResponseWriter, req *http.Request) {
	respWriter.Header().Add("content-type", "text/html; charset=utf-8")
	respWriter.WriteHeader(200)

	html := `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`

	msg := fmt.Sprintf(html, cfg.fileserverHits.Load())
	respWriter.Write([]byte(msg))
}

func (cfg *apiConfig) resetMetricsHandler(respWriter http.ResponseWriter, req *http.Request) {
	respWriter.Header().Add("content-type", "text/plain; charset=utf-8")
	respWriter.WriteHeader(200)

	old := cfg.fileserverHits.Swap(0)
	msg := fmt.Sprintf(`Hits reset, previous hits was %v`, old)
	respWriter.Write([]byte(msg))

}

type jsonBody struct {
	Body string `json:"body"`
}

type jsonErrorResp struct {
	Error string `json:"error"`
}

type jsonValidResp struct {
	Valid bool `json:"valid"`
}
