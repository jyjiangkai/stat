package router

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jyjiangkai/stat/api"
	"github.com/jyjiangkai/stat/controller"
	"github.com/jyjiangkai/stat/log"
)

func RegisterUsersRouter(group *gin.RouterGroup,
	ctrl *controller.UserController) {
	wrapRouterGroup(group, http.MethodGet, "", ctrl.List)

	pathID := fmt.Sprintf("/:%s", controller.ParamOfUserOID)
	wrapRouterGroup(group, http.MethodGet, pathID, ctrl.Get)
}

func RegisterCollectRouter(group *gin.RouterGroup,
	ctrl *controller.CollectorController) {
	wrapRouterGroup(group, http.MethodGet, "", ctrl.Collect)
}

type HandlerFunc func(ctx *gin.Context) (any, error)

func wrapHandlerFunc(f HandlerFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		resp, err := f(ctx)
		if err != nil {
			log.Info(ctx).Err(err).Str("path", ctx.Request.URL.Path).Msg("request has error")
			api.ResponseWithError(ctx, err)
			return
		}
		api.ResponseWithSuccess(ctx, resp)
	}
}

func wrapRouterGroup(group *gin.RouterGroup, method, relativePath string, f HandlerFunc) {
	group.Handle(method, relativePath, wrapHandlerFunc(f))
}
