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

package utils

import (
	"math"
	"strconv"
	"time"
)

func PtrS(s string) *string {
	return &s
}

func PtrInt32(s int32) *int32 {
	return &s
}

func PtrInt64(s int64) *int64 {
	return &s
}

func PtrBool(s bool) *bool {
	return &s
}

func Format(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func MustStrToInt(s string) int {
	num, _ := strconv.Atoi(s)
	return num
}

func StrToInt(s string) (int, error) {
	num, err := strconv.Atoi(s)
	if err != nil {
		return -1, err
	}
	return num, nil
}

func StrToInt32(s string) (int32, error) {
	num, err := strconv.Atoi(s)
	if err != nil {
		return -1, err
	}
	return int32(num), nil
}

func KeepTwoDecimalPlaces(in float64) float64 {
	return math.Round(in*100) / 100
}

func MustToStr(data interface{}) string {
	if data == nil {
		return ""
	}
	if str, ok := data.(string); ok {
		return str
	}
	return ""
}

func MustToFloat64(data interface{}) float64 {
	if data == nil {
		return float64(0)
	}
	if ret, ok := data.(float64); ok {
		return ret
	}
	return float64(0)
}
