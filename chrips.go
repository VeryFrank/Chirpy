package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/veryfrank/Chirpy/internal/database"
)

var profanityMap = map[string]bool{"kerfuffle": true, "sharbert": true, "fornax": true}

const (
	chripMaxLength       = 140
	chirpIdPathValueName = "chripID"
)

func handleGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpIdString := r.PathValue(chirpIdPathValueName)
	dbChirp, err := cfg.DbQueries.GetChirp(r.Context(), uuid.MustParse(chirpIdString))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(404)
			return
		} else {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
	}

	chirp := GetChirpFromDbChirp(dbChirp)
	bytes, err := json.Marshal(chirp)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Header().Add("content-type", "application/json")
	w.Write(bytes)

}

func handleGetAllChirps(w http.ResponseWriter, r *http.Request) {
	allDbChirps, err := cfg.DbQueries.GetAllChirps(r.Context())
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	chirpCount := len(allDbChirps)
	chirps := make([]Chirp, chirpCount)
	for i, dbChirp := range allDbChirps {
		insertIndex := chirpCount - 1 - i
		chirps[insertIndex] = GetChirpFromDbChirp(dbChirp)
	}

	bytes, err := json.Marshal(chirps)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
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

	chirp := GetChirpFromDbChirp(dbChirp)
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

func GetChirpFromDbChirp(dbChirp database.Chirp) (chirp Chirp) {
	chirp = Chirp{
		ID:        dbChirp.ID,
		UserID:    dbChirp.UserID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body.String,
	}

	return chirp
}
