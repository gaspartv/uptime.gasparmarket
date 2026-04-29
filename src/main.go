package main

import (
	"github.com/gaspartv/uptime.gasparmarket/src/config"
	"github.com/gaspartv/uptime.gasparmarket/src/internal/router"
)

func main() {
	env, err := config.ValidateEnv()
	if err != nil {
		panic(err)
	}

	router.InitializeRoutes(env)
}
