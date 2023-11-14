package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the application configuration.
type Config struct {
	Env              string       `default:"local" json:"env"`
	ServiceName      string       `default:"ws-sender" split_words:"true" json:"service_name"`
	ServiceNamespace string       `default:"chat" split_words:"true" json:"service_namespace"`
	Redis            Redis        `json:"redis"`
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
	Port              int           `default:"8081" split_words:"true" json:"port"`
	ShutdownTimeout   time.Duration `default:"10s" split_words:"true" json:"shutdown_timeout"`
}

type TelemetryCfg struct {
	MeterEndpoint string `default:"localhost:4318" json:"http_otel_endpoint" envconfig:"TELEMETRY_HTTP_METER_ENDPOINT"`
	TraceEndpoint string `default:"localhost:4318" json:"http_meter_endpoint" envconfig:"TELEMETRY_HTTP_TRACE_ENDPOINT"`
	Username      string `json:"-" envconfig:"TELEMETRY_HTTP_TRACE_ENDPOINT_USERNAME"`
	Password      string `json:"-" envconfig:"TELEMETRY_HTTP_TRACE_ENDPOINT_PASSWORD"`
}

func (cfg TelemetryCfg) TraceBasicAuthHeader() map[string]string {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.Username, cfg.Password)))
	return map[string]string{"Authorization": "Basic " + auth}
}

type Redis struct {
	Addr string `default:"localhost:6379" json:"addr"`
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
