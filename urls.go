package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/julienschmidt/httprouter"
)

var idlen int = 4

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
		var retryCount int
	retry:
		id = NewID(idlen)
		// If primary key collision, retry
		_, err := t.Exec(context.Background(),
			`INSERT INTO urls (id, url, needs_captcha, needs_password) VALUES ($1, $2, false, false)`,
			id, url)
		if err != nil {
			log.Printf("Error inserting into urls: %v", err)
			if strings.Contains(err.Error(), "duplicate key value") {
				if retryCount > 2 {
					idlen++ // doesn't matter atomicity, just to make it more likely to succeed
				}
				retryCount++
				goto retry
			}
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("Error inserting into urls: %v", err)
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
