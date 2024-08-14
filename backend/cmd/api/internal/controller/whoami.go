package controller

import "net/http"

func WhoAmI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/jwt")
	w.Write([]byte(r.Header[http.CanonicalHeaderKey("authorization")][0]))
}
