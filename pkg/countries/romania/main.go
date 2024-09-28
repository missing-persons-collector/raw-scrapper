package romania

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
	"strconv"
	"strings"
	"sync"
)

func Start() {
	pages, err := getNumOfPages("https://www.politiaromana.ro/ro/persoane-disparute")
	if err != nil {
		fmt.Println(fmt.Errorf("failed getting pages. Cannot continue: %w", err))
		return
	}

	wg := &sync.WaitGroup{}
	for _, p := range pages {
		wg.Add(1)

		anchors, err := getList(fmt.Sprintf("https://www.politiaromana.ro/ro/persoane-disparute&page=%d", p))
		if err != nil {
			log.Println(fmt.Errorf("failed to get list: page: %d: %w", p, err))
			return
		}

		for _, a := range anchors {
			href := htmlParser.Attr("href", a.Attr)

			personId := getPersonIDFromHref(href)
			personPage, err := getPersonPage(href)
			if err != nil {
				log.Println(fmt.Errorf("failed to get individual person page: page: %d: %w", p, err))
				continue
			}

			tokens := make([]string, 0)
			if err := getBasicInfo(personPage, &tokens); err != nil {
				log.Println(fmt.Errorf("failed to get basic info: page: %d, %w", p, err))
				continue
			}

			if err := getDescription(personPage, &tokens); err != nil {
				log.Println(fmt.Errorf("failed to get person description: page: %d, %w", p, err))
				continue
			}

			if err := getDetails(personPage, &tokens); err != nil {
				log.Println(fmt.Errorf("failed to get person details: page: %d, %w", p, err))
				continue
			}

			img, err := getImage(personPage)
			if err != nil {
				// we don't have to react if the image src is not there, maybe it will be on one
				// of the next runs of this program
			}

			if err := tryDbOperation(tokens, personId, img, createUniqueIdentifier(tokens)); err != nil {
				log.Fatalln(err)
			}
		}

		fmt.Printf("Finished page %d\n", p)
	}

	wg.Wait()
}

func tryDbOperation(tokens []string, personId string, imgSrc, uniqueIdentifier string) error {
	if err := storage.DB.Transaction(func(tx *gorm.DB) error {
		createPerson := func() RawData {
			b, _ := json.Marshal(tokens)
			return NewRawData(b, personId, uniqueIdentifier)
		}

		getImage := func() ([]byte, string, error) {
			buff := strings.Split(imgSrc, ".")
			if len(buff) < 2 {
				return nil, "", errors.New("cannot extract image extension")
			}

			body, err := downloadImage(imgSrc)
			return body, buff[len(buff)-1], err
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

func getPersonIDFromHref(href string) string {
	s := strings.Split(href, "-")
	return s[len(s)-1]
}

func getBasicInfo(page *html.Node, tokens *[]string) error {
	docs, err := cascadia.Parse(".descDetaliiDisparuti *")
	if err != nil {
		return err
	}

	elems := cascadia.QueryAll(page, docs)

	for _, e := range elems {
		if e.FirstChild != nil {
			*tokens = append(*tokens, e.FirstChild.Data)
		}
	}

	return nil
}

func getDescription(page *html.Node, tokens *[]string) error {
	docs, err := cascadia.Parse(".semnalmenteDisparuti p")
	if err != nil {
		return err
	}

	elem := cascadia.Query(page, docs)

	if elem != nil && elem.FirstChild != nil {
		*tokens = append(*tokens, elem.FirstChild.Data)
	}

	return nil
}

func getImage(page *html.Node) (string, error) {
	docs, err := cascadia.Parse(".pozaDetaliiDisparuti img")
	if err != nil {
		return "", err
	}

	elem := cascadia.Query(page, docs)

	if elem != nil {
		for _, e := range elem.Attr {
			if e.Key == "src" {
				return e.Val, nil
			}
		}
	}

	return "", fmt.Errorf("could not find image")
}

func getDetails(page *html.Node, tokens *[]string) error {
	docs, err := cascadia.Parse(".detaliiSuplimentareDisparuti p")
	if err != nil {
		return err
	}

	elem := cascadia.Query(page, docs)

	if elem != nil && elem.FirstChild != nil {
		*tokens = append(*tokens, elem.FirstChild.Data)
	}

	return nil
}

func createUniqueIdentifier(tokens []string) string {
	joined := strings.Join(tokens, "")
	h := sha256.New()
	h.Write([]byte(joined))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func getNumOfPages(url string) ([]int64, error) {
	body, err := htmlParser.GetBody(fmt.Sprintf(url))
	if err != nil {
		return nil, err
	}

	parsed, err := htmlParser.Parse(string(body))
	if err != nil {
		return nil, err
	}

	doc, err := cascadia.Parse("#num_page option")
	final := cascadia.QueryAll(parsed, doc)

	pages := make([]int64, len(final))
	for i, f := range final {
		p, err := strconv.ParseInt(f.FirstChild.Data, 10, 32)
		if err != nil {
			fmt.Errorf("Cannot convert page to number: %v", err)
		}

		pages[i] = p
	}

	return pages, nil
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

	doc, err := cascadia.Parse(".contentList .boxPoza a")
	final := cascadia.QueryAll(parsed, doc)

	return final, nil
}

func getPersonPage(url string) (*html.Node, error) {
	body, err := htmlParser.GetBody(url)
	if err != nil {
		return nil, err
	}

	parsed, err := htmlParser.Parse(string(body))
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

func updateRecord(id int, tokens []string, personId, uniqueIdentifier string, tx *gorm.DB) *gorm.DB {
	b, _ := json.Marshal(tokens)
	data := NewRawData(b, personId, uniqueIdentifier)
	data.ID = id

	return tx.Save(&data)
}
