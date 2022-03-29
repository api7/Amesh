package main

import "C"

import (
	"context"
	"github.com/api7/amesh/pkg/amesh"
	"github.com/api7/amesh/pkg/utils"
)

func main() {
}

//export StartAmesh
func StartAmesh(src string) {
	ctx := context.Background()
	agent, err := amesh.NewAgent(ctx, src, nil, "debug", "stderr")
	if err != nil {
		utils.Dief("failed to create generator: %v", err.Error())
	}

	if err = agent.Run(ctx.Done()); err != nil {
		utils.Dief("agent error: %v", err.Error())
	}
}
