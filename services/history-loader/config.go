package main

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the application configuration.
type Config struct {
	Env              string       `default:"local" json:"env"`
	ServiceName      string       `default:"history-loader" split_words:"true" json:"service_name"`
	ServiceNamespace string       `default:"chat" split_words:"true" json:"service_namespace"`
	Cassandra        Cassandra    `json:"cassandra"`
	Log              Log          `json:"log"`
	HTTP             HTTPCfg      `json:"http"`
	JwksURL          string       `split_words:"true" json:"jwks_url"`
	LogConfig        bool         `default:"false" split_words:"true" json:"log_config"`
	Telemetry        TelemetryCfg `json:"telemetry"`
}

type Log struct {
	Level     string `default:"debug" json:"level"`
	AddSource bool   `split_words:"true" json:"add_source"`
	JSON      bool   `json:"json"`
}

// HTTPCfg represents the HTTP server configuration.
type HTTPCfg struct {
	ReqLogLevel       string        `default:"debug" split_words:"true" json:"req_log_level"`
	ReadHeaderTimeout time.Duration `default:"10s" split_words:"true" json:"read_header_timeout"`
	ReadTimeout       time.Duration `default:"30s" split_words:"true" json:"read_timeout"`
	WriteTimeout      time.Duration `default:"30s" split_words:"true" json:"write_timeout"`
	Port              int           `default:"8083" split_words:"true" json:"port"`
	ShutdownTimeout   time.Duration `default:"10s" split_words:"true" json:"shutdown_timeout"`
	CORS              HTTPCorsCfg   `json:"cors"`
}

type HTTPCorsCfg struct {
	AllowedOrigins []string `default:"http://localhost:5173" split_words:"true" json:"allowed_origins"`
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
