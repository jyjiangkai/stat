// Copyright 2023 Linkall Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jyjiangkai/stat/api"
	"github.com/jyjiangkai/stat/internal/services"
	"github.com/jyjiangkai/stat/log"
)

type ActionController struct {
	svc *services.ActionService
}

func NewActionController(handler *services.ActionService) *ActionController {
	return &ActionController{
		svc: handler,
	}
}

func (ac *ActionController) List(ctx *gin.Context) (any, error) {
	pg := api.Page{}
	if err := ctx.BindQuery(&pg); err != nil {
		return nil, api.ErrParsePaging
	}
	if pg.SortBy == "null" {
		pg.SortBy = "time"
	}
	filter := api.NewFilter()
	if err := ctx.Bind(&filter); err != nil {
		log.Error(ctx).Err(err).Msg("failed to parse filting parameters")
		// TODO(jiangkai): fix me
		if !strings.Contains(err.Error(), "EOF") {
			return nil, api.ErrParseFilting.WithError(err)
		}
	}
	kind, _ := ctx.GetQuery(QueryOfUserKind)
	userType, _ := ctx.GetQuery(QueryOfUserType)
	opts := &api.ListOptions{
		KindSelector: kind,
		TypeSelector: userType,
	}
	result, err := ac.svc.List(ctx, pg, filter, opts)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (ac *ActionController) Get(ctx *gin.Context) (any, error) {
	pg := api.Page{}
	if err := ctx.BindQuery(&pg); err != nil {
		return nil, api.ErrParsePaging
	}
	pg.SortBy = "time"
	opts := &api.GetOptions{}
	result, err := ac.svc.Get(ctx, ctx.Param(ParamOfUserOID), pg, opts)
	if err != nil {
		return nil, err
	}
	return result, nil
}
