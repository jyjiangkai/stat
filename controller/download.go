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

const (
	QueryOfUrl = "url"
)

type DownloadController struct {
	svc *services.DownloadService
}

func NewDownloadController(handler *services.DownloadService) *DownloadController {
	return &DownloadController{
		svc: handler,
	}
}

func (dc *DownloadController) Get(ctx *gin.Context) {
	url, _ := ctx.GetQuery(QueryOfUrl)
	dc.svc.Download(ctx, url)
}
