package main

import (
	"fmt"
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

	serveMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	serveMux.HandleFunc("GET /metrics", apiConfig.showMetricsHandler)
	serveMux.HandleFunc("POST /reset", apiConfig.resetMetricsHandler)

	server.ListenAndServe()
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
	respWriter.Header().Add("content-type", "text/plain; charset=utf-8")
	respWriter.WriteHeader(200)

	msg := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	respWriter.Write([]byte(msg))
}

func (cfg *apiConfig) resetMetricsHandler(respWriter http.ResponseWriter, req *http.Request) {
	respWriter.Header().Add("content-type", "text/plain; charset=utf-8")
	respWriter.WriteHeader(200)

	old := cfg.fileserverHits.Swap(0)
	msg := fmt.Sprintf("Hits reset, previous hits was %v", old)
	respWriter.Write([]byte(msg))

}
