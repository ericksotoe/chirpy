package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ericksotoe/chirpy/internal/auth"
	"github.com/ericksotoe/chirpy/internal/database"
	"github.com/google/uuid"
)

type parameters struct {
	Body string `json:"body"`
}

type ChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type SliceChirpResponse struct {
	SliceChirp []ChirpResponse
}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	unverifiedToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	userID, err := auth.ValidateJWT(unverifiedToken, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if params.Body == "" || userID == uuid.Nil {
		respondWithError(w, http.StatusBadRequest, "userid or body was left empty")
		return
	}

	const maxChirpLen = 140
	if len(params.Body) > maxChirpLen {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanUpBadWords(&params)

	chirpParams := database.CreateChirpParams{
		Body:   params.Body,
		UserID: userID}

	chirp, err := cfg.db.CreateChirp(context.Background(), chirpParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong when creating chirp")
		return
	}
	respondWithJSON(w, http.StatusCreated, ChirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type returnErrVal struct {
		Error string `json:"error"`
	}

	respBody := returnErrVal{
		Error: msg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func cleanUpBadWords(params *parameters) {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	wordsToCheck := params.Body
	words := strings.Split(wordsToCheck, " ")
	for i, word := range words {
		if slices.Contains(badWords, strings.ToLower(word)) {
			words[i] = "****"
		}
	}
	wordsToCheck = strings.Join(words, " ")
	params.Body = wordsToCheck
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirpSlice, err := cfg.db.GetChirps(context.Background())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something wrong happened when retrieving all the chirps from db")
		return
	}
	chirpResponseSlice := []ChirpResponse{}
	for _, chirp := range chirpSlice {
		chirpResponseSlice = append(chirpResponseSlice, ChirpResponse{ID: chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID})
	}

	respondWithJSON(w, http.StatusOK, chirpResponseSlice)
}

func (cfg *apiConfig) getChirpsByIDHandler(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp Id")
		return
	}
	chirp, err := cfg.db.GetChirpsByID(context.Background(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Something went wrong when retrieving the chirp by id")
		return
	}

	respondWithJSON(w, http.StatusOK, ChirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
}
