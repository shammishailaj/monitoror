package repository

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/monitoror/monitoror/monitorables/pingdom/api"
	"github.com/monitoror/monitoror/monitorables/pingdom/api/models"
	"github.com/monitoror/monitoror/monitorables/pingdom/config"
	"github.com/monitoror/monitoror/pkg/gopingdom"

	pingdomAPI "github.com/jsdidierlaurent/go-pingdom/pingdom"
)

type (
	pingdomRepository struct {
		config *config.Pingdom

		// Pingdom check client
		pingdomCheckAPI            gopingdom.PingdomCheckAPI
		pingdomTransactionCheckAPI gopingdom.PingdomTransactionCheckAPI
	}
)

func NewPingdomRepository(config *config.Pingdom) api.Repository {
	// Remove last /
	config.URL = strings.TrimRight(config.URL, "/")

	client, err := pingdomAPI.NewClientWithConfig(pingdomAPI.ClientConfig{
		BaseURL:  config.URL,
		APIToken: config.Token,
		HTTPClient: &http.Client{
			Timeout: time.Millisecond * time.Duration(config.Timeout),
		},
	})

	// Only if Pingdom URL is not a valid URL
	if err != nil {
		panic(fmt.Sprintf("unable to initiate connection to Pingdom\n. %v\n", err))
	}

	return &pingdomRepository{
		config:                     config,
		pingdomCheckAPI:            client.Checks,
		pingdomTransactionCheckAPI: client.TransactionChecks,
	}
}

func (r *pingdomRepository) GetCheck(id int) (result *models.Check, err error) {
	check, err := r.pingdomCheckAPI.Read(id)
	if err != nil {
		return
	}

	result = &models.Check{
		ID:     check.ID,
		Name:   check.Name,
		Status: check.Status,
	}

	return
}

func (r *pingdomRepository) GetChecks(tags string) (results []models.Check, err error) {
	params := make(map[string]string)
	if tags != "" {
		params["tags"] = tags
	}

	checks, err := r.pingdomCheckAPI.List(params)
	if err != nil {
		return
	}

	for _, check := range checks {
		results = append(results, models.Check{
			ID:     check.ID,
			Name:   check.Name,
			Status: check.Status,
		})
	}

	return
}

func (r *pingdomRepository) GetTransactionCheck(id int) (result *models.Check, err error) {
	check, err := r.pingdomTransactionCheckAPI.Read(id)
	if err != nil {
		return
	}

	result = &models.Check{
		ID:     check.ID,
		Name:   check.Name,
		Status: check.Status,
	}

	return
}

func (r *pingdomRepository) GetTransactionChecks(tags string) (results []models.Check, err error) {
	params := make(map[string]string)
	if tags != "" {
		params["tags"] = tags
	}

	checks, err := r.pingdomTransactionCheckAPI.List(params)
	if err != nil {
		return
	}

	for _, check := range checks {
		results = append(results, models.Check{
			ID:     check.ID,
			Name:   check.Name,
			Status: check.Status,
		})
	}

	return
}
