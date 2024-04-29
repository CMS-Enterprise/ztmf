package main

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/engine"
	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
)

func main() {

	cfg := config.GetConfig()
	http.Handle("/graphql", CorsMiddleware(engine.GetHttpHandler()))

	log.Printf("%s environment listening on %s\n", cfg.ENV, cfg.PORT)
	switch cfg.ENV {
	case "local":
		log.Fatal(http.ListenAndServe(":"+cfg.PORT, nil))
	case "dev", "prod":
		log.Fatal(http.ListenAndServeTLS(":"+cfg.PORT, cfg.CERT_FILE, cfg.KEY_FILE, nil))
	}

}

func CorsMiddleware(next http.Handler) http.Handler {
	cfg := config.GetConfig()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// allow cross domain AJAX requests
		w.Header().Set("Access-Control-Allow-Origin", cfg.GetCorsOrigin())
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		next.ServeHTTP(w, r)
	})
}
