package main

import (
	"github.com/joho/godotenv"
	"log"
	"missing-persons-scrapper/pkg/countries/croatia"
	"missing-persons-scrapper/pkg/countries/romania"
	"missing-persons-scrapper/pkg/storage"
	"sync"
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

	if err := romania.Migrate(); err != nil {
		log.Fatalln(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		croatia.Start()
	}()

	go func() {
		romania.Start()
	}()

	wg.Wait()
}
