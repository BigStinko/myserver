package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

type apiConfig struct {
	fileserverHits int
	db *DB
	jwtSecret string
	polkaKey string
}

const PORT = "8080"
const ROOTPATH = "."

func main() {
	godotenv.Load()
	dbs, err := NewDB("database.json")
	if err != nil {
		fmt.Printf("Error loading database: %s", err)
		return
	}

	apicfg := apiConfig{
		fileserverHits: 0,
		db: dbs,
		jwtSecret: os.Getenv("JWT_SECRET"),
		polkaKey: os.Getenv("POLKA_KEY"),
	}
	mainMux := chi.NewRouter()
	apiMux := chi.NewRouter()
	adminMux := chi.NewRouter()

	fsHandler := apicfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(ROOTPATH))))

	mainMux.Handle("/app/*", fsHandler)
	mainMux.Handle("/app", fsHandler)
	apiMux.Get("/healthz", readinessHandler)
	apiMux.Get("/reset", apicfg.resetHandler)
	apiMux.Post("/chirps", apicfg.chirpPostHandler)
	apiMux.Post("/users", apicfg.userPostHandler)
	apiMux.Put("/users", apicfg.userPutHandler)
	apiMux.Post("/login", apicfg.loginPostHandler)
	apiMux.Post("/refresh", apicfg.refreshPostHandler)
	apiMux.Post("/revoke", apicfg.revokePostHandler)
	apiMux.Get("/chirps", apicfg.chirpGetHandler)
	apiMux.Get("/chirps/{id}", apicfg.chirpGetIdHandler)
	apiMux.Delete("/chirps/{id}", apicfg.chirpDeleteIdHandler)
	apiMux.Post("/polka/webhooks", apicfg.polkaPostHandler)
	adminMux.Get("/metrics", apicfg.metricsHandler)

	mainMux.Mount("/api", apiMux)
	mainMux.Mount("/admin", adminMux)
	corsMux := middlewareCors(mainMux)
	server := &http.Server{
		Addr: "localhost:" + PORT,
		Handler: corsMux,
	}

	log.Printf("Serving files from %s on port: %s\n", ROOTPATH, PORT)
	log.Fatal(server.ListenAndServe())
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods",
			"GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
