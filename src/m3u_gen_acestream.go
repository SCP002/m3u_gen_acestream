package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/adampresley/sigint"
	"github.com/cockroachdb/errors"
	goFlags "github.com/jessevdk/go-flags"

	"m3u_gen_acestream/acestream"
	"m3u_gen_acestream/cli"
	"m3u_gen_acestream/config"
	"m3u_gen_acestream/m3u"
	"m3u_gen_acestream/util/logger"
)

func main() {
	log := logger.New(logger.FatalLevel)

	flags, err := cli.Parse()
	if flags.Version {
		fmt.Println("v2.0.0")
		os.Exit(0)
	}
	if cli.IsErrOfType(err, goFlags.ErrHelp) {
		// Help message will be prined by go-flags.
		os.Exit(0)
	}
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(flags.LogLevel)
	logFile, err := log.AddFileWriter(flags.LogFile)
	if err == nil {
		// Closing nil file does not panic.
		defer logFile.Close()
	} else {
		log.Error(err)
	}

	sigint.Listen(func() {
		log.Warn("SIGINT or SIGTERM signal received, shutting down")
		os.Exit(0)
	})

	log.Info("Starting")

	cfg, isNewCfg, err := config.Init(log, flags.CfgPath)
	if err != nil {
		log.Fatal(errors.Wrap(err, "Initialize config"))
	}
	if isNewCfg {
		log.InfoFi("Config is written, please verify it and start this program again", "path", flags.CfgPath)
		os.Exit(0)
	}

	httpClient := &http.Client{}
	engine := acestream.NewEngine(log, httpClient, cfg.EngineAddr)
	engine.WaitForConnection(context.Background())

	results, err := engine.SearchAll(context.Background())
	if err != nil {
		log.Error(errors.Wrap(err, "Search for available acestream channels"))
	}

	if err := m3u.Generate(log, results, cfg); err != nil {
		log.Error(errors.Wrap(err, "Generate M3U file"))
	}
}
