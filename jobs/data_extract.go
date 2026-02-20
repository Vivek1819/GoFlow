package jobs

import (
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func executeDataExtract(payload map[string]interface{}) (int, []byte, error) {

	url, ok := payload["url"].(string)
	if !ok || url == "" {
		return 0, nil, fmt.Errorf("missing 'url'")
	}

	selector, ok := payload["selector"].(string)
	if !ok || selector == "" {
		return 0, nil, fmt.Errorf("missing 'selector'")
	}

	extractType := "text"
	if t, ok := payload["extract"].(string); ok {
		extractType = t
	}

	attrName := ""
	if extractType == "attr" {
		a, ok := payload["attr"].(string)
		if !ok || a == "" {
			return 0, nil, fmt.Errorf("missing 'attr' for attr extract type")
		}
		attrName = a
	}

	client := &http.Client{
		Timeout: 8 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return resp.StatusCode, nil,
			fmt.Errorf("http status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	var results []string

	doc.Find(selector).Each(func(i int, s *goquery.Selection) {

		switch extractType {

		case "text":
			results = append(results, s.Text())

		case "html":
			html, err := s.Html()
			if err == nil {
				results = append(results, html)
			}

		case "attr":
			val, exists := s.Attr(attrName)
			if exists {
				results = append(results, val)
			}
		}
	})

	response := map[string]interface{}{
		"url":      url,
		"selector": selector,
		"count":    len(results),
		"results":  results,
	}

	jsonBytes, err := jsonMarshalSafe(response)
	if err != nil {
		return 0, nil, err
	}

	return 200, jsonBytes, nil
}