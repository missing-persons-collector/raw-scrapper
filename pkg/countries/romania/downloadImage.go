package romania

import (
	"errors"
	"fmt"
	"io"
	"missing-persons-scrapper/pkg/httpClient"
	"net/http"
)

func downloadImage(URL string) ([]byte, error) {
	response, err := httpClient.SendRequest(URL)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Request returned non 200 for %s", URL))
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
