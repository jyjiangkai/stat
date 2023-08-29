package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ResponseWithSuccess(ctx *gin.Context, data any) {
	if data == nil || &data == nil {
		data = gin.H{}
	}
	ctx.JSONP(http.StatusOK, data)
}

func ResponseWithError(ctx *gin.Context, err error) {
	em, ok := err.(errorMessage)
	if ok {
		ctx.JSONP(em.HTTPCode, em)
	} else {
		em, _ = ErrUnknown.(errorMessage)
		ctx.JSONP(em.HTTPCode, em.WithError(err))
	}
}
