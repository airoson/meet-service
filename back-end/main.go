package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"meet-service/controllers"
	"meet-service/sfu"
	"meet-service/utils"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Can't load .env file")
	}
}

func main() {
	connSrt := fmt.Sprintf("dbname=%s user=%s password=%s host=%s sslmode=disable", os.Getenv("POSTGRES_DB"), os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"))
	db, err := sql.Open("postgres", connSrt)
	if err != nil {
		log.Fatal("Can't connect do database: ", err)
	}
	utils.ClearUnusedDataStartup(db)
	roomManager := sfu.GetInternalRoomManager()

	r := mux.NewRouter()
	r.Methods("GET").PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./resources/static"))))
	r.Methods("GET").Path("/").Handler(&controllers.HTMLHandler{PathToFile: "./resources/static/index.html"})
	r.Methods("GET").Path("/login").HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, "index.html", http.StatusPermanentRedirect)
	})

	restApi := controllers.RestApi{
		Manager: roomManager,
		Db:      db,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	apiSubrouter := r.PathPrefix("/api/").Subrouter()
	apiSubrouter.Use(controllers.GetContentTypeMiddleware("application/json"))
	apiSubrouter.Use(controllers.HandlePanic)

	roomSubrouter := apiSubrouter.Methods("POST", "DELETE").Subrouter()
	roomSubrouter.Use(controllers.GetJWTValidation("USER", "ADMIN"))
	roomSubrouter.Path("/room").Methods("POST").HandlerFunc(restApi.HandleRoomPostRequest)
	roomSubrouter.Path("/room").Methods("DELETE").HandlerFunc(restApi.HandleRoomDeleteRequest)
	roomSubrouter = apiSubrouter.Methods("GET").Subrouter()
	roomSubrouter.Path("/room").HandlerFunc(restApi.HandleRoomGetRequest)
	roomSubrouter = apiSubrouter.Path("/room/ws").Subrouter()
	roomSubrouter.Use(controllers.GetJWTValidation("USER", "ADMIN", "UNAUTHORIZED"))
	roomSubrouter.Methods("GET").HandlerFunc(restApi.HandleRoomConnectionRequest)

	authSubrouter := apiSubrouter.PathPrefix("/auth/").Subrouter()
	authSubrouter.Methods("POST").Path("/signup").HandlerFunc(restApi.HandleSignupRequest)
	authSubrouter.Methods("POST").Path("/login").HandlerFunc(restApi.HandleLoginRequest)
	authSubrouter.Methods("POST").Path("/logout").HandlerFunc(restApi.HandleLogoutRequest)
	authSubrouter.Methods("POST").Path("/refresh-token").HandlerFunc(restApi.HandleRefreshTokenRequest)

	userSubrouter := apiSubrouter.PathPrefix("/user").Subrouter()
	userSubrouter.Use(controllers.GetJWTValidation("ADMIN", "USER"))
	userSubrouter.Methods("GET").HandlerFunc(restApi.HandleUserGetRequest)
	userSubrouter.Methods("PATCH").HandlerFunc(restApi.HandleUserPatchRequest)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		// Logger:           log.Default(),
	})
	err = http.ListenAndServe("localhost:8080", c.Handler(r))
	if err != nil {
		fmt.Println("Error: ", err)
	}
}
