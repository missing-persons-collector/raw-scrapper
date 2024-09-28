package croatia

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"log"
	"missing-persons-scrapper/pkg/storage"
	"testing"
)

func loadEnv() {
	err := godotenv.Load("../../.env")

	if err != nil {
		log.Fatal(err)
	}
}

/*
*
This should only check that the database is populated and constant after multiple runs.
It can fail if the missing persons database is updated but the only thing a programmer
must do is change the assertion, since that is the only thing that is important.

The other important thing is that the number of raw data in the database is constant
after multiple runs.
*/
func TestRunning(t *testing.T) {
	loadEnv()
	storage.Connect()
	reset := []string{
		"TRUNCATE table croatia_scrapped",
		"TRUNCATE table croatia_images",
		"ALTER SEQUENCE croatia_scrapped_id_seq RESTART WITH 1",
		"ALTER SEQUENCE croatia_images_id_seq RESTART WITH 1",
	}

	for _, r := range reset {
		storage.DB.Exec(r)
	}

	Start()

	var scrappedDataCount int
	res := storage.DB.Raw(fmt.Sprintf("SELECT COUNT(id) FROM %s", Croatia_Scrapper_Table)).Scan(&scrappedDataCount)
	assert.Nil(t, res.Error)
	assert.Equal(t, scrappedDataCount, 2877)

	var scrappedImageCount int
	res = storage.DB.Raw(fmt.Sprintf("SELECT COUNT(id) FROM %s", Croatia_Images_Table)).Scan(&scrappedImageCount)
	assert.Nil(t, res.Error)
	assert.Equal(t, scrappedImageCount, 2863)

	storage.Close()
}
