package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ericksotoe/chirpy/internal/auth"
	"github.com/ericksotoe/chirpy/internal/database"
	"github.com/google/uuid"
)

type UserWithToken struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

type emailAndPassword struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type responseToken struct {
	Token string `json:"token"`
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userEmailAndPassword := emailAndPassword{}
	err := decoder.Decode(&userEmailAndPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading the password and email from the request body")
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

	addedUser := User{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed}

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

	ctx := context.Background()
	user, err := cfg.db.GetUserUsingEmail(ctx, userEmailAndPassword.Email)
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

	token, err := auth.MakeJWT(user.ID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}
	sixtyDays := (time.Hour * 24) * 60
	params := database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(sixtyDays),
	}
	_, err = cfg.db.CreateRefreshToken(ctx, params)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	addedUser := UserWithToken{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		IsChirpyRed:  user.IsChirpyRed,
		Token:        token,
		RefreshToken: refreshToken}

	respondWithJSON(w, http.StatusOK, addedUser)
}

func (cfg *apiConfig) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	user, err := cfg.db.GetUserFromRefreshToken(context.Background(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "The user's token has been rejected or doesn't exist")
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Can't create a new JWT for the user")
		return
	}

	jsonToken := responseToken{
		Token: token,
	}

	respondWithJSON(w, http.StatusOK, jsonToken)
}

func (cfg *apiConfig) revokeTokenHandler(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	_, err = cfg.db.RevokeToken(context.Background(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Refresh token was not found in the db")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Access token is malformed or missing")
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "malformed / bad signature / expired token")
		return
	}

	decoder := json.NewDecoder(r.Body)
	userEmailAndPassword := emailAndPassword{}
	err = decoder.Decode(&userEmailAndPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading the password and email from the request body")
		return
	}

	hashedPass, err := auth.HashPassword(userEmailAndPassword.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing the password passed in")
		return
	}

	params := database.UpdateUserPassEmailParams{
		ID:             userID,
		Email:          userEmailAndPassword.Email,
		HashedPassword: hashedPass,
	}

	responseUser, err := cfg.db.UpdateUserPassEmail(context.Background(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating the users email and password")
		return
	}

	response := UserResponse{
		ID:          responseUser.ID,
		CreatedAt:   responseUser.CreatedAt,
		UpdatedAt:   responseUser.UpdatedAt,
		Email:       responseUser.Email,
		IsChirpyRed: responseUser.IsChirpyRed,
	}

	respondWithJSON(w, http.StatusOK, response)

}
