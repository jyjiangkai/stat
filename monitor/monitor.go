package monitor

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"gopkg.in/resty.v1"

	"github.com/jyjiangkai/stat/log"
)

var (
	cfg    Config
	client *resty.Client
	once   sync.Once
)

type Config struct {
	Enable     bool   `yaml:"enable"`
	WebhookUrl string `yaml:"webhook_url"`
}

type Alarm struct {
	Message string `json:"message" yaml:"message"`
}

func Init(ctx context.Context, c Config) {
	once.Do(func() {
		cfg = c
		client = resty.New()
		log.Info(ctx).Str("webhook_url", c.WebhookUrl).Msgf("the monitoring alarm function has been %s\n", monitorSwitchStatus(c.Enable))
	})
}

func SendAlarm(ctx context.Context, message string) error {
	req := &Alarm{
		Message: message,
	}
	resp, err := client.R().SetBody(req).Post(cfg.WebhookUrl)
	if err == handleHTTPResponse(ctx, resp, err) {
		return err
	}
	return nil
}

func handleHTTPResponse(ctx context.Context, res *resty.Response, err error) error {
	if err != nil {
		log.Warn(ctx).Err(err).Msg("HTTP request failed")
		return err
	}
	if res.StatusCode() != http.StatusOK {
		log.Warn(ctx).Err(err).
			Int("status_code", res.StatusCode()).
			Str("body", string(res.Body())).
			Msg("HTTP response not 200 failed")
		return errors.New(string(res.Body()))
	}
	return nil
}

func monitorSwitchStatus(enable bool) string {
	if enable {
		return "enabled"
	}
	return "disabled"
}
