package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v4"
	"github.com/julienschmidt/httprouter"
)

func newURL(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url := r.Form.Get("url")
	if url == "" || len(url) > 512 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var id string
	err = DB.BeginFunc(context.Background(), func(t pgx.Tx) error {
	retry:
		id = NewID()
		_, err := t.Exec(context.Background(),
			`INSERT INTO urls (id, url) VALUES ($1, $2)`,
			id, url)
		if err != nil {
			if err == pgx.ErrNoRows {
				goto retry
			}
			return err
		}
		return nil
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Redirect to the result page
	http.Redirect(w, r, fmt.Sprintf("/result/%s", id), http.StatusSeeOther)
}

func redirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	var url string
	err := DB.QueryRow(context.Background(),
		`SELECT url FROM urls WHERE id = $1`,
		id).Scan(&url)
	if err != nil {
		if err == pgx.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func result(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	var url string
	err := DB.QueryRow(context.Background(),
		`SELECT url FROM urls WHERE id = $1`,
		id).Scan(&url)
	if err != nil {
		if err == pgx.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = templates.ExecuteTemplate(w, "result.html", fmt.Sprintf("https://%s/u/%s", r.Host, id))
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}
