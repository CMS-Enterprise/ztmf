package router

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/gorilla/mux"
)

func Handler() http.Handler {
	var userIdPattern = "[a-zA-Z0-9]+-[a-zA-Z0-9]+-[a-zA-Z0-9]+-[a-zA-Z0-9]+-[a-zA-Z0-9]+"
	router := mux.NewRouter()
	router.Use(auth.Middleware)

	router.HandleFunc("/api/v1/datacalls", controller.ListDataCalls).Methods("GET")
	router.HandleFunc("/api/v1/datacalls/{datacallid:[0-9]+}/export", controller.GetDatacallExport).Methods("GET")

	router.HandleFunc("/api/v1/fismasystems", controller.ListFismaSystems).Methods("GET")
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}", controller.GetFismaSystem).Methods("GET")
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}/questions", controller.ListQuestions).Methods("GET")

	router.HandleFunc("/api/v1/functions/{functionid:[0-9]+}/options", controller.ListFunctionOptions).Methods("GET")

	router.HandleFunc("/api/v1/users", controller.ListUsers).Methods("GET")
	router.HandleFunc("/api/v1/users", controller.SaveUser).Methods("POST")
	router.HandleFunc("/api/v1/users/current", controller.GetCurrentUser).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}", controller.GetUserById).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}", controller.SaveUser).Methods("PUT")

	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}/assignedfismasystems", controller.ListUserFismaSystems).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}/assignedfismasystems", controller.CreateUserFismaSystem).Methods("POST")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}/assignedfismasystems/{fismasystemid:[0-9]+}", controller.DeleteUserFismaSystem).Methods("DELETE")

	router.HandleFunc("/api/v1/scores", controller.ListScores).Methods("GET")
	router.HandleFunc("/api/v1/scores/aggregate", controller.GetScoresAggregate).Methods("GET") // yes "aggregate" is a noun
	router.HandleFunc("/api/v1/scores", controller.SaveScore).Methods("POST")
	router.HandleFunc("/api/v1/scores/{scoreid:[0-9]+}", controller.SaveScore).Methods("PUT")

	return router
}
