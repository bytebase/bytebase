package gitlab

import (
	"fmt"
	"log"
	"net/http"
)

const (
	ApiPath = "/api/v4"
)

func Delete(instanceURL string, resourcePath string, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s%s", instanceURL, ApiPath, resourcePath)
	log.Printf("url: %s token: %s\n", url, token)
	req, err := http.NewRequest("DELETE",
		url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct DELETE %v (%w)", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed DELETE %v (%w)", url, err)
	}

	return resp, nil
}
