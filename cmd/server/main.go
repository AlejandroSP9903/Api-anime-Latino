
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	apianime "github.com/dst3v3n/api-anime"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	service := apianime.NewAnimeFlv()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "chimbaanime-api",
		})
	})

	mux.HandleFunc("GET /anime/search", func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Falta el parámetro q"})
			return
		}

		page := uint(1)
		if raw := r.URL.Query().Get("page"); raw != "" {
			n, err := strconv.ParseUint(raw, 10, 32)
			if err != nil || n == 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Página inválida"})
				return
			}
			page = uint(n)
		}

		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		result, err := service.SearchAnime(ctx, query, page)
		if err != nil {
			log.Printf("SearchAnime: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Falló la búsqueda en AnimeFLV"})
			return
		}

		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("GET /anime/info", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Falta el parámetro id"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		result, err := service.AnimeInfo(ctx, id)
		if err != nil {
			log.Printf("AnimeInfo: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "No se pudo obtener el anime"})
			return
		}

		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("GET /anime/links", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		rawEpisode := r.URL.Query().Get("episode")

		episode, err := strconv.ParseUint(rawEpisode, 10, 32)
		if id == "" || err != nil || episode == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Se requieren id y episode válido",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		result, err := service.Links(ctx, id, uint(episode))
		if err != nil {
			log.Printf("Links: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "No se pudieron obtener los enlaces"})
			return
		}

		writeJSON(w, http.StatusOK, result)
	})

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           cors(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("ChimbaAnime API escuchando en el puerto %s", port)
	log.Fatal(server.ListenAndServe())
}

