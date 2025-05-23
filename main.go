package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/veryfrank/Chirpy/internal/database"
)

var cfg = apiConfig{}
var envCfg environmentConfig

func main() {
	envCfg = GetEnvironmentConfig()
	db, err := sql.Open("postgres", envCfg.DbConnectionString)
	if err != nil {
		panic(err)
	}

	serveMux := http.NewServeMux()
	server := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	cfg.Db = db
	cfg.DbQueries = database.New(db)

	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))

	registerApiHanlders(serveMux)
	registerAdminHandlers(serveMux)

	server.ListenAndServe()
}

func registerApiHanlders(serveMux *http.ServeMux) {
	serveMux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	serveMux.HandleFunc(fmt.Sprintf("GET /api/chirps/{%v}", chirpIdPathValueName), handleGetChirp)
	serveMux.HandleFunc("GET /api/chirps", handleGetAllChirps)
	serveMux.HandleFunc("POST /api/chirps", handlePostChirp)
	serveMux.HandleFunc("GET /api/users", handleGetUsers)
	serveMux.HandleFunc("POST /api/users", handleUserCreation)
}

func registerAdminHandlers(serveMux *http.ServeMux) {
	serveMux.HandleFunc("GET /admin/metrics", cfg.showMetricsHandler)

	serveMux.HandleFunc("POST /admin/reset", cfg.resetMetricsHandler)
}

type apiConfig struct {
	fileserverHits atomic.Int32
	Db             *sql.DB
	DbQueries      *database.Queries
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

	if envCfg.Platform == "dev" {
		err := cfg.DbQueries.DeleteUsers(req.Context())
		if err != nil {
			log.Println(err)
			respWriter.WriteHeader(500)
		}
	} else {
		respWriter.WriteHeader(403)
	}
}

type jsonErrorResp struct {
	Error string `json:"error"`
}

type jsonEmail struct {
	Email string `json:"email"`
}

type environmentConfig struct {
	DbConnectionString string
	Platform           string
}

func GetEnvironmentConfig() environmentConfig {
	godotenv.Load()

	envCfg := environmentConfig{
		DbConnectionString: os.Getenv("DB_URL"),
		Platform:           os.Getenv("PLATFORM"),
	}

	return envCfg
}
