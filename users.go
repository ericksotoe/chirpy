package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ericksotoe/chirpy/internal/auth"
	"github.com/ericksotoe/chirpy/internal/database"
)

type emailAndPassword struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userEmailAndPassword := emailAndPassword{}
	err := decoder.Decode(&userEmailAndPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	hash, err := auth.HashPassword(userEmailAndPassword.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	user, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{
		Email:          userEmailAndPassword.Email,
		HashedPassword: hash,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	addedUser := User{ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email}

	res, err := json.Marshal(addedUser)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(res)

}

func (cfg *apiConfig) loginUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userEmailAndPassword := emailAndPassword{}
	err := decoder.Decode(&userEmailAndPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	user, err := cfg.db.GetUserUsingEmail(context.Background(), userEmailAndPassword.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	match, err := auth.CheckPasswordHash(userEmailAndPassword.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	if !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}

	addedUser := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email}

	respondWithJSON(w, http.StatusOK, addedUser)
}
