package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// Config holds non-secret trading settings. Secrets stay in .env.
type Config struct {
	NiftyLots     int `json:"nifty_lots"`
	BankNiftyLots int `json:"banknifty_lots"`
	TickerSeconds int `json:"ticker_seconds"`
	HoldMinutes   int `json:"hold_minutes"`
}

// App is the configuration loaded at startup by Load.
var App Config

// Load reads and validates the config file. Call it once from main before serving.
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if err := c.validate(); err != nil {
		return fmt.Errorf("invalid %s: %w", path, err)
	}
	App = c
	return nil
}

func (c Config) validate() error {
	var errs []error
	if c.NiftyLots <= 0 {
		errs = append(errs, errors.New("nifty_lots must be > 0"))
	}
	if c.BankNiftyLots <= 0 {
		errs = append(errs, errors.New("banknifty_lots must be > 0"))
	}
	if c.TickerSeconds <= 0 {
		errs = append(errs, errors.New("ticker_seconds must be > 0"))
	}
	if c.HoldMinutes <= 0 {
		errs = append(errs, errors.New("hold_minutes must be > 0"))
	}
	return errors.Join(errs...)
}

// Ticker is the price poll interval.
func (c Config) Ticker() time.Duration {
	return time.Duration(c.TickerSeconds) * time.Second
}

// Hold is how long a worker lives before timing out.
func (c Config) Hold() time.Duration {
	return time.Duration(c.HoldMinutes) * time.Minute
}
