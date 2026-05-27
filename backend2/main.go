package main

import (
	"backend-app/config"
	"backend-app/internal/handlers"
	"fmt"
	"net/http"
)

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	http.HandleFunc("/", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "JustSent backend running")
	}))

	http.HandleFunc("/health", enableCORS(handlers.HandleHealth))
	http.HandleFunc("/v1/shares", enableCORS(handlers.HandleShares))
	http.HandleFunc("/v1/transfers", enableCORS(handlers.HandleTransfers))
	http.HandleFunc("/share/", enableCORS(handlers.HandleDownload))

	// START SERVER IN BACKGROUND
	go func() {
		fmt.Printf("Server running on :%s\n", config.ServerPort)

		err := http.ListenAndServe(":"+config.ServerPort, nil)
		if err != nil {
			panic(err)
		}
	}()
	select {}
}
