// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	terratestlogger "github.com/gruntwork-io/terratest/modules/logger"

	_ "github.com/api7/amesh/e2e/test/amesh"
	_ "github.com/api7/amesh/e2e/test/base"
	_ "github.com/api7/amesh/e2e/test/istio"
)

func runE2E() {
	var err error
	log.DefaultLogger, err = log.NewLogger(
		log.WithLogLevel("info"),
		log.WithSkipFrames(3),
	)
	if err != nil {
		panic(err)
	}

	terratestlogger.Default = terratestlogger.Discard
	terratestlogger.Terratest = terratestlogger.Discard
	terratestlogger.Global = terratestlogger.Discard

	color.NoColor = false
}
