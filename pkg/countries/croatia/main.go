package croatia

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"gorm.io/gorm"
	"log"
	"missing-persons-scrapper/pkg/htmlParser"
	"missing-persons-scrapper/pkg/storage"
	"strings"
	"sync"
)

func Start() {
	letters := []string{"a", "b", "c", "č", "ć", "d", "đ", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "r", "s", "š", "t", "u", "v", "w", "x", "z", "ž"}

	wg := &sync.WaitGroup{}

	/**
	https://nestali.gov.hr
	Website navigation goes by letters (peoples names) and by that letter, by pages. So every letter can have multiple
	people missing with around 15 per page.

	This program goes through letters and run one goroutine per letter.
	*/
	for _, letter := range letters {
		wg.Add(1)
		page := 1
		defer wg.Done()

		for {
			// get the list of all persons on letter and page
			// if it fails, continue on to the next one
			list, err := getList(fmt.Sprintf("https://nestali.gov.hr/nestale-osobe-403/403?slovo=%s&page=%d", letter, page))
			if err != nil {
				log.Println(fmt.Errorf("failed to get list: letter: %s, page: %d: %w", letter, page, err))
				break
			}

			// if the page that we are on does not have any entries, break from this loop and
			// another letter
			if len(list) == 0 {
				break
			}

			for _, l := range list {
				// get the name of the person so you could get the id (id is the website id)
				name, err := htmlParser.Find(l, ".osoba-ime")
				if err != nil {
					log.Println(fmt.Errorf("failed to find person: letter: %s, page: %d: %w", letter, page, err))
					break
				}

				href := htmlParser.Attr("href", name.Attr)
				if href != "" {
					s := strings.Split(href, "=")

					personId := s[1]
					tokens, image, err := getTokens(personId)

					if err != nil {
						log.Println(fmt.Errorf("failed getting tokens: letter: %s, page: %d: %s; -> %w", letter, page, personId, err))
						break
					}

					uniqueIdentifier := createUniqueIdentifier(tokens)
					if err := tryDbOperation(tokens, personId, uniqueIdentifier, image); err != nil {
						log.Println(err)
						break
					}
				}
			}

			fmt.Printf("Finished letter %s; page %d\n", letter, page)

			page += 1
		}
	}

	wg.Wait()
}

func createUniqueIdentifier(tokens []string) string {
	joined := strings.Join(tokens, "")
	h := sha256.New()
	h.Write([]byte(joined))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func updateRecord(id int, tokens []string, personId, uniqueIdentifier string, tx *gorm.DB) *gorm.DB {
	b, _ := json.Marshal(tokens)
	data := NewRawData(b, personId, uniqueIdentifier)
	data.ID = id

	return tx.Save(&data)
}

func tryDbOperation(tokens []string, personId, uniqueIdentifier, image string) error {
	if err := storage.DB.Transaction(func(tx *gorm.DB) error {
		createPerson := func() RawData {
			b, _ := json.Marshal(tokens)
			return NewRawData(b, personId, uniqueIdentifier)
		}

		getImage := func() ([]byte, string, error) {
			buff := strings.Split(image, ".")
			if len(buff) != 2 {
				return nil, "", errors.New("cannot extract image extension")
			}

			body, err := downloadImage(fmt.Sprintf("https://nestali.gov.hr%s", image))
			return body, buff[1], err
		}

		getCurrentPersonData := func() (err error, id int, rowsAffected int64) {
			var p RawData
			res := tx.Where("unique_identifier = ?", uniqueIdentifier).Select("id").First(&p)

			return res.Error, p.ID, res.RowsAffected
		}

		personErr, id, rowsAffected := getCurrentPersonData()
		/**
		If the user does not exist, create it alongside the scraped image
		*/
		if errors.Is(personErr, gorm.ErrRecordNotFound) {
			person := createPerson()
			if res := tx.Create(&person); res.Error != nil {
				return fmt.Errorf("failed saving to database item_id: %s; -> %v", personId, res.Error)
			}

			img, extension, err := getImage()
			if err != nil {
				// the image could not be downloaded but that is not a reason to throw away the transaction
				// in the next iteration, since this program is in cron, should pick it up. Otherwise, a good
				// logging system should be created for this use case
				return nil
			}

			dbImage := NewDbImage(person.ID, extension, img)
			if res := tx.Create(&dbImage); res.Error != nil {
				return res.Error
			}
		} else if personErr != nil {
			return fmt.Errorf("an error occurred while trying to query the record: %s; -> %w", personId, personErr)
		}

		/**
		If the person exists, update it alongside the image
		*/
		if rowsAffected == 1 && id != 0 {
			if res := updateRecord(id, tokens, personId, uniqueIdentifier, tx); res.Error != nil {
				return fmt.Errorf("failed updating database with item_id: %s; -> %v", personId, res.Error)
			}

			img, extension, err := getImage()
			if err != nil {
				// the image could not be downloaded but that is not a reason to throw away the transaction
				// in the next iteration, since this program is in cron, should pick it up. Otherwise, a good
				// logging system should be created for this use case
				return nil
			}

			var dbImg DbImage
			res := tx.Where("item_id = ?", id).Select("id").First(&dbImg)
			if res.Error != nil {
				return nil
			}
			dbImg.ItemID = id
			dbImg.Extension = extension
			dbImg.Blob = img
			if res := tx.Where("id = ?", dbImg.ID).Save(&dbImg); res.Error != nil {
				return res.Error
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func getList(url string) ([]*html.Node, error) {
	body, err := htmlParser.GetBody(fmt.Sprintf(url))
	if err != nil {
		return nil, err
	}

	parsed, err := htmlParser.Parse(string(body))
	if err != nil {
		return nil, err
	}

	doc, err := cascadia.Parse(".nestali-list li")
	final := cascadia.QueryAll(parsed, doc)

	return final, nil
}

/*
*
Goes to the actual single page of the missing person and scrapps all the data that it can.
It stores that data in an array. How to represent that data should be done later.

This is where the missing person image is also scrapped (the <img> src attribute).
*/
func getTokens(personId string) ([]string, string, error) {
	url := fmt.Sprintf("https://nestali.gov.hr/nestale-osobe-403/403?osoba_id=%s", personId)

	body, err := htmlParser.GetBody(fmt.Sprintf(url))
	if err != nil {
		return nil, "", err
	}

	parsed, err := htmlParser.Parse(string(body))
	if err != nil {
		return nil, "", err
	}

	doc, err := cascadia.Parse(".profile_details_right dl *")
	if err != nil {
		return nil, "", err
	}
	tokens := cascadia.QueryAll(parsed, doc)

	data := make([]string, 0)
	for _, t := range tokens {
		data = append(data, t.FirstChild.Data)
	}

	imgParse, err := cascadia.Parse(".menuLeftPhoto img")
	if err != nil {
		return nil, "", err
	}

	img := cascadia.Query(parsed, imgParse)
	if err != nil {
		return nil, "", err
	}

	return data, htmlParser.Attr("src", img.Attr), nil
}
