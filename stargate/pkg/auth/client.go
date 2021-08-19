package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type tableBasedTokenProvider struct {
	client   *client
	username string
	password string
}

type client struct {
	serviceURL string
	httpClient *http.Client
}

type authResponse struct {
	AuthToken string `json:"authToken"`
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewTableBasedTokenProvider(serviceURL, username, password string) AuthProvider {
	return tableBasedTokenProvider{
		client:   getClient(serviceURL),
		username: username,
		password: password,
	}
}

// TODO: [doug] Figure out how to cache
func (t tableBasedTokenProvider) GetToken(ctx context.Context) (string, error) {
	authReq := authRequest{
		Username: t.username,
		Password: t.password,
	}
	jsonString, err := json.Marshal(authReq)
	if err != nil {
		log.Errorf("error marshalling request: %v", err)
		return "", fmt.Errorf("error marshalling request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.client.serviceURL, bytes.NewBuffer(jsonString))
	if err != nil {
		log.Errorf("error creating request: %v", err)
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	response, err := t.client.httpClient.Do(req)
	if err != nil {
		log.Errorf("error calling auth service: %v", err)
		return "", fmt.Errorf("error calling auth service: %v", err)
	}

	defer func() {
		err := response.Body.Close()
		if err != nil {
			log.Warnf("unable to close response body: %v", err)
		}
	}()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("error reading response body: %v", err)
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	ar := authResponse{}
	err = json.Unmarshal(body, &ar)
	if err != nil {
		log.Errorf("error unmarshalling response body: %v", err)
		return "", fmt.Errorf("error unmarshalling response body: %v", err)
	}

	return ar.AuthToken, nil
}

func getClient(serviceURL string) *client {
	return &client{
		serviceURL: serviceURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}
