package request

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Param(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}
