package main

import (
	"log"

	"github.com/lemon-mint/envaddr"
	"github.com/lemon-mint/godotenv"
)

func main() {
	godotenv.Load()
	lnHost := envaddr.Get(":9090")
	log.Println("Listening on", lnHost)
}
