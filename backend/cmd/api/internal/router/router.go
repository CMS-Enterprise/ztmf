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

	router.HandleFunc("/functions/{functionid}/options", controller.ListFunctionOptions).Methods("GET")

	router.HandleFunc("/users/{email:[a-zA-Z0-9.]+@cms.hhs.gov}", controller.GetUserByEmail).Methods("GET")
	router.HandleFunc("/users/{userid:[a-zA-Z0-9\\-]+}", controller.GetUserById).Methods("GET")

	router.HandleFunc("/scores", controller.ListScores).Queries("datacallid", "{datacallid:[0-9]+}", "fismasystemid", "{fismasystemid:[0-9]+}").Methods("GET")
	router.HandleFunc("/scores", controller.SaveScore).Methods("POST")
	router.HandleFunc("/scores/{scoreid}", controller.SaveScore).Methods("PUT")

	router.HandleFunc("/whoami", controller.WhoAmI).Methods("GET")
	return router
}
