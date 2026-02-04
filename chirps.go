package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ericksotoe/chirpy/internal/database"
	"github.com/google/uuid"
)

type parameters struct {
	Body   string `json:"body"`
	UserId string `json:"user_id"`
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

	if params.Body == "" || params.UserId == "" {
		respondWithError(w, http.StatusBadRequest, "userid or body was left empty")
		return
	}

	const maxChirpLen = 140
	if len(params.Body) > maxChirpLen {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanUpBadWords(&params)
	parsedID, err := uuid.Parse(params.UserId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong when parsing string to uuid")
		return
	}
	chirpParams := database.CreateChirpParams{
		Body:   params.Body,
		UserID: parsedID}

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
