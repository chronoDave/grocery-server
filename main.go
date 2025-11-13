package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

type Item struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}

func read(path string) []Item {
	file, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		} else {
			return []Item{}
		}
	}

	var list []Item
	if err := json.Unmarshal(file, &list); err != nil {
		panic(err)
	}

	return list
}

func write(path string, list []Item) {
	data, err := json.Marshal(list)
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(path, data, 0666); err != nil {
		panic(err)
	}
}

const PORT = 3333

func main() {
	list := read("grocery.json")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(writer http.ResponseWriter, request *http.Request) {
		payload, err := json.Marshal(list)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Write(payload)
	})

	mux.HandleFunc("POST /", func(writer http.ResponseWriter, request *http.Request) {
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&list)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)

			return
		}

		write("grocery.json", list)

		payload, err := json.Marshal(list)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Write(payload)
	})

	fmt.Printf("Starting server on port: %d\n", PORT)
	err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), mux)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Server closed\n")
	} else if err != nil {
		fmt.Printf("Error starting server %s\n", err)
		os.Exit(1)
	}
}
