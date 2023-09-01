package utils

import (
	"context"
	"github.com/jyjiangkai/stat/constant"
)

func GetUserID(ctx context.Context) string {
	val := ctx.Value(constant.ContextUserID)
	if val == nil {
		return "system"
	}
	return val.(string)
}
