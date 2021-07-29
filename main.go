package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/streadway/amqp"
)

func main() {
	var sFlags arrayFlags
	flag.Var(&sFlags, "service", fmt.Sprintf("Select one of more services for activate.\nAvailable: %s, %s, %s, %s, %s.", serviceSendSearchBodyFromDDGToQueue, serviceProcessSearchBodyFromDDG, serviceSendHTMLFromProxySourceToQueue, serviceProcessSourceHTML, serviceProcessRawProxy))
	configFlag := flag.String("config", "", "Select config .json file.")
	flag.Parse()
	if len(sFlags) == 0 {
		panic("Select one of more services for activate (flag -service)")
	}
	if *configFlag == "" {
		panic("Select config file (flag -config)")
	}

	confData, err := ioutil.ReadFile(*configFlag)
	if err != nil {
		panic("Config file is wrong")
	}
	var conf config
	if err := json.Unmarshal(confData, &conf); err != nil {
		panic("Config file is wrong")
	}

	rabbitConn, err := amqp.Dial(conf.RabbitURL)
	if err != nil {
		panic(err)
	}
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}(rabbitConn)

	queues := []string{queueSearchBodiesFromDDG, queueProxySources, queueProxySourceHTML, queueRawProxies}
	if err := initQueues(rabbitConn, queues); err != nil {
		panic(err)
	}

	rdbForSites, err := newRedisDB(conf.RedisURLForSites)
	if err != nil {
		panic(err)
	}
	rdbForProxies, err := newRedisDB(conf.RedisURLForProxies)
	if err != nil {
		panic(err)
	}

	postgresPool, err := pgxpool.Connect(context.Background(), conf.PostgresURL)
	if err != nil {
		return
	}
	defer postgresPool.Close()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	if isExist(serviceSendSearchBodyFromDDGToQueue, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceSendSearchBodyFromDDGToQueue)
		go func() {
			for {
				for _, query := range []string{"proxy list", "proxy", "proxies", "прокси", "список прокси"} {
					if err := sendSearchBodyFromDDGToQueue(rabbitConn, query, proxyURL, userAgent); err != nil {
						panic(err)
					}
					time.Sleep(time.Minute)
				}
			}
		}()
	}

	if isExist(serviceProcessSearchBodyFromDDG, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceProcessSearchBodyFromDDG)
		go func() {
			if err := processSearchBodyFromDDG(rabbitConn, rdbForSites); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceSendHTMLFromProxySourceToQueue, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceSendHTMLFromProxySourceToQueue)
		go func() {
			if err := sendHTMLFromProxySourceToQueue(rabbitConn); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceProcessSourceHTML, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceProcessSourceHTML)
		go func() {
			if err := processSourceHTML(rabbitConn, rdbForSites, rdbForProxies, postgresPool); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceProcessRawProxy, sFlags) {
		fmt.Printf("Start worker for %s\n", serviceProcessRawProxy)
		go func() {
			if err := processRawProxy(rabbitConn, postgresPool); err != nil {
				panic(err)
			}
		}()
	}

	<-exit
}
