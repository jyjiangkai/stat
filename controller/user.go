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

const (
	ParamOfUserOID  = "oid"
	QueryOfUserKind = "kind"
	QueryOfUserType = "type"
	QueryOfOperator = "operator"
)

type UserController struct {
	svc *services.UserService
}

func NewUserController(handler *services.UserService) *UserController {
	return &UserController{
		svc: handler,
	}
}

func (uc *UserController) List(ctx *gin.Context) (any, error) {
	pg := api.Page{}
	if err := ctx.BindQuery(&pg); err != nil {
		log.Error(ctx).Err(err).Msg("failed to parse page parameters")
		return nil, api.ErrParsePaging
	}
	req := api.NewRequest()
	if err := ctx.Bind(&req); err != nil {
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
	result, err := uc.svc.List(ctx, pg, req, opts)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *UserController) Get(ctx *gin.Context) (any, error) {
	kind, _ := ctx.GetQuery(QueryOfUserKind)
	opts := &api.GetOptions{
		KindSelector: kind,
	}
	result, err := uc.svc.Get(ctx, ctx.Param(ParamOfUserOID), opts)
	if err != nil {
		return nil, err
	}
	return result, nil
}
