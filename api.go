package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/joho/godotenv/autoload"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewApiServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))
	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleGetAccountByID), s.store))

	log.Println("JSON API server running on port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed %s", r.Method)
	}

	req := &LoginRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}

	acc, err := s.store.GetAccountByEmail(string(req.Email))
	if err != nil {
		WriteJSON(w, http.StatusNotFound, "Email not found")
		return err
	}

	if !acc.ValidPassword(req.Password) {
		return fmt.Errorf("Failed Login")
	}

	token, err := createJWT(acc)
	if err != nil {
		return err
	}

	resp := &LoginResponse{
		AccountID: acc.ID,
		Token:     token,
	}

	return WriteJSON(w, http.StatusOK, resp)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	switch meth := r.Method; meth {
	case "GET":
		return s.handleGetAccount(w, r)
	case "POST":
		return s.handleCreateAccount(w, r)
	default:
		defer r.Body.Close()
		return fmt.Errorf("method not allow %s", meth)
	}
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	defer r.Body.Close()
	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		id, err := getID(r)
		if err != nil {
			return err
		}

		account, err := s.store.GetAccountByID(id)
		if err != nil {
			return err
		}

		return WriteJSON(w, http.StatusOK, account)
	}

	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	defer r.Body.Close()
	return fmt.Errorf("method not allowed %s", r.Method)
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	req := &CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}

	account, err := NewAccount(req.FirstName, req.LastName, req.Email, req.Password)
	if err != nil {
		return nil
	}

	if _, err := s.store.CreateAccount(account); err != nil {
		return err
	}

	defer r.Body.Close()
	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}

	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}

	defer r.Body.Close()
	return WriteJSON(w, http.StatusOK, map[string]uuid.UUID{"deleted": id})
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

func permissionDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, ApiError{Error: "permission denied"})
}

// decorator
// store jwt in the database when creating and check against that when the user is fetching data
func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("x-jwt-token")
		token, err := validateJWT(tokenString)
		if err != nil {
			permissionDenied(w)
			return
		}
		if !token.Valid {
			permissionDenied(w)
			return
		}

		userID, err := getID(r)
		if err != nil {
			permissionDenied(w)
			return
		}

		account, err := s.GetAccountByID(userID)
		if err != nil {
			permissionDenied(w)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		parsedUUID, err := uuid.Parse(claims["accountID"].(string))
		if err != nil {
			return
		}

		if account.ID != parsedUUID {
			permissionDenied(w)
			return
		}

		handlerFunc(w, r)
	}
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt": 15000,
		"accountID": account.ID,
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}

// function signature of our handler.
type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Error string `json:"error"`
}

// decorating our apiFunc into a http.HandlerFunc.
// handling our errors externally.
func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func getID(r *http.Request) (uuid.UUID, error) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		return id, fmt.Errorf("invalid id given %s", idStr)
	}

	defer r.Body.Close()
	return id, nil
}
