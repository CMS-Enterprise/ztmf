package router

import (
	"net/http"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/auth"
	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/controller"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

func Handler() http.Handler {
	var userIdPattern = "[a-zA-Z0-9]+-[a-zA-Z0-9]+-[a-zA-Z0-9]+-[a-zA-Z0-9]+-[a-zA-Z0-9]+"
	root := mux.NewRouter()

	// The pre-auth IdP lookup is the one unauthenticated route: the caller has
	// no session yet, so it cannot sit behind auth.Middleware. It is registered
	// before the authenticated subrouter so mux matches it first, and it is
	// wrapped in an in-process rate limiter as defense-in-depth (WAF on this
	// path is the authoritative limit).
	lookupLimiter := auth.NewRateLimiter(rate.Limit(5), 10, 10*time.Minute)
	root.Handle("/api/v1/auth/lookup", lookupLimiter.Middleware(http.HandlerFunc(controller.LookupIdP))).Methods("GET")

	// Post-OIDC login. The ALB authenticates /login* per IdP and forwards here
	// with the IdP token in the auth header; SessionHandler mints the app
	// session cookie. These live outside auth.Middleware because no app session
	// cookie exists yet - the ALB OIDC handshake is the gate for these paths.
	root.HandleFunc("/login", auth.SessionHandler).Methods("GET")
	root.PathPrefix("/login/").HandlerFunc(auth.SessionHandler).Methods("GET")

	// Every other route requires authentication. Registering them on a subrouter
	// keeps auth.Middleware off the public lookup route above.
	router := root.PathPrefix("/").Subrouter()
	router.Use(auth.Middleware)

	router.HandleFunc("/api/v1/datacalls", controller.ListDataCalls).Methods("GET")
	router.HandleFunc("/api/v1/datacalls", controller.SaveDataCall).Methods("POST")
	router.HandleFunc("/api/v1/datacalls/latest", controller.GetLatestDataCall).Methods("GET")
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
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}", controller.DeleteFismaSystem).Methods("DELETE")
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}/reactivate", controller.ReactivateFismaSystem).Methods("PUT")
	// returns a list of data calls that this fisma system has marked complete
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}/datacalls", controller.ListFismaSystemDataCalls).Methods("GET")

	// TODO: deprecate this in favor of non-nested URIs
	router.HandleFunc("/api/v1/fismasystems/{fismasystemid:[0-9]+}/questions", controller.ListFismaSystemQuestions).Methods("GET")

	router.HandleFunc("/api/v1/functions/{functionid:[0-9]+}/options", controller.ListFunctionOptions).Methods("GET")

	router.HandleFunc("/api/v1/opdivs", controller.ListOpDivs).Methods("GET")

	router.HandleFunc("/api/v1/users", controller.ListUsers).Methods("GET")
	router.HandleFunc("/api/v1/users", controller.SaveUser).Methods("POST")
	router.HandleFunc("/api/v1/users/current", controller.GetCurrentUser).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}", controller.GetUserByID).Methods("GET")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}", controller.SaveUser).Methods("PUT")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}", controller.DeleteUser).Methods("DELETE")
	router.HandleFunc("/api/v1/users/{userid:"+userIdPattern+"}/restore", controller.RestoreUser).Methods("PUT")

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

	router.HandleFunc("/api/v1/systemenrichment/{fisma_uuid:[a-zA-Z0-9-]+}", controller.GetSystemEnrichment).Methods("GET")

	// massemails resource only supports a single verb as there are no records to get list and details for, but the operation is non-idempotent
	router.HandleFunc("/api/v1/massemails", controller.SaveMassEmail).Methods("POST")

	return root
}
