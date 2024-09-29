package main

import (
	"github.com/joho/godotenv"
	"log"
	"missing-persons-scrapper/pkg/countries/croatia"
	"missing-persons-scrapper/pkg/countries/romania"
	"missing-persons-scrapper/pkg/storage"
)

func main() {
	loadEnv()
	storage.Connect()
	migrate()

	run()
}

func loadEnv() {
	err := godotenv.Load("../.env")

	if err != nil {
		log.Fatal(err)
	}
}

func run() {
	p := newParallel()
	p.add(func() { croatia.Start() })
	p.add(func() { romania.Start() })

	p.wait()
}

func migrate() {
	if err := croatia.Migrate(); err != nil {
		log.Fatalln(err)
	}

	if err := romania.Migrate(); err != nil {
		log.Fatalln(err)
	}
}
