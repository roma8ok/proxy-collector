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
)

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

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
	serviceFlag := flag.String("service", "", fmt.Sprintf("Select single service for activate.\nAvailable: %s.", strings.Join(services, ", ")))
	workersFlag := flag.Int("workers", 1, "Select number of running workers of the service")
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

	hostExtIP, err := externalIP(conf.IPAPIURL)
	if err != nil {
		loki.errorWithExit(errors.WithMessage(errors.WithStack(err), fmt.Sprintf(`Can't get host external IP from "%s"`, conf.IPAPIURL)))
	}

	app := App{
		loki:          loki,
		postgresPool:  postgresPool,
		rdbForSites:   rdbForSites,
		rdbForProxies: rdbForProxies,
		conf:          conf,
		hostExtIP:     hostExtIP,
	}

	for i := 0; i < *workersFlag; i++ {
		loki.info(fmt.Sprintf("Service started (worker %d)", i))

		if *serviceFlag == serviceFillSearchQueries {
			fromProxies := false
			queueDest := NewRabbitMQSession(app.loki, queueSearchQueries, app.conf.RabbitURL)

			go func() {
				for {
					if err := fillSearchQueries(app, queueDest, fromProxies); err != nil {
						loki.error(errors.WithStack(err))
					}
					fromProxies = !fromProxies
					time.Sleep(time.Minute)
				}
			}()
		}

		if *serviceFlag == serviceSendSearchBodyFromDDGToQueue {
			queueSource := NewRabbitMQSession(app.loki, queueSearchQueries, app.conf.RabbitURL)
			queueDest := NewRabbitMQSession(app.loki, queueSearchBodiesFromDDG, app.conf.RabbitURL)
			go sendSearchBodyFromDDGToQueue(app, queueSource, queueDest, proxyURL, userAgent)
		}

		if *serviceFlag == serviceProcessSearchBodyFromDDG {
			queueSource := NewRabbitMQSession(app.loki, queueSearchBodiesFromDDG, app.conf.RabbitURL)
			queueDest := NewRabbitMQSession(app.loki, queueProxySources, app.conf.RabbitURL)
			go processSearchBodyFromDDG(app, queueSource, queueDest)
		}

		if *serviceFlag == serviceSendHTMLFromProxySourceToQueue {
			queueSource := NewRabbitMQSession(app.loki, queueProxySources, app.conf.RabbitURL)
			queueDest := NewRabbitMQSession(app.loki, queueProxySourceHTML, app.conf.RabbitURL)
			go sendHTMLFromProxySourceToQueue(app, queueSource, queueDest)
		}

		if *serviceFlag == serviceProcessSourceHTML {
			queueSource := NewRabbitMQSession(app.loki, queueProxySourceHTML, app.conf.RabbitURL)
			chDestRawProxies := NewRabbitMQSession(app.loki, queueRawProxies, app.conf.RabbitURL)
			chDestProxySources := NewRabbitMQSession(app.loki, queueProxySources, app.conf.RabbitURL)
			go processSourceHTML(app, queueSource, chDestRawProxies, chDestProxySources)
		}

		if *serviceFlag == serviceProcessRawProxy {
			queueSource := NewRabbitMQSession(app.loki, queueRawProxies, app.conf.RabbitURL)
			go processRawProxy(app, queueSource)
		}

		if *serviceFlag == serviceFillCheckProxiesQueue {
			queueDest := NewRabbitMQSession(app.loki, queueCheckProxies, app.conf.RabbitURL)
			go func() {
				for {
					if err := fillCheckProxiesQueue(app, queueDest); err != nil {
						loki.error(errors.WithStack(err))
					}
					time.Sleep(10 * time.Minute)
				}
			}()
		}

		if *serviceFlag == serviceProcessCheckProxies {
			queueSource := NewRabbitMQSession(app.loki, queueCheckProxies, app.conf.RabbitURL)
			go processCheckProxy(app, queueSource)
		}
	}

	if shutdown := <-exit; shutdown != nil {
		app.loki.info("Service shutdown")
		time.Sleep(lokiBatchWait + time.Second*2)
		os.Exit(1)
	}
}
