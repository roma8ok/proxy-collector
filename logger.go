package main

import (
	"fmt"
	"os"
	"time"

	"github.com/afiskon/promtail-client/promtail"
)

type Logger struct {
	client  promtail.Client
	jobName string
}

func newLogger(promtailURL, jobName string) (Logger, error) {
	sourceName := "proxy-collector"
	labels := "{source=\"" + sourceName + "\",job=\"" + jobName + "\"}"
	conf := promtail.ClientConfig{
		PushURL:            promtailURL,
		Labels:             labels,
		BatchWait:          lokiBatchWait,
		BatchEntriesNumber: 10000,
		SendLevel:          promtail.DEBUG,
		PrintLevel:         promtail.DEBUG,
	}

	client, err := promtail.NewClientJson(conf)
	if err != nil {
		return Logger{}, err
	}
	return Logger{
		client:  client,
		jobName: jobName,
	}, nil
}

func (l *Logger) info(message string) {
	l.client.Infof("%s - %s", l.jobName, message)
}

func (l *Logger) error(err error) {
	l.client.Errorf("%s - %s", l.jobName, err)
}

func (l *Logger) errorWithExit(err error) {
	l.error(err)
	time.Sleep(lokiBatchWait + time.Second*2)
	os.Exit(1)
}

func createJobName(serviceName string) (jobName string) {
	hostname, err := os.Hostname()
	if err == nil {
		jobName += fillString(hostname, 18, " ") + "|"
	}

	jobName += fmt.Sprintf("%d|%s", time.Now().Unix(), fillString(serviceName, 24, " "))
	return
}
