package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

func main() {

	port := os.Getenv("PORT")

	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/", func(
		w http.ResponseWriter,
		r *http.Request,
	) {

		enableCORS(w)

		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "CLIProxyAPI",
			"runtime": "vercel",
			"time":    time.Now().Unix(),
		})
	})

	http.HandleFunc(
		"/v1/chat/completions",
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			enableCORS(w)

			json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-vercel",
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Hello from Vercel",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	http.ListenAndServe(":"+port, nil)
}

func enableCORS(
	w http.ResponseWriter,
) {

	w.Header().Set(
		"Access-Control-Allow-Origin",
		"*",
	)

	w.Header().Set(
		"Access-Control-Allow-Headers",
		"*",
	)

	w.Header().Set(
		"Access-Control-Allow-Methods",
		"GET,POST,OPTIONS",
	)

	w.Header().Set(
		"Content-Type",
		"application/json",
	)
}
