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
	router.HandleFunc("/api/v1/datacalls", controller.SaveDataCall).Methods("POST")
	router.HandleFunc("/api/v1/datacalls/{datacallid:[0-9]+}", controller.GetDataCallByID).Methods("GET")
	router.HandleFunc("/api/v1/datacalls/{datacallid:[0-9]+}", controller.SaveDataCall).Methods("PUT")
	// records that a fisma system has completed the data call
	router.HandleFunc("/api/v1/datacalls/{datacallid:[0-9]+}/fismasystems/{fismasystemid:[0-9]+}", controller.SaveDataCallFismaSystem).Methods("PUT")
	// returns a list of fisma systems that have marked this data call as complete
	router.HandleFunc("/api/v1/datacalls/{datacallid:[0-9]+}/fismasystems", controller.ListDataCallFismaSystems).Methods("GET")

	router.HandleFunc("/api/v1/datacalls/{datacallid:[0-9]+}/export", controller.GetDatacallExport).Methods("GET")

	router.HandleFunc("/api/v1/fismasystems", controller.ListFismaSystems).Methods("GET")
	router.HandleFunc("/api/v1/fismasystems", controller.SaveFismaSystem).Methods("POST")
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}", controller.GetFismaSystem).Methods("GET")
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}", controller.SaveFismaSystem).Methods("PUT")
	// returns a list of data calls that this fisma system has marked complete
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}/datacalls", controller.ListFismaSystemDataCalls).Methods("GET")

	// TODO: deprecate this in favor of non-nested URIs
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}/questions", controller.ListFismaSystemQuestions).Methods("GET")

	router.HandleFunc("/api/v1/functions/{functionid:[0-9]+}/options", controller.ListFunctionOptions).Methods("GET")

	router.HandleFunc("/api/v1/users", controller.ListUsers).Methods("GET")
	router.HandleFunc("/api/v1/users", controller.SaveUser).Methods("POST")
	router.HandleFunc("/api/v1/users/current", controller.GetCurrentUser).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}", controller.GetUserByID).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}", controller.SaveUser).Methods("PUT")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}", controller.DeleteUser).Methods("DELETE")

	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}/assignedfismasystems", controller.ListUserFismaSystems).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}/assignedfismasystems", controller.CreateUserFismaSystem).Methods("POST")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}/assignedfismasystems/{fismasystemid:[0-9]+}", controller.DeleteUserFismaSystem).Methods("DELETE")

	router.HandleFunc("/api/v1/scores", controller.ListScores).Methods("GET")
	router.HandleFunc("/api/v1/scores/aggregate", controller.GetScoresAggregate).Methods("GET") // yes "aggregate" is a noun
	router.HandleFunc("/api/v1/scores", controller.SaveScore).Methods("POST")
	router.HandleFunc("/api/v1/scores/{scoreid:[0-9]+}", controller.SaveScore).Methods("PUT")

	router.HandleFunc("/api/v1/questions", controller.ListQuestions).Methods("GET")
	router.HandleFunc("/api/v1/questions/{questionid:[0-9]+}", controller.GetQuestionByID).Methods("GET")
	router.HandleFunc("/api/v1/questions", controller.SaveQuestion).Methods("POST")
	router.HandleFunc("/api/v1/questions/{questionid:[0-9]+}", controller.SaveQuestion).Methods("PUT")

	router.HandleFunc("/api/v1/functions", controller.ListFunctions).Methods("GET")
	router.HandleFunc("/api/v1/functions/{functionid:[0-9]+}", controller.GetFunctionByID).Methods("GET")
	router.HandleFunc("/api/v1/functions", controller.SaveFunction).Methods("POST")
	router.HandleFunc("/api/v1/functions/{functionid:[0-9]+}", controller.SaveFunction).Methods("PUT")

	router.HandleFunc("/api/v1/events", controller.GetEvents).Methods("GET")

	// massemails resource only supports a single verb as there are no records to get list and details for, but the operation is non-idempotent
	router.HandleFunc("/api/v1/massemails", controller.SaveMassEmail).Methods("POST")

	return router
}
