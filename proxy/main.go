package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	_ "proxy/docs" // Импортируйте сгенерированные документы

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Swagger Example API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host petstore.swagger.io
// @BasePath
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

// ErrorResponse представляет ответ с ошибкой
// @Description Ошибка, возникающая при обработке запроса
// @Success 400 {object} ErrorResponse
type ErrorResponse struct {
	BadRequest      string `json:"400"`
	DadataBad       string `json:"500"`
	SuccefulRequest string `json:"200"`
}

// @Summary Get Geo Coordinates
// @Description This endpoint allows you to get geo coordinates by address
// @Param address body RequestAddressSearch true "Address search query"
// @Router /api/address/geocode [post]
// @Router /api/address/search [post]
// @Success 200 {object} ResponseAddress "Успешное выполнение"
// @Success 400 {object} ErrorResponse "Ошибка запроса"
// @Success 500 {object} ErrorResponse "Ошибка подключения к серверу"
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
	r.Get("/swagger/*", httpSwagger.WrapHandler)
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
