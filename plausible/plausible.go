package plausible

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/jyjiangkai/stat/log"
	"gopkg.in/resty.v1"
)

const (
	BaseUrl string = "https://plausible.io"
	ApiKey  string = "LxDml1A0s71E5cm4VgijkBx7t5zrWHwOtj_wDRLVCqzbWaA4CHANjKs3UpYXTAMq"
)

var (
	// cfg    Config
	client *resty.Client
	once   sync.Once
)

// type Config struct {
// 	Enable     bool   `yaml:"enable"`
// 	WebhookUrl string `yaml:"webhook_url"`
// }

type Response struct {
	Results Results `json:"results" yaml:"results"`
}

type Results struct {
	Visitors Visitors `json:"visitors" yaml:"visitors"`
}

type Visitors struct {
	Value int64 `json:"value" yaml:"value"`
}

func init() {
	once.Do(func() {
		fmt.Println("init")
		client = resty.New()
	})
}

func GetVisitors(ctx context.Context, site, page, date string) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/stats/aggregate?site_id=%s&period=custom&metrics=visitors&filters=event:page==%s&period=custom&date=%s,%s", BaseUrl, site, page, date, date)
	rep, err := client.R().SetHeader("Authorization", fmt.Sprintf("Bearer %s", ApiKey)).Get(url)
	if handleHTTPResponse(ctx, rep, err) != nil {
		return 0, err
	}
	response := &Response{}
	err = json.Unmarshal(rep.Body(), response)
	if err != nil {
		return 0, err
	}
	return response.Results.Visitors.Value, nil
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
