package poller

import (
	"context"
	"encoding/json"
	"time"

	"github.com/amerkurev/doku/app/docker"
	"github.com/amerkurev/doku/app/store"
	log "github.com/sirupsen/logrus"
)

const (
	pollingShortInterval = time.Second
	pollingLongInterval  = time.Minute
)

// Run starts a goroutine to poll the Docker daemon.
func Run(ctx context.Context, d *docker.Client) {
	messages, errs := d.DockerEvents(ctx)
	numMessages := 0 // count of Docker daemon events.

	go func() {
		// run it immediately on start
		poll(ctx, d)

		// run poll with interval while context is not cancel
		for {
			select {
			case m := <-messages:
				if docker.IsSignificantEvent(m.Type) {
					numMessages++
				}
			case err := <-errs:
				if err != nil {
					log.WithField("err", err).Error("failed to listen to docker events")

					// reconnect to the Docker daemon
					select {
					case <-time.After(pollingLongInterval):
						messages, errs = d.DockerEvents(ctx)
					case <-ctx.Done():
						log.Info("gracefully poller shutdown")
						return
					}
				}
			case <-ctx.Done():
				log.Info("gracefully poller shutdown")
				return
			case <-time.After(pollingShortInterval):
				// execute poll only if was happened Docker daemon events
				if numMessages > 0 {
					numMessages = 0
					poll(ctx, d)
				}
			case <-time.After(pollingLongInterval):
				// forced poll every minute
				poll(ctx, d)
			}
		}
	}()
}

func poll(ctx context.Context, d *docker.Client) {
	defer elapsed("yet another poll execution is done")()

	r, err := d.DockerInfo(ctx)
	if err != nil {
		log.WithField("err", err).Error("failed to docker request")
		return
	}

	s := store.Get()
	s.Set("latestPolling", r)

	b, err := json.Marshal(r.Info)
	if err != nil {
		log.WithField("err", err).Error("failed to serialize docker info")
		return
	}
	s.Set("json.dockerInfo", b)

	b, err = json.Marshal(r.DiskUsage)
	if err != nil {
		log.WithField("err", err).Error("failed to serialize docker disk usage")
		return
	}
	s.Set("json.dockerDiskUsage", b)

	// wake up those who are waiting.
	s.NotifyAll()
}

func elapsed(what string) func() {
	start := time.Now()
	return func() {
		log.WithField("took", time.Since(start)).Debug(what)
	}
}