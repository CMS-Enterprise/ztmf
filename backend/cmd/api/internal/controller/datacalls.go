package controller

import (
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
)

func ListDataCalls(w http.ResponseWriter, r *http.Request) {
	datacalls, err := model.FindDataCalls(r.Context())
	respond(w, r, datacalls, err)
}
