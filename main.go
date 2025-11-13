package main

import (
	"crypto/sha256"
	"crypto/subtle"
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

type User struct {
	Username string
	Password string
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

func auth(user User, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()

		if ok {
			username_actual := sha256.Sum256([]byte(username))
			username_expected := sha256.Sum256([]byte(user.Username))
			password_actual := sha256.Sum256([]byte(password))
			password_expected := sha256.Sum256([]byte(user.Password))

			username_match := subtle.ConstantTimeCompare(username_actual[:], username_expected[:]) == 1
			password_match := subtle.ConstantTimeCompare(password_actual[:], password_expected[:]) == 1

			if username_match && password_match {
				next.ServeHTTP(writer, request)

				return
			}
		}

		writer.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
	})
}

func env() User {
	file, err := os.ReadFile("env.json")
	if err != nil {
		panic(err)
	}

	var user User
	if err := json.Unmarshal(file, &user); err != nil {
		panic(err)
	}

	return user
}

func main() {
	user := env()
	list := read("grocery.json")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", auth(user, func(writer http.ResponseWriter, request *http.Request) {
		payload, err := json.Marshal(list)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Write(payload)
	}))

	mux.HandleFunc("POST /", auth(user, func(writer http.ResponseWriter, request *http.Request) {
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
	}))

	fmt.Printf("Starting server on port: %d\n", PORT)
	err := http.ListenAndServe(fmt.Sprintf(":%d", PORT), mux)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Server closed\n")
	} else if err != nil {
		fmt.Printf("Error starting server %s\n", err)
		os.Exit(1)
	}
}
