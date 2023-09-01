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
	"github.com/gin-gonic/gin"

	"github.com/jyjiangkai/stat/internal/services"
)

type CollectorController struct {
	svc *services.CollectorService
}

func NewCollectorController(handler *services.CollectorService) *CollectorController {
	return &CollectorController{
		svc: handler,
	}
}

func (cc *CollectorController) Collect(ctx *gin.Context) (any, error) {
	kind, _ := ctx.GetQuery(QueryOfUserKind)
	err := cc.svc.Collect(ctx, kind)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
