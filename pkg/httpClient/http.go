package httpClient

import (
	"net/http"
	"time"
)

func SendRequest(url string) (*http.Response, error) {
	var backoffSchedule = []time.Duration{
		1 * time.Second,
		3 * time.Second,
		10 * time.Second,
	}

	var res *http.Response
	var err error

	for _, backoff := range backoffSchedule {
		request, rErr := NewRequest(Request{
			Headers: nil,
			Url:     url,
			Method:  "GET",
			Body:    nil,
		})

		if rErr != nil {
			return nil, rErr
		}

		res, err = Make(request, NewClient(ClientParams{}))

		if err != nil {
			time.Sleep(backoff)

			continue
		}

		return res, err
	}

	return res, err
}
