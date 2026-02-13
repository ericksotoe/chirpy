package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"sort"
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

	dbChirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps")
		return
	}

	authorID := uuid.Nil
	authorIDString := r.URL.Query().Get("author_id")
	if authorIDString != "" {
		authorID, err = uuid.Parse(authorIDString)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid author ID")
			return
		}
	}

	sortOrder := r.URL.Query().Get("sort")

	chirps := []ChirpResponse{}
	for _, dbChirp := range dbChirps {
		if authorID != uuid.Nil && dbChirp.UserID != authorID {
			continue
		}

		chirps = append(chirps, ChirpResponse{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			UserID:    dbChirp.UserID,
			Body:      dbChirp.Body,
		})
	}

	if sortOrder == "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
		})
	} else {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
		})
	}

	respondWithJSON(w, http.StatusOK, chirps)
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

func (cfg *apiConfig) deleteChirpHandler(w http.ResponseWriter, r *http.Request) {
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

	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp Id")
		return
	}

	ctx := context.Background()
	chirp, err := cfg.db.GetChirpsByID(ctx, chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Something went wrong when retrieving the chirp by id (it might not exist)")
		return
	}

	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Chirps can only be deleted by their creators")
		return
	}

	err = cfg.db.DeleteChirpsByID(ctx, chirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting the users chirp")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type UpgradeEvent struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) addChirpyRedHandler(w http.ResponseWriter, r *http.Request) {
	api, err := auth.GetBearerApi(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Api Key is malformed or missing")
		return
	}

	if api != cfg.polkaApiKey {
		respondWithError(w, http.StatusUnauthorized, "malformed / bad signature / expired api key")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := UpgradeEvent{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	userID, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error chaning the string userid to a uuid")
		return
	}

	_, err = cfg.db.UpgradeToChirpyRed(context.Background(), userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
