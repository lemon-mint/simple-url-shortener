package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/kataras/hcaptcha"
	"github.com/lemon-mint/envaddr"
	"github.com/lemon-mint/godotenv"
)

var DB *pgxpool.Pool

var (
	siteKey   = os.Getenv("CAPTCHA_SITEKEY")
	secretKey = os.Getenv("CAPTCHA_SECRETKEY")

	client = hcaptcha.New(secretKey)

	newURLForm *template.Template
)

type templateData struct {
	SiteKey string
}

var tdata templateData = templateData{
	SiteKey: siteKey,
}

//go:embed templates/*
var templatesFS embed.FS

func FatalOnError(err error) {
	switch err {
	case nil:
		return
	case pgx.ErrNoRows:
		return
	}

	log.Fatalln(err)
}

func initDatabase() {
	FatalOnError(DB.QueryRow(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS urls (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC')
		);`,
	).Scan())
}

func main() {
	godotenv.Load()
	lnHost := envaddr.Get(":9090")

	DB, err := pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	FatalOnError(err)
	defer DB.Close()
	initDatabase()

	if f, err := os.Stat("templates/index.html"); err == nil && !f.IsDir() {
		newURLForm = template.Must(template.ParseFiles("templates/index.html"))
	} else {
		newURLForm = template.Must(template.ParseFS(templatesFS))
	}

	mux := httprouter.New()

	healthz := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	}
	mux.GET("/healthz", healthz)
	mux.GET("/health", healthz)

	mux.GET("/", index)

	newH := client.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		newURL(rw, r)
	})
	mux.POST("/new", func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		newH(rw, r)
	})

	mux.GET("/u/:id", redirect)

	ln, err := net.Listen("tcp", lnHost)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on", lnHost)
	defer ln.Close()

	err = http.Serve(ln, mux)
	if err != nil {
		log.Fatal(err)
	}
}
