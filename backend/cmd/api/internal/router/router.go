package router

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/gorilla/mux"
)

func Handler() http.Handler {
	router := mux.NewRouter()
	router.Use(auth.Middleware)

	router.HandleFunc("/api/v1/datacalls", controller.ListDataCalls).Methods("GET")

	router.HandleFunc("/api/v1/fismasystems", controller.ListFismaSystems).Methods("GET")
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid}", controller.GetFismaSystem).Methods("GET")
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid}/questions", controller.ListQuestions).Methods("GET")

	router.HandleFunc("/api/v1/functions/{functionid}/options", controller.ListFunctionOptions).Methods("GET")

	router.HandleFunc("/api/v1/users/current", controller.GetCurrentUser).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:[a-zA-Z0-9\\-]+}", controller.GetUserById).Methods("GET")

	router.HandleFunc("/api/v1/scores", controller.ListScores).Queries("datacallid", "{datacallid:[0-9]+}", "fismasystemid", "{fismasystemid:[0-9]+}").Methods("GET")
	router.HandleFunc("/api/v1/scores", controller.SaveScore).Methods("POST")
	router.HandleFunc("/api/v1/scores/{scoreid}", controller.SaveScore).Methods("PUT")

	return router
}