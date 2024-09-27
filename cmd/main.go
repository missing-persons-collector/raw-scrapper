package main

import (
	"github.com/joho/godotenv"
	"log"
	"missing-persons-scrapper/pkg/croatia"
	"missing-persons-scrapper/pkg/storage"
)

func loadEnv() {
	err := godotenv.Load("../.env")

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	loadEnv()
	storage.Connect()

	if err := croatia.Migrate(); err != nil {
		log.Fatalln(err)
	}

	croatia.Start()
}
