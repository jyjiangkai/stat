package mailchimp

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sync"

	"gopkg.in/resty.v1"

	"github.com/jyjiangkai/stat/log"
)

var (
	cfg         Config
	client      *resty.Client
	once        sync.Once
	defaultTags []string = []string{"vanus_ai", "no_knowledge_base"}
)

type Config struct {
	Enable     bool   `yaml:"enable"`
	WebhookUrl string `yaml:"webhook_url"`
}

type MailChimp struct {
	Email string   `json:"email" yaml:"email"`
	Tags  []string `json:"tags" yaml:"tags"`
}

func Init(ctx context.Context, c Config) {
	once.Do(func() {
		cfg = c
		client = resty.New()
		log.Info(ctx).Str("webhook_url", c.WebhookUrl).Msgf("the mailchimp subscription function has been %s\n", mailchimpSwitchStatus(c.Enable))
	})
}

func AddMember(ctx context.Context, email string) error {
	if !cfg.Enable {
		log.Info(ctx).Str("email", email).Msg("mailchimp function is disable, no need add member")
		return nil
	}
	req := &MailChimp{
		Email: email,
		Tags:  defaultTags,
	}
	resp, err := client.R().SetBody(req).Post(cfg.WebhookUrl)
	if err == handleHTTPResponse(ctx, resp, err) {
		return err
	}
	return nil
}

func ValidateEmail(email string) bool {
	regex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,4}$`
	match, _ := regexp.MatchString(regex, email)
	return match
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

func mailchimpSwitchStatus(enable bool) string {
	if enable {
		return "enabled"
	}
	return "disabled"
}
