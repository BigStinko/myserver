package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v5"
)


func (cfg *apiConfig) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}

	params, err := decodeParameters[parameters](r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	user, err := cfg.db.GetUserFromEmail(params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "passwords don't match")
		return
	}

	idStr := strconv.Itoa(user.Id)
	jwtAccessToken := generateJWT("access", idStr)
	jwtRefreshToken := generateJWT("refresh", idStr)

	accessToken, err := jwtAccessToken.SignedString([]byte(cfg.jwtSecret))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	refreshToken, err := jwtRefreshToken.SignedString([]byte(cfg.jwtSecret))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.db.AddToken(refreshToken)

	respondWithJSON(w, http.StatusOK, 
		struct{
			Id int `json:"id"`
			Email string `json:"email"`
			IsChirpyRed bool `json:"is_chirpy_red"`
			Token string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		}{
			Id: user.Id,
			Email: user.Email,
			IsChirpyRed: user.IsChirpyRed,
			Token: accessToken,
			RefreshToken: refreshToken,
		},
	)
}

func (cfg *apiConfig) refreshPostHandler(w http.ResponseWriter, r *http.Request) {
	tok, err := GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id, err := cfg.validateJWT(tok, "refresh", cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	jwtAccessToken := generateJWT("access", id)
	accessToken, err := jwtAccessToken.SignedString([]byte(cfg.jwtSecret))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK,
		struct{
			Token string `json:"token"`
		}{
			Token: accessToken,
		})
}

func (cfg *apiConfig) revokePostHandler(w http.ResponseWriter, r *http.Request) {
	tok, err := GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	err = cfg.db.RevokeToken(tok)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (cfg *apiConfig) validateJWT(tokenString, tokenType, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil },
	)
	if err != nil { return "", err }

	id, err := token.Claims.GetSubject()
	if err != nil { return "", err }
	
	issuer, err := token.Claims.GetIssuer()
	if err != nil { return "", err }

	if issuer != "chirpy-" + tokenType {
		return "", errors.New("invalid token type")
	}

	if tokenType == "refresh" && !cfg.db.ValidToken(tokenString) {
		return "", ErrRevokedToken
	}

	return id, nil
}

func generateJWT(tokenType, id string) *jwt.Token {
	switch tokenType {
	case "access":
		return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Issuer: "chirpy-access",
			IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			Subject: id,
		})
	case "refresh":
		return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Issuer: "chirpy-refresh",
			IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Duration(60 * 24) * time.Hour)),
			Subject: id,
		})
	default:
		return nil
	}
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header not included")
	}
	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "Bearer" {
		return "", errors.New("malformed authorization header")
	}

	return splitAuth[1], nil
}
