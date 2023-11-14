package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the application configuration.
type Config struct {
	Env              string       `default:"local" json:"env"`
	ServiceName      string       `default:"history-writer" split_words:"true" json:"service_name"`
	ServiceNamespace string       `default:"chat" split_words:"true" json:"service_namespace"`
	Redis            Redis        `json:"redis"`
	Cassandra        Cassandra    `json:"cassandra"`
	Log              Log          `json:"log"`
	LogConfig        bool         `default:"false" split_words:"true" json:"log_config"`
	Telemetry        TelemetryCfg `json:"telemetry"`
}

type Log struct {
	Level     string `default:"debug" json:"level"`
	AddSource bool   `split_words:"true" json:"add_source"`
	JSON      bool   `json:"json"`
}

type Redis struct {
	Addr string `default:"localhost:6379" json:"addr"`
}

type Cassandra struct {
	Host     string `default:"localhost" json:"host"`
	Keyspace string `default:"chat" json:"keyspace"`
	Log      bool   `json:"log"`
}

type TelemetryCfg struct {
	MeterEndpoint string `default:"localhost:4318" json:"http_trace_endpoint" envconfig:"TELEMETRY_HTTP_METER_ENDPOINT"`
	TraceEndpoint string `default:"localhost:4318" json:"http_meter_endpoint" envconfig:"TELEMETRY_HTTP_TRACE_ENDPOINT"`
	Username      string `json:"-" envconfig:"TELEMETRY_HTTP_TRACE_ENDPOINT_USERNAME"`
	Password      string `json:"-" envconfig:"TELEMETRY_HTTP_TRACE_ENDPOINT_PASSWORD"`
}

func (cfg TelemetryCfg) TraceBasicAuthHeader() map[string]string {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.Username, cfg.Password)))
	return map[string]string{"Authorization": "Basic " + auth}
}

func LoadConfig() Config {
	c := Config{}
	envconfig.MustProcess("", &c)
	if !c.LogConfig {
		return c
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		slog.Error("failed marshal config", slog.Any("err", err))
		return c
	}
	slog.Info("parsed config", slog.String("config", string(b)))
	return c
}
