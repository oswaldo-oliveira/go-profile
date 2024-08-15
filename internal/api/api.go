package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ID uuid.UUID

func (id ID) String() string {
	return uuid.UUID(id).String()
}

type application struct {
	data map[ID]User
}

func NewApplication(data map[ID]User) *application {
	return &application{data}
}

func (a *application) Insert(user User) (User, string) {
	id := ID(uuid.New())
	_, ok := a.data[id]
	for ok {
		id = ID(uuid.New())
		_, ok = a.data[id]
	}
	a.data[id] = user

	return a.data[id], id.String()
}

func (a *application) FindAll() map[ID]User {
	return a.data
}

func (a *application) FindByID(id ID) *User {
	user, ok := a.data[id]
	if !ok {
		return nil
	}
	return &user
}

func (a *application) Update(id ID, user User) {
	if _, ok := a.data[id]; !ok {
		return
	}
	a.data[id] = user
}

func (a *application) Delete(id ID) {
	if _, ok := a.data[id]; !ok {
		return
	}

	delete(a.data, id)
}

type User struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Biography string `json:"biography"`
}

type UserResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Biography string `json:"biography"`
}

type Response struct {
	Error string `json:"error,omitempty"`
	Data  any    `json:"data,omitempty"`
}

func NewHandler(db *application) http.Handler {
	r := chi.NewMux()
	r.Use(middleware.Recoverer, middleware.RequestID, middleware.Logger)

	r.Route("/api", func(r chi.Router) {
		r.Post("/users", CreateUserHandler(db))
		r.Get("/users", FindAllUsersHandler(db))
		r.Get("/users/{id}", FindByIDUserHandler(db))
		r.Put("/users/{id}", UpdateUserHandler(db))
		r.Delete("/users/{id}", DeleteUserHandler(db))
	})
	return r
}

func CreateUserHandler(db *application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body User
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			sendJSON(
				w,
				Response{
					Error: "Please provide FirstName LastName and bio for the user",
				},
				http.StatusBadRequest,
			)
			return
		}

		user, ID := db.Insert(body)

		sendJSON(
			w,
			Response{Data: UserResponse{
				ID:        ID,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Biography: user.Biography,
			}},
			http.StatusCreated,
		)
	}
}
func FindAllUsersHandler(db *application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var response []UserResponse
		users := db.FindAll()
		for id, user := range users {
			response = append(response, UserResponse{
				ID:        id.String(),
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Biography: user.Biography,
			})
		}
		if response == nil {
			sendJSON(w, Response{Data: []UserResponse{}}, http.StatusOK)
			return
		}

		sendJSON(w, Response{Data: response}, http.StatusOK)
	}
}
func FindByIDUserHandler(db *application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawID := chi.URLParam(r, "id")

		id, err := uuid.Parse(rawID)
		if err != nil {
			http.Error(w, "UUID not valid", http.StatusBadRequest)
			return
		}

		user := db.FindByID(ID(id))
		if user == nil {
			sendJSON(w, Response{Error: "The user with the specified ID does not exist."}, http.StatusNotFound)
			return
		}

		sendJSON(w, Response{Data: UserResponse{ID: rawID, FirstName: user.FirstName, LastName: user.LastName, Biography: user.Biography}}, http.StatusOK)
	}
}
func UpdateUserHandler(db *application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body User
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			sendJSON(w, Response{Error: "Please provide FirstName LastName and bio for the user"}, http.StatusBadRequest)
			return
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			http.Error(w, "UUID not valid", http.StatusBadRequest)
			return
		}

		if user := db.FindByID(ID(id)); user == nil {
			sendJSON(w, Response{Error: "The user with the specified ID does not exist."}, http.StatusNotFound)
			return
		}

		db.Update(ID(id), body)

		sendJSON(w, Response{}, http.StatusNoContent)
	}
}
func DeleteUserHandler(db *application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			http.Error(w, "UUID not valid", http.StatusBadRequest)
			return
		}

		if user := db.FindByID(ID(id)); user == nil {
			sendJSON(w, Response{Error: "The user with the specified ID does not exist."}, http.StatusNotFound)
			return
		}

		db.Delete(ID(id))

		sendJSON(w, Response{}, http.StatusNoContent)
	}
}

func sendJSON(w http.ResponseWriter, resp Response, status int) {
	w.Header().Set("Content-Type", "application/json")

	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal JSON data.", "error", err)
		sendJSON(w, Response{Error: "something went wrong"}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		slog.Error("failed to write response to client.", "error", err)
		return
	}
}
