package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	ErrUnsupportedApp          = newErrorMessage(http.StatusBadRequest, 4101, "unsupported app")
	ErrParseBody               = newErrorMessage(http.StatusBadRequest, 4102, "failed to parse request body data")
	ErrParseResourceID         = newErrorMessage(http.StatusBadRequest, 4103, "failed to parse resource id")
	ErrParsePaging             = newErrorMessage(http.StatusBadRequest, 4104, "failed to parse paging parameters")
	ErrParseRange              = newErrorMessage(http.StatusBadRequest, 4105, "failed to parse range parameters")
	ErrInvalidMissingParameter = newErrorMessage(http.StatusBadRequest, 4106, "missing parameters")
	ErrPageArgumentsTooLarge   = newErrorMessage(http.StatusBadRequest, 4107, "page arguments too large")
	ErrInvalidResourceStatus   = newErrorMessage(http.StatusBadRequest, 4108, "invalid resource status")
	ErrFailedExchange          = newErrorMessage(http.StatusBadRequest, 4109, "failed to exchange an authorization code for a token")
	ErrShopifyNeedShopName     = newErrorMessage(http.StatusBadRequest, 4110, "shopify integration need a shop name")
	ErrInvalidID               = newErrorMessage(http.StatusBadRequest, 4111, "invalid mongodb id")
	ErrProcessIsNil            = newErrorMessage(http.StatusBadRequest, 4112, "process is nil")
	ErrQuotaLimit              = newErrorMessage(http.StatusBadRequest, 4113, "too many resource, quota limit")
	ErrInvalidParameter        = newErrorMessage(http.StatusBadRequest, 4114, "invalid parameters")
	ErrUnsupportedCurrency     = newErrorMessage(http.StatusBadRequest, 4115, "unsupported currency")
	ErrUnsupportedMethod       = newErrorMessage(http.StatusBadRequest, 4116, "unsupported method")
	ErrVerificationFailed      = newErrorMessage(http.StatusBadRequest, 4117, "verification failed")
	ErrUnsupportedKind         = newErrorMessage(http.StatusBadRequest, 4118, "unsupported kind")

	ErrUnauthorized         = newErrorMessage(http.StatusUnauthorized, 4201, "unauthorized")
	ErrIDTokenNotFound      = newErrorMessage(http.StatusForbidden, 4301, "no id_token field in oauth2 token")
	ErrInvalidState         = newErrorMessage(http.StatusForbidden, 4302, "invalid state parameter")
	ErrFailedVerification   = newErrorMessage(http.StatusForbidden, 4303, "failed to verify ID Token")
	ErrResourceNotFound     = newErrorMessage(http.StatusNotFound, 4401, "requested resource not found")
	ErrResourceAlreadyExist = newErrorMessage(http.StatusConflict, 4501, "resource you requested already exist")

	ErrInternal              = newErrorMessage(http.StatusInternalServerError, 5001, "internal error")
	ErrUnknown               = newErrorMessage(http.StatusInternalServerError, 5002, "unknown error")
	ErrRecoverableConnection = newErrorMessage(http.StatusInternalServerError, 5003,
		"this connection error by accident, please edit/enable/disable again later, if you still see this error, please contact us")
	ErrUnrecoverableConnection = newErrorMessage(http.StatusInternalServerError, 5004,
		"this connection is unrecoverable, please delete it and recreating")
	ErrHostExhausted = newErrorMessage(http.StatusInternalServerError, 5005,
		"ingress host resource is exhausted")
)

type ErrorMessage interface {
	error
	WithMessage(msg string) ErrorMessage
	WithError(err error) ErrorMessage
	IsSame(err error) bool
}

type errorMessage struct {
	Err      string `json:"error"`
	Message  string `json:"message,omitempty"`
	HTTPCode int    `json:"http_code"`
	AppCode  int    `json:"app_code"`
}

func newErrorMessage(httpCode, appCode int, errorMsg string) ErrorMessage {
	return errorMessage{
		HTTPCode: httpCode,
		AppCode:  appCode,
		Err:      errorMsg,
	}
}
func (em errorMessage) WithError(err error) ErrorMessage {
	if err == nil {
		return em
	}
	return em.WithMessage(err.Error())
}

func (em errorMessage) WithMessage(msg string) ErrorMessage {
	if msg == "" {
		return em
	}
	m := errorMessage{
		HTTPCode: em.HTTPCode,
		AppCode:  em.AppCode,
		Err:      em.Err,
		Message:  msg,
	}
	if em.Message != "" {
		m.Message = fmt.Sprintf("%s\n%s", msg, em.Message)
	}
	return m
}

func (em errorMessage) Error() string {
	data, _ := json.Marshal(em)
	return string(data)
}

func (em errorMessage) IsSame(err error) bool {
	m, ok := err.(errorMessage)
	if !ok {
		return false
	}
	return em.AppCode == m.AppCode
}
