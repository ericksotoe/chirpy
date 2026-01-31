package main

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"
)

type parameters struct {
	Body string `json:"body"`
}

func chirpHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	const maxChirpLen = 140
	if len(params.Body) > maxChirpLen {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanUpBadWords(&params)
	type validateResp struct {
		CleanedBody string `json:"cleaned_body"`
	}

	resp := validateResp{
		CleanedBody: params.Body,
	}
	respondWithJSON(w, http.StatusOK, resp)
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
