package log

import (
	"context"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/jyjiangkai/stat/utils"
)

const (
	KeyNamespace        = "namespace"
	KeyEventbus         = "eventbus"
	KeyConnection       = "connection"
	KeyConnectionID     = "connection_id"
	KeyEventbusID       = "eventbus_id"
	KeyRequestID        = requestId
	KeySubscriptionID   = "subscription_id"
	KeySourceID         = "source_id"
	KeySinkID           = "sink_id"
	KeySubscriptionName = "subscription_name"
	KeyOrder            = "order"
	KeySubscribe        = "subscribe"
	KeyPromotion        = "promotion"
)

var lg zerolog.Logger
var out zerolog.ConsoleWriter

const (
	requestId = "request_id"
	userId    = "user_id"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	out = zerolog.NewConsoleWriter()
	out.TimeFormat = "2006-01-02T15:04:05.999Z07:00"
	lg = zerolog.New(out).With().Caller().Timestamp().Logger()
}

func CustomLogger(ctx *gin.Context, l zerolog.Logger) zerolog.Logger {
	return l.Output(out).With().
		Str(requestId, requestid.Get(ctx)).
		Logger()
}

func Info(ctx context.Context) *zerolog.Event {
	e := lg.Info().
		Str(userId, utils.GetUserID(ctx))
	gCtx, ok := ctx.(*gin.Context)
	if ok {
		e.Str(requestId, requestid.Get(gCtx))
	}
	return e
}

func Warn(ctx context.Context) *zerolog.Event {
	e := lg.Warn().Str(userId, utils.GetUserID(ctx))
	gCtx, ok := ctx.(*gin.Context)
	if ok {
		e.Str(requestId, requestid.Get(gCtx))
	}
	return e
}

func Error(ctx context.Context) *zerolog.Event {
	e := lg.Error().
		Str(userId, utils.GetUserID(ctx))
	gCtx, ok := ctx.(*gin.Context)
	if ok {
		e.Str(requestId, requestid.Get(gCtx))
	}
	return e
}
