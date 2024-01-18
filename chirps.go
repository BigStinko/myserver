package main

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (cfg *apiConfig) chirpPostHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string	`json:"body"`
	}

	tok, err := GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	idStr, err := cfg.validateJWT(tok, "access", cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	params, err := decodeParameters[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleaned := clean(params.Body)

	chirp, err := cfg.db.CreateChirp(id, cleaned)
	if err != nil {
		msg := fmt.Sprintf("Couldn't create chirp: %s", err)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}
	respondWithJSON(w, http.StatusCreated, chirp)
}

func (cfg *apiConfig) chirpGetHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps()
	idStr := r.URL.Query().Get("author_id")
	if idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		trimmed := []Chirp{}
		for _, chirp := range chirps {
			if chirp.AuthorId == id {
				trimmed = append(trimmed, chirp)
			}
		}
		chirps = trimmed
	}

	sortStr := r.URL.Query().Get("sort")
	if sortStr == "desc" {
		slices.Reverse[[]Chirp](chirps)
	}

	if err != nil {
		msg := fmt.Sprintf("Couldn't get chirps from db: %s", err)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}
	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) chirpGetIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
	}

	chirp, err := cfg.db.GetChirp(id)
	if err != nil {
		log.Print(err)
		w.WriteHeader(404)
		return
	}

	respondWithJSON(w, http.StatusOK, chirp)
}

func (cfg *apiConfig) chirpDeleteIdHandler(w http.ResponseWriter, r *http.Request) {

	tok, err := GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	idStr, err := cfg.validateJWT(tok, "access", cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	authorId, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
	}

	if !cfg.db.IsChirpAuthor(authorId, id) {
		respondWithError(w, http.StatusForbidden, "Not author of chirp")
		return
	}

	err = cfg.db.DeleteChirp(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, struct{}{})
}
