package main

import (
	"encoding/json"
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
	HttpOtelEndpoint string `default:"localhost:4318" split_words:"true" json:"http_otel_endpoint"`
	GrpcOtelEndpoint string `default:"localhost:4317" split_words:"true" json:"grpc_otel_endpoint"`
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
