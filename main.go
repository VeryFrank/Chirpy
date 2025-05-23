package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/veryfrank/Chirpy/internal/database"
)

var profanityMap = map[string]bool{"kerfuffle": true, "sharbert": true, "fornax": true}
var cfg = apiConfig{}
var envCfg environmentConfig

const (
	chripMaxLength = 140
)

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

	serveMux.HandleFunc("POST /api/chirps", handlePostChirp)
	serveMux.HandleFunc("GET /api/users", handleGetUsers)
	serveMux.HandleFunc("POST /api/users", handleUserCreation)
}

func registerAdminHandlers(serveMux *http.ServeMux) {
	serveMux.HandleFunc("GET /admin/metrics", cfg.showMetricsHandler)

	serveMux.HandleFunc("POST /admin/reset", cfg.resetMetricsHandler)
}

func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	dbUsers, err := cfg.DbQueries.GetUsers(r.Context())
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	userSlice := make([]User, 0, len(dbUsers))
	for _, dbUser := range dbUsers {
		usr := getUserFromDbUser(dbUser)
		userSlice = append(userSlice, usr)
	}

	bytes, err := json.Marshal(userSlice)
	if err != nil {
		w.WriteHeader(500)
		log.Println(err)
	}

	w.WriteHeader(200)
	w.Header().Add("content-type", "application/json")
	w.Write(bytes)
}

func handleUserCreation(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var emailWrapper jsonEmail
	err := decoder.Decode(&emailWrapper)
	if err != nil {
		log.Println(err)

		w.WriteHeader(500)
		return
	}

	if len(emailWrapper.Email) <= 5 {
		errResp := jsonErrorResp{
			Error: "invalid email",
		}

		bytes, err := json.Marshal(errResp)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(400)
		w.Header().Add("content-type", "application/json")
		w.Write(bytes)

		return
	}
	user, err := cfg.DbQueries.CreateUser(r.Context(), emailWrapper.Email)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	creationAnswer := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	bytes, err := json.Marshal(creationAnswer)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(201)
	w.Header().Add("content-type", "application/json")
	w.Write(bytes)
}

func handlePostChirp(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	postChirp := postChirp{}
	err := decoder.Decode(&postChirp)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	chirpLength := len(postChirp.Body)
	if chirpLength > chripMaxLength {
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

		return
	} //chirp too long?

	postChirp.Body = cleanChirp(postChirp.Body)
	params := database.CreateChirpParams{
		UserID: postChirp.UserId,
		Body:   sql.NullString{String: postChirp.Body, Valid: true},
	}

	dbChirp, err := cfg.DbQueries.CreateChirp(r.Context(), params)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	chirp := GetChirpFromDb(dbChirp)
	bytes, err := json.Marshal(chirp)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(201)
	w.Header().Add("content-type", "application/json")
	w.Write(bytes)
}

func cleanChirp(chirp string) (cleanedChirp string) {
	words := strings.Split(chirp, " ")
	cleanedChirp = ""
	for i, word := range words {
		lowerWord := strings.ToLower(word)

		if i > 0 {
			cleanedChirp += " "
		}

		if profanityMap[lowerWord] {
			cleanedChirp += "****"

		} else {
			cleanedChirp += word
		}
	}

	return cleanedChirp
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

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func getUserFromDbUser(dbUser database.User) User {
	return User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
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

type postChirp struct {
	Body   string    `json:"body"`
	UserId uuid.UUID `json:"user_id"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
}

func GetChirpFromDb(dbChirp database.Chirp) (chirp Chirp) {
	chirp = Chirp{
		ID:        dbChirp.ID,
		UserID:    dbChirp.UserID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body.String,
	}

	return chirp
}
