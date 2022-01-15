package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var indexHTML []byte

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(indexHTML)
	if err != nil {
		log.Printf("error writing index: %v, connection: %v, xff: %v", err, r.RemoteAddr, r.Header.Get("X-Forwarded-For"))
	}
}
