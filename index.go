package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	newURLForm.Execute(w, tdata)
}
