package repository

import (
	"errors"
	"time"

	"github.com/monitoror/monitoror/monitorables/ping/api"
	"github.com/monitoror/monitoror/monitorables/ping/api/models"
	"github.com/monitoror/monitoror/monitorables/ping/config"

	goPing "github.com/sparrc/go-ping"
)

type (
	pingRepository struct {
		config *config.Ping
	}
)

func NewPingRepository(config *config.Ping) api.Repository {
	return &pingRepository{config}
}

func (r *pingRepository) ExecutePing(hostname string) (*models.Ping, error) {
	pinger, err := goPing.NewPinger(hostname)
	if err != nil {
		return nil, err
	}

	pinger.Count = r.config.Count
	pinger.Interval = time.Millisecond * time.Duration(r.config.Interval)
	pinger.Timeout = time.Millisecond * time.Duration(r.config.Timeout)
	pinger.SetPrivileged(true) // NEED ROOT PRIVILEGED

	pinger.Run()
	stats := pinger.Statistics()

	if stats.PacketsRecv == 0 {
		return nil, errors.New("ping failed")
	}

	ping := &models.Ping{}
	ping.Min = stats.MinRtt
	ping.Max = stats.MaxRtt
	ping.Average = stats.AvgRtt

	return ping, nil
}
