package main

import (
	"log"
	"github.com/joho/godotenv"

	"github.com/SantiagoSu15/Task-it-Chat-MS/internal/bootstrap"
)

func main() {
	_ = godotenv.Load()
	if err := bootstrap.NewApp().Run(); err != nil {
		log.Fatal(err)
	}
}
