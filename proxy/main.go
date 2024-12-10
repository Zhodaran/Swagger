package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
)

// @title Address API
// @version 1.0
// @description API для поиска
// @host localhost:8080
// @BasePath /api

// @RequestAddressSearch представляет запрос для поиска
// @Description Этот эндпоинт позволяет получить адрес по наименованию
// @Param address body ResponseAddress true "Географические координаты"

type RequestAddressSearch struct {
	Query string `json:"query"`
}

type ResponseAddresses struct {
	Lat string `json:"geo_lat"`
	Lng string `json:"geo_lon"`
}

type ResponseAddress struct {
	Suggestions []struct {
		Address ResponseAddresses `json:"data"`
	} `json:"suggestions"`
}

// @Success 200 {object} ResponseAddress
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// Логика геокодирования

func getGeoCoordinates(query string) (string, error) {
	url := "http://suggestions.dadata.ru/suggestions/api/4_1/rs/suggest/address"
	reqData := map[string]string{"query": query}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token d9e0649452a137b73d941aa4fb4fcac859372c8c")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Println(string(body))
	var response ResponseAddress
	err = json.Unmarshal(body, &response)
	if err != nil {
		panic(err)
	}

	if len(response.Suggestions) > 0 {
		return fmt.Sprintf("%s %s", response.Suggestions[0].Address.Lat, response.Suggestions[0].Address.Lng), nil
	}

	return "", fmt.Errorf("no suggestions found")
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/api/address/geocode", func(w http.ResponseWriter, r *http.Request) {
		geo, err := getGeoCoordinates("москва сухонская 11") // Здесь можно передать запрос из тела
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(geo))
	})

	r.Post("/api/address/search", func(w http.ResponseWriter, r *http.Request) {
		geo, err := getGeoCoordinates("москва сухонская 11") // Здесь можно передать запрос из тела
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		proxyMiddleware(geo)(w, r)
	})

	http.ListenAndServe(":8080", r)
}

func proxyMiddleware(geo string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(geo))
	})
}