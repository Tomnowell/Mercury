package signalling

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Tomnowell/Mercury/internal/registry"
)

type RegistryClient interface {
	Lookup(number registry.PhoneNumber) (*registry.Record, error)
}

type HTTPRegistryClient struct {
	BaseURL string
}

func NewHTTPRegistryClient(baseURL string) *HTTPRegistryClient {
	return &HTTPRegistryClient{BaseURL: baseURL}

}

func (client *HTTPRegistryClient) Lookup(number registry.PhoneNumber) (*registry.Record, error) {
	url := client.BaseURL + "/lookup/" + number.String()

	response, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Lookup Failed: %s", response.Status)
	}

	var out struct {
		Number string `json:"number"`
		IP     string `json:"ip"`
		Port   int    `json:"port"`
	}

	err = json.NewDecoder(response.Body).Decode(&out)

	if err != nil {
		return nil, err
	}

	parsedNumber, err := registry.ParsePhoneNumber(out.Number)
	if err != nil {
		return nil, err
	}

	return &registry.Record{
		Number: parsedNumber,
		Endpoint: registry.Endpoint{
			IP:   out.IP,
			Port: out.Port,
		},
	}, nil
}
