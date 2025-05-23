package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/veryfrank/Chirpy/internal/database"
)

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
