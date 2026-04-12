package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"neo_collector_go/internal/app"
)

func main() {
	var (
		envPath    string
		configPath string
		runOnce    bool
	)

	flag.StringVar(&envPath, "env", ".env", "path to the .env file")
	flag.StringVar(&configPath, "config", "", "path to the YAML configuration file")
	flag.BoolVar(&runOnce, "once", false, "run all jobs only once and exit")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runtime, err := app.Bootstrap(ctx, app.Options{
		EnvPath:    envPath,
		ConfigPath: configPath,
		Once:       runOnce,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed: %v\n", err)
		os.Exit(1)
	}

	if err := runtime.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "runtime failed: %v\n", err)
		os.Exit(1)
	}
}
