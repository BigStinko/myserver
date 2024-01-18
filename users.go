package main

import (
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func (cfg *apiConfig) userPostHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}

	params, err := decodeParameters[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	encPass, err := bcrypt.GenerateFromPassword([]byte(params.Password), 0)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't encrypt password")
	}

	user, err := cfg.db.CreateUser(params.Email, string(encPass))
	if err != nil {
		msg := fmt.Sprintf("Couldn't create user: %s", err)
		respondWithError(w, http.StatusInternalServerError, msg)
		return
	}
	respondWithJSON(w, http.StatusCreated,
		struct{
			Id int `json:"id"`
			Email string `json:"email"`
			IsChirpyRed bool `json:"is_chirpy_red"`			
		}{
			Id: user.Id,
			Email: user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	)
}

func (cfg *apiConfig) userPutHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
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

	params, err := decodeParameters[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	encPass, err := bcrypt.GenerateFromPassword([]byte(params.Password), 0)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	user, err := cfg.db.UpdateUser(id, params.Email, string(encPass))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, 
		struct{
			Id int `json:"id"`
			Email string `json:"email"`
			IsChirpyRed bool `json:"is_chirpy_red"`
		}{
			Id: user.Id,
			Email: user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	)
}
