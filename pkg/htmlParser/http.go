package htmlParser

import (
	"io"
	"missing-persons-scrapper/pkg/httpClient"
)

func GetBody(url string) ([]byte, error) {
	response, err := httpClient.SendRequest(url)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}
