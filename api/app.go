package api

import (
	"context"
	"encoding/json"
	"github.com/Bnei-Baruch/wf-trim/common"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"net/http"
)

type App struct {
	Router        *mux.Router
	tokenVerifier *oidc.IDTokenVerifier
}

func (a *App) InitAuthClient() {
	oidcProvider, err := oidc.NewProvider(context.TODO(), common.ACC_URL)
	if err != nil {
		log.Fatal().Str("source", "APP").Err(err).Msg("oidc.NewProvider")
	}
	a.tokenVerifier = oidcProvider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})
}

func (a *App) InitServer() {
	InitLog()
	log.Info().Str("source", "APP").Msg("initializing app")
	a.Router = mux.NewRouter()
	a.InitializeRoutes()
}

func (a *App) Run(port string) {
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Content-Length", "Accept-Encoding", "Content-Range", "Content-Disposition", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "DELETE", "POST", "PUT", "OPTIONS"})

	if port == "" {
		port = ":8080"
	}

	log.Info().Str("source", "APP").Msgf("app run %s", port)

	if err := http.ListenAndServe(port, handlers.CORS(originsOk, headersOk, methodsOk)(a.Router)); err != nil {
		log.Fatal().Str("source", "APP").Err(err).Msg("http.ListenAndServe")
	}
}

func (a *App) InitializeRoutes() {
	a.Router.Use(a.LoggingMiddleware)
	a.Router.HandleFunc("/convert", a.convertExec).Methods("GET")
	a.Router.HandleFunc("/{ep}/upload", a.handleUpload).Methods("POST")
	a.Router.HandleFunc("/workflow/{ep}", a.putJson).Methods("PUT")
	a.Router.HandleFunc("/{ep}/status", a.statusJson).Methods("GET")
	a.Router.PathPrefix("/data/").Handler(http.StripPrefix("/data/", http.FileServer(http.Dir("/data"))))
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}
