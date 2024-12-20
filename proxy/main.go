package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "studentgit.kata.academy/Zhodaran/go-kata/docs"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth"
	httpSwagger "github.com/swaggo/http-swagger"
	"golang.org/x/crypto/bcrypt"
)

// @title Address API
// @version 1.0
// @description API для поиска
// @host localhost:8080
// @BasePath

// @RequestAddressSearch представляет запрос для поиска
// @Description Этот эндпоинт позволяет получить адрес по наименованию
// @Param address body ResponseAddress true "Географические координаты"

type RequestAddressSearch struct {
	Query string `json:"query"`
}

type ResponseAddresses struct {
	Addresses []*Address `json:"addresses"`
}

type ResponseAddress struct {
	Suggestions []struct {
		Address Address `json:"data"`
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

var (
	tokenAuth *jwtauth.JWTAuth
	users     = make(map[string]string)
)

func init() {
	tokenAuth = jwtauth.New("HS256", []byte("your_secret_key"), nil)
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return
	}
	if _, exists := users[user.Username]; exists {
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	if err != nil {
		http.Error(w, "Could not hash password", http.StatusInternalServerError)
		return
	}
	users[user.Username] = string(hashedPassword)
	w.WriteHeader(http.StatusCreated)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return
	}
	hashedPassword, exists := users[user.Username]
	if !exists || bcrypt.CompareHashAndPassword([]byte(hashedPassword),
		[]byte(user.Password)) != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	claims := map[string]interface{}{
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	}
	_, tokenString, _ := tokenAuth.Encode(claims)

	w.Header().Set("Authorization", "Bearer "+tokenString)
	w.WriteHeader(http.StatusOK)
}

// @Success 200 {object} ResponseAddress
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse

// Логика геокодирования
// @Summary Get Geo Coordinates
// @Description This endpoint allows you to get geo coordinates by address
// @Param address body RequestAddressSearch true "Address search query"
// @Router /api/address/geocode [post]
// @Router /api/address/search [post]
// @Success 200 {object} ResponseAddress "Успешное выполнение"
// @Success 400 {object} ErrorResponse "Ошибка запроса"
// @Success 500 {object} ErrorResponse "Ошибка подключения к серверу"
func GetGeoCoordinates(query string) (ResponseAddresses, error) {
	url := "http://suggestions.dadata.ru/suggestions/api/4_1/rs/suggest/address"
	reqData := map[string]string{"query": query}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return ResponseAddresses{}, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return ResponseAddresses{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token d9e0649452a137b73d941aa4fb4fcac859372c8c")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ResponseAddresses{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResponseAddresses{}, err
	}

	var response ResponseAddress
	err = json.Unmarshal(body, &response)
	if err != nil {
		return ResponseAddresses{}, err
	}

	var addresses ResponseAddresses
	for _, suggestion := range response.Suggestions {
		address := &Address{
			City:   suggestion.Address.City,
			Street: suggestion.Address.Street,
			House:  suggestion.Address.House,
			Lat:    suggestion.Address.Lat,
			Lon:    suggestion.Address.Lon,
		}
		addresses.Addresses = append(addresses.Addresses, address)
	}

	return addresses, nil
}

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	r := chi.NewRouter()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	go func() {

		r.Use(middleware.Logger)
		r.Use(proxyMiddleware)
		r.Get("/swagger/*", httpSwagger.WrapHandler)
		r.Post("/api/register", Register)
		r.Post("/api/login", Login)
		r.Post("/api/address/geocode", func(w http.ResponseWriter, r *http.Request) {
			var req RequestAddressSearch
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			geo, err := GetGeoCoordinates(req.Query) // Здесь можно передать запрос из тела
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			jsonData, err := json.Marshal(geo)
			if err != nil {
				panic(err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(jsonData)
		})

		r.Post("/api/address/search", func(w http.ResponseWriter, r *http.Request) {
			var req RequestAddressSearch
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			geo, err := GetGeoCoordinates(req.Query) // Здесь можно передать запрос из тела
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			jsonData, err := json.Marshal(geo)
			if err != nil {
				panic(err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(jsonData)
		})

		http.ListenAndServe(":8080", r)
	}()
	<-signalChan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Ошибка при завершении работы %v", err)
	} else {
		fmt.Printf("Server stopped gracefully")
	}
}

func proxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, если путь начинается с /api
		if strings.HasPrefix(r.URL.Path, "/api") {
			// Передаем управление следующему обработчику
			next.ServeHTTP(w, r)
			return
		}
		// Перенаправление на hugo
		proxyURL, _ := url.Parse("http://hugo:1313")
		proxy := httputil.NewSingleHostReverseProxy(proxyURL)
		proxy.ServeHTTP(w, r)
	})
}
