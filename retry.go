package main

import (
	"time"

	"github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

const _retryAttempts = 3

func failTrying(description string, f func() error) error {
	if err := retryIt(f); err != nil {
		log.Errorf("%s failed after %d attempts", description, _retryAttempts)
		return err
	}
	return nil
}

func dieTrying(description string, f func() error) {
	if err := retryIt(f); err != nil {
		log.Fatalf("%s failed after %d attempts", description, _retryAttempts)
	}
}

func retryIt(f func() error) error {
	return retry.Do(
		func() error {
			if err := f(); err != nil {
				log.Error(err)
				return err
			}
			return nil
		},
		retry.Attempts(_retryAttempts),
		retry.Delay(1000*time.Millisecond),
		retry.MaxJitter(500*time.Millisecond),
	)
}
