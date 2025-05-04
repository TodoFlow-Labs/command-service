package config

import (
	"flag"
	"os"
)

type Config struct {
	HTTPAddr   string
	NATSURL    string
	LogLevel   string
	MetricsAddr string
}

func Load() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.HTTPAddr, "http-addr", ":3333", "HTTP listen address")
	flag.StringVar(&cfg.NATSURL,  "nats-url",  os.Getenv("NATS_URL"), "NATS server URL")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level")
	flag.StringVar(&cfg.MetricsAddr, "metrics-addr", ":9090", "Metrics listen address")
	flag.Parse()
	return cfg, nil
}