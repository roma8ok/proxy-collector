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
	var serviceFlags arrayFlags
	flag.Var(&serviceFlags, "service", fmt.Sprintf("Select one of more services for activate.\nAvailable: %s, %s, %s, %s, %s, %s.", serviceFillSearchQueries, serviceSendSearchBodyFromDDGToQueue, serviceProcessSearchBodyFromDDG, serviceSendHTMLFromProxySourceToQueue, serviceProcessSourceHTML, serviceProcessRawProxy))
	configFlag := flag.String("config", "", "Select config .json file.")
	flag.Parse()
	if len(serviceFlags) == 0 {
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

	queues := []string{queueSearchQueries, queueSearchBodiesFromDDG, queueProxySources, queueProxySourceHTML, queueRawProxies}
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

	if isExist(serviceFillSearchQueries, serviceFlags) {
		fmt.Printf("Start worker for %s\n", serviceFillSearchQueries)
		go func() {
			fromProxies := false
			for {
				if err := fillSearchQueries(rabbitConn, postgresPool, fromProxies); err != nil {
					panic(err)
				}
				fromProxies = !fromProxies
				time.Sleep(time.Minute)
			}
		}()
	}

	if isExist(serviceSendSearchBodyFromDDGToQueue, serviceFlags) {
		fmt.Printf("Start worker for %s\n", serviceSendSearchBodyFromDDGToQueue)
		go func() {
			if err := sendSearchBodyFromDDGToQueue(rabbitConn, proxyURL, userAgent); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceProcessSearchBodyFromDDG, serviceFlags) {
		fmt.Printf("Start worker for %s\n", serviceProcessSearchBodyFromDDG)
		go func() {
			if err := processSearchBodyFromDDG(rabbitConn, rdbForSites); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceSendHTMLFromProxySourceToQueue, serviceFlags) {
		fmt.Printf("Start worker for %s\n", serviceSendHTMLFromProxySourceToQueue)
		go func() {
			if err := sendHTMLFromProxySourceToQueue(rabbitConn); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceProcessSourceHTML, serviceFlags) {
		fmt.Printf("Start worker for %s\n", serviceProcessSourceHTML)
		go func() {
			if err := processSourceHTML(rabbitConn, rdbForSites, rdbForProxies, postgresPool); err != nil {
				panic(err)
			}
		}()
	}

	if isExist(serviceProcessRawProxy, serviceFlags) {
		fmt.Printf("Start worker for %s\n", serviceProcessRawProxy)
		go func() {
			if err := processRawProxy(rabbitConn, postgresPool); err != nil {
				panic(err)
			}
		}()
	}

	<-exit
}
