package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/veryfrank/Chirpy/internal/auth"
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
	var jsonCreateUser jsonCreateUser
	err := decoder.Decode(&jsonCreateUser)
	if err != nil {
		log.Println(err)

		w.WriteHeader(500)
		return
	}

	HasValidEmail := jsonCreateUser.HasValidEmail()
	if !HasValidEmail {
		errResp := jsonErrorResp{
			Error: "invalid email",
		}

		SendErrorResponse(errResp, 400, w, r)
		return
	}

	hasValidPassword := jsonCreateUser.HasValidPassword()
	if !hasValidPassword {
		errResp := jsonErrorResp{
			Error: "invalid password",
		}

		SendErrorResponse(errResp, 400, w, r)
		return
	}

	hashedPw, err := auth.HashPassword(jsonCreateUser.Password)
	if err != nil {
		log.Println(err)

		w.WriteHeader(500)
		return
	}

	createUserParams := database.CreateUserParams{
		Email:          jsonCreateUser.Email,
		HashedPassword: hashedPw,
	}

	user, err := cfg.DbQueries.CreateUser(r.Context(), createUserParams)
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

func handleUserLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var login jsonLogin
	err := decoder.Decode(&login)
	if err != nil {
		log.Println(err)

		w.WriteHeader(500)
		return
	}

	dbUser, err := cfg.DbQueries.GetUser(r.Context(), login.Email)
	if err != nil {
		log.Println(err)

		jsonErrorResp := jsonErrorResp{
			Error: "Incorrect email or password",
		}
		SendErrorResponse(jsonErrorResp, 401, w, r)

		return
	}

	err = auth.CheckPasswordHash(dbUser.HashedPassword, login.Password)
	if err != nil {
		jsonErrorResp := jsonErrorResp{
			Error: "Incorrect email or password",
		}
		SendErrorResponse(jsonErrorResp, 401, w, r)

		return
	}

	user := getUserFromDbUser(dbUser)
	bytes, err := json.Marshal(user)
	if err != nil {
		println(err)

		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Header().Add("content-type", "application/json")
	w.Write(bytes)
}

func SendErrorResponse(errResp jsonErrorResp, statusCode int, w http.ResponseWriter, r *http.Request) {
	bytes, err := json.Marshal(errResp)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(statusCode)
	w.Header().Add("content-type", "application/json")
	w.Write(bytes)
}

type User struct {
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        uuid.UUID `json:"id"`
}

func getUserFromDbUser(dbUser database.User) User {
	return User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
}

type jsonCreateUser struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

func (j *jsonCreateUser) HasValidEmail() bool {
	return (len(j.Email) >= 5)
}

func (j *jsonCreateUser) HasValidPassword() bool {
	return len(j.Password) >= 5
}

type jsonLogin struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}
