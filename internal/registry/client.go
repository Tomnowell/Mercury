package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

}

type RegisterRequest struct {
	Number string `json:"number"`
	Port   int    `json:"port"`
}

type RegisterResponse struct {
	Status string `json:"status"`
}

func (client *Client) RegisterSelf(number PhoneNumber, port int) error {
	requestBody := RegisterRequest{
		Number: number.String(),
		Port:   port,
	}

	data, err := json.Marshal(requestBody)

	if err != nil {
		return err
	}

	response, err := client.http.Post(
		client.baseURL+"/register/",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned %s", response.Status)
	}
	return nil
}

type LookupResponse struct {
	Number string `json:"number"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
}

func (client *Client) Lookup(number PhoneNumber) (*Endpoint, error) {
	url := fmt.Sprintf("%s/lookup/%s", client.baseURL, number.String())

	response, err := client.http.Get(url)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned %s", response.Status)
	}

	var out LookupResponse

	err = json.NewDecoder(response.Body).Decode(&out)

	if err != nil {
		return nil, err
	}

	return &Endpoint{
		IP:   out.IP,
		Port: out.Port,
	}, nil
}
