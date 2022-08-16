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
//
package main

import (
	"context"
	"os"

	"C"
	"github.com/api7/gopkg/pkg/log"
	"github.com/spf13/cobra"

	"github.com/api7/amesh/pkg/amesh"
	"github.com/api7/amesh/pkg/utils"
)

func main() {
	cmd := NewAmeshCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

// NewAmeshCommand creates the root command for apisix-mesh-agent.
func NewAmeshCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "amesh [command] [flags]",
		Short: "An Apache APISIX service mesh mock.",
	}
	cmd.AddCommand(
		NewRunCommand(),
	)
	return cmd
}

// Config contains configurations required for running apisix-mesh-agent.
type Config struct {
	LogLevel        string `json:"log_level" yaml:"log_level"`
	LogOutput       string `json:"log_output" yaml:"log_output"`
	XDSConfigSource string `json:"xds_config_source" yaml:"xds_config_source"`
}

// NewDefaultConfig returns a Config object with all items filled by
// their default values.
func NewDefaultConfig() *Config {
	return &Config{
		LogLevel:  "info",
		LogOutput: "stderr",
	}
}

func NewRunCommand() *cobra.Command {
	cfg := NewDefaultConfig()
	cmd := &cobra.Command{
		Use:   "run [flags]",
		Short: "start amesh",
		Run: func(cmd *cobra.Command, args []string) {
			logger, err := log.NewLogger(
				log.WithLogLevel(cfg.LogLevel),
				log.WithOutputFile(cfg.LogOutput),
			)
			if err != nil {
				utils.Dief("failed to set logger: %v", err.Error())
			}
			log.DefaultLogger = logger
			log.Infow("amesh started")
			defer log.Info("amesh exited")

			ctx, cancel := context.WithCancel(context.Background())
			g, err := amesh.NewAgent(ctx, cfg.XDSConfigSource, nil, nil, cfg.LogLevel, cfg.LogOutput)
			if err != nil {
				utils.Dief("failed to create generator: %v", err.Error())
			}

			go func() {
				utils.WaitForSignal(func() {
					cancel()
				})
			}()

			if err = g.Run(ctx.Done()); err != nil {
				utils.Dief("agent error: %v", err.Error())
			}
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.LogOutput, "log-output", "stderr", "the output file path of error log")
	cmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "the error log level")
	cmd.PersistentFlags().StringVar(&cfg.XDSConfigSource, "xds-config-source", "", "the xds config source address, required if provisioner is \"xds-v3-grpc\"")
	return cmd
}
