// Copyright 2026 thumbrise
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

package backoff

import (
	"math"
	"time"
)

func Backoff(attempt int, base time.Duration, multiplier float64) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	if multiplier <= 0 {
		multiplier = 1.0
	}

	factor := math.Pow(multiplier, float64(attempt))

	return time.Duration(float64(base) * factor)
}
