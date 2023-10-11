package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"clipboard-remote/clipboard"
	"clipboard-remote/utils"

	log "github.com/sirupsen/logrus"
)

var (
	configPath = flag.String("config", "client.yaml", "client config file path")
)

func init() {
	// Set the log output to stdout
	log.SetReportCaller(true)
	log.SetFormatter(&utils.Formatter{
		HideKeys:    true,
		CallerFirst: true,
		NoColors:    true,
	})

	// Set the log output to the specified file
	log.SetOutput(os.Stdout)

	// Set the log level
	log.SetLevel(log.InfoLevel)
}

func main() {
	flag.Parse()

	clientConfig, err := utils.ClientConfigRead(*configPath)
	if err != nil {
		log.Errorf("Failed to load client config file(%s), err: %v.", *configPath, err)
		return
	}

	// add interrupt sigal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// watch context, used for watch break
	ctx, cancel := context.WithCancel(context.Background())

	// handle io local to server
	client := NewClient(clientConfig)
	go client.handleIO(ctx, clipboard.Watch(ctx))

	<-interrupt
	cancel()
	log.Infoln("Interrupt manually by user.")

	client.close()
}
