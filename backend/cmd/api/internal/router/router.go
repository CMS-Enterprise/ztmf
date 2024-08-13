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

	router.HandleFunc("/fismasystems", controller.ListFismaSystems).Methods("GET")
	router.HandleFunc("/fismasystems/{fismasystemid}", controller.GetFismaSystem).Methods("GET")
	router.HandleFunc("/fismasystems/{fismasystemid}/questions", controller.ListQuestions).Methods("GET")
	// router.HandleFunc("/fismasystems", fismasystemsHandler.CreateRecipe).Methods("POST")
	// router.HandleFunc("/fismasystems/{id}", fismasystemsHandler.UpdateRecipe).Methods("PUT")

	// router.HandleFunc("/functions/{id}/options", controller.GetFunction).Methods("GET")

	// router.HandleFunc("/scores", controller.ListScores).Queries("datacallid", "{datacallid:[0-9]+}", "fismasystemid", "{fismasystemid:[0-9]+}").Methods("GET")
	return router
}
