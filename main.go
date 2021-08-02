package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

func main() {
	services := []string{
		serviceFillSearchQueries,
		serviceSendSearchBodyFromDDGToQueue,
		serviceProcessSearchBodyFromDDG,
		serviceSendHTMLFromProxySourceToQueue,
		serviceProcessSourceHTML,
		serviceProcessRawProxy,
		serviceFillCheckProxiesQueue,
		serviceProcessCheckProxies,
	}

	configFlag := flag.String("config", "", "Select config .json file.")
	serviceFlag := flag.String("service", "", fmt.Sprintf("Select one service for activate.\nAvailable: %s.", strings.Join(services, ", ")))
	flag.Parse()
	if *serviceFlag == "" || !isExist(*serviceFlag, services) {
		_, _ = fmt.Fprintf(os.Stderr, "Select one service for activate (flag -service)\n")
		os.Exit(1)
	}
	if *configFlag == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Select config file (flag -config)\n")
		os.Exit(1)
	}

	confData, err := ioutil.ReadFile(*configFlag)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Can't read config file\n")
		os.Exit(1)
	}
	var conf Config
	if err := json.Unmarshal(confData, &conf); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Config file is wrong\n")
		os.Exit(1)
	}

	loki, err := newLogger(conf.PromtailURL, createJobName(*serviceFlag))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Can't create logger\n")
		os.Exit(1)
	}

	rabbitConn, err := amqp.Dial(conf.RabbitURL)
	if err != nil {
		loki.errorWithExit(errors.WithMessage(errors.WithStack(err), "Can't connect to rabbitMQ"))
	}
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			loki.errorWithExit(errors.WithMessage(errors.WithStack(err), "Can't close connection to rabbitMQ"))
		}
	}(rabbitConn)

	queues := []string{queueSearchQueries, queueSearchBodiesFromDDG, queueProxySources, queueProxySourceHTML, queueRawProxies, queueCheckProxies}
	if err := initQueues(rabbitConn, queues); err != nil {
		loki.errorWithExit(errors.WithMessage(errors.WithStack(err), "Can't init queues"))
	}

	rdbForSites, err := newRedisDB(conf.RedisURLForSites)
	if err != nil {
		loki.errorWithExit(errors.WithMessage(errors.WithStack(err), "Can't connect to redis for site urls"))
	}
	rdbForProxies, err := newRedisDB(conf.RedisURLForProxies)
	if err != nil {
		loki.errorWithExit(errors.WithMessage(errors.WithStack(err), "Can't connect to redis for proxy urls"))
	}

	postgresPool, err := pgxpool.Connect(context.Background(), conf.PostgresURL)
	if err != nil {
		loki.errorWithExit(errors.WithMessage(errors.WithStack(err), "Can't connect to postgreSQL"))
	}
	defer postgresPool.Close()

	app := App{
		loki:          loki,
		rabbitConn:    rabbitConn,
		postgresPool:  postgresPool,
		rdbForSites:   rdbForSites,
		rdbForProxies: rdbForProxies,
		conf:          conf,
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	if *serviceFlag == serviceFillSearchQueries {
		loki.info("worker started")
		go func() {
			fromProxies := false
			for {
				if err := fillSearchQueries(app, fromProxies); err != nil {
					loki.errorWithExit(errors.WithStack(err))
				}
				fromProxies = !fromProxies
				time.Sleep(time.Minute)
			}
		}()
	}

	if *serviceFlag == serviceSendSearchBodyFromDDGToQueue {
		loki.info("worker started")
		go func() {
			if err := sendSearchBodyFromDDGToQueue(app, proxyURL, userAgent); err != nil {
				loki.errorWithExit(errors.WithStack(err))
			}
		}()
	}

	if *serviceFlag == serviceProcessSearchBodyFromDDG {
		loki.info("worker started")
		go func() {
			if err := processSearchBodyFromDDG(app); err != nil {
				loki.errorWithExit(errors.WithStack(err))
			}
		}()
	}

	if *serviceFlag == serviceSendHTMLFromProxySourceToQueue {
		loki.info("worker started")
		go func() {
			if err := sendHTMLFromProxySourceToQueue(app); err != nil {
				loki.errorWithExit(errors.WithStack(err))
			}
		}()
	}

	if *serviceFlag == serviceProcessSourceHTML {
		loki.info("worker started")
		go func() {
			if err := processSourceHTML(app); err != nil {
				loki.errorWithExit(errors.WithStack(err))
			}
		}()
	}

	if *serviceFlag == serviceProcessRawProxy {
		loki.info("worker started")
		go func() {
			if err := processRawProxy(app); err != nil {
				loki.errorWithExit(errors.WithStack(err))
			}
		}()
	}

	if *serviceFlag == serviceFillCheckProxiesQueue {
		loki.info("worker started")
		go func() {
			for {
				if err := fillCheckProxiesQueue(app); err != nil {
					loki.errorWithExit(errors.WithStack(err))
				}
				time.Sleep(time.Minute * 10)
			}
		}()
	}

	if *serviceFlag == serviceProcessCheckProxies {
		loki.info("worker started")
		go func() {
			if err := processCheckProxy(app); err != nil {
				loki.errorWithExit(errors.WithStack(err))
			}
		}()
	}

	<-exit
}
