package utils

import (
	"net/http"
	"time"
)

func Fetch(url string) (*http.Response, error) {
	return FetchWithRetry(url, 2)
}

func FetchWithRetry(url string, maxRetries int) (*http.Response, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	var resp *http.Response
	var err error

	for i := 0; i <= maxRetries; i++ {
		req, errReq := http.NewRequest("GET", url, nil)
		if errReq != nil {
			return nil, errReq
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0")
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")

		resp, err = client.Do(req)
		if err == nil {
			if resp.StatusCode < 500 {
				return resp, nil
			}
			// It's a 5xx error, we might want to retry
			if i < maxRetries {
				resp.Body.Close()
				time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
				continue
			}
			return resp, nil
		}

		if i < maxRetries {
			time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
			continue
		}
	}

	return resp, err
}
