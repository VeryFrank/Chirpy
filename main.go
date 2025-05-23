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
			//length ok, check for prafanity
			words := strings.Split(jsonBody.Body, " ")
			cleanedBody := ""
			for i, word := range words {
				lowerWord := strings.ToLower(word)

				if i > 0 {
					cleanedBody += " "
				}

				if profanityMap[lowerWord] {
					cleanedBody += "****"

				} else {
					cleanedBody += word
				}
			}

			answerJson := jsonValidResp{
				CleanedBody: cleanedBody,
			}

			jsonBytes, err := json.Marshal(answerJson)
			if err != nil {
				log.Printf("Error marshaling json %s", err)
				return
			}

			w.WriteHeader(200)
			w.Header().Add("content-type", "application/json")
			w.Write(jsonBytes)
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
	emailNullString := sql.NullString{String: emailWrapper.Email, Valid: true}
	user, err := cfg.DbQueries.CreateUser(r.Context(), emailNullString)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	creationAnswer := User{
		ID:        user.ID.UUID,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
		Email:     user.Email.String,
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

type jsonBody struct {
	Body string `json:"body"`
}

type jsonErrorResp struct {
	Error string `json:"error"`
}

type jsonValidResp struct {
	CleanedBody string `json:"cleaned_body"`
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
		ID:        dbUser.ID.UUID,
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
		Email:     dbUser.Email.String,
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
