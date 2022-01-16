package main

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"io/fs"
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
	secretKey string

	client *hcaptcha.Client

	templates *template.Template
)

var tdata string

//go:embed templates/*
var templatesFS embed.FS

//go:embed public/*
var staticFS embed.FS

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
	/* v1
	FatalOnError(DB.QueryRow(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS urls (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC')
		);`,
	).Scan())
	*/

	// v2
	FatalOnError(DB.QueryRow(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS urls (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			needs_captcha BOOLEAN NOT NULL DEFAULT FALSE,
			needs_password BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC')
		);`,
	).Scan())

	FatalOnError(DB.QueryRow(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS passwords (
			id TEXT PRIMARY KEY,
			salt TEXT NOT NULL,
			password TEXT NOT NULL,
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'UTC')
		);`,
	).Scan())

	// ALTER TABLES v1 => v2
	FatalOnError(DB.QueryRow(
		context.Background(),
		`ALTER TABLE urls ADD COLUMN IF NOT EXISTS needs_captcha BOOLEAN NOT NULL DEFAULT FALSE;`,
	).Scan())

	FatalOnError(DB.QueryRow(
		context.Background(),
		`ALTER TABLE urls ADD COLUMN IF NOT EXISTS needs_password BOOLEAN NOT NULL DEFAULT FALSE;`,
	).Scan())
}

func main() {
	godotenv.Load()
	lnHost := envaddr.Get(":9090")

	secretKey = os.Getenv("CAPTCHA_SECRETKEY")
	client = hcaptcha.New(secretKey)
	tdata = os.Getenv("CAPTCHA_SITEKEY")

	var err error
	DB, err = pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	FatalOnError(err)
	defer DB.Close()
	initDatabase()

	templates = template.Must(template.ParseFS(templatesFS, "templates/*"))

	var staticHTTPFS http.FileSystem
	if f, err := os.Stat("public"); err == nil && f.IsDir() {
		staticHTTPFS = http.Dir("public")
	} else {
		f, err := fs.Sub(staticFS, "public")
		FatalOnError(err)
		staticHTTPFS = http.FS(f)
	}

	mux := httprouter.New()

	healthz := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	}
	mux.GET("/healthz", healthz)
	mux.GET("/health", healthz)

	mux.ServeFiles("/public/*filepath", staticHTTPFS)

	mux.GET("/", index)

	newH := client.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		newURL(rw, r)
	})
	mux.POST("/new", func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		newH(rw, r)
	})

	mux.GET("/u/:id", redirect)
	mux.GET("/result/:id", result)

	var templateBuffer bytes.Buffer
	err = templates.ExecuteTemplate(&templateBuffer, "index.html", tdata)
	FatalOnError(err)
	indexHTML = templateBuffer.Bytes()

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
