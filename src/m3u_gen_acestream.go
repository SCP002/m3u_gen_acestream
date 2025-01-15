package main

import (
	"context"
	"encoding/json"
	"m3u_gen_acestream/acestream"
	"m3u_gen_acestream/util/logger"
	"net/http"
)

func main() {
	log := logger.New(logger.DebugLevel)
	log.Info("Starting")

	httpClient := &http.Client{}
	engine := acestream.NewEngine(log, httpClient, "127.0.0.1:6878")
	engine.WaitForConnection(context.Background())

	results, err := engine.SearchAll(context.Background())
	if err != nil {
		log.Error(err)
	}
	prettyResults, _ := json.MarshalIndent(results, "", "  ")
	log.Info(string(prettyResults))
}
