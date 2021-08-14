## 0.3.2 - in progress

- changed the recommended number of workers

## 0.3.1 - 2021.08.13

- updated findProxySourcesFromDDG finish logger info
- added test for convertBytesToStringSlice
- added makeDDGSearchURL description
- added tests for sendGetRequest
- added tests for findSiteURLsFromDDG
- fix typo
- readme updated

## 0.3.0 - 2021.08.12

- refactored queues

## 0.2.2 (release)  - 2021.08.12

- updated tests for set
- updated tests for isExist
- added test for randomElementFromSlice
- added tests for urlsHaveSameDomain
- added link to proxy-list in readme
- added tests for formatDuration
- added test for fillString

## 0.2.1 - 2021.08.12

- removed unused postgres type

## 0.2.0 - 2021.08.10

- added advanced config to rabbitMQ
- updated service exit channel
- ip_api_url changed
- updated checkProxy

## 0.1.1 (release) - 2021.08.06

- updated readme
- added comments
- refactored

### 0.1.0 - 2021.08.04

- added error logs
- changed queues durable to true
- added reconnect to rabbitMQ
- updated readme

### 0.0.13 - 2021.08.02

- added logger

### 0.0.12 - 2021.08.01

- added config file
- updated readme
- added service to fill search queries queue
- added service to fill check proxies queue
- added service to process check proxy

### 0.0.11 - 2021.07.28

- added postgres
- added started postgres migrations
- saved processed proxy to postgres

### 0.0.10 - 2021.07.26

- added proxies checker
- added service to process raw proxies

### 0.0.9 - 2021.07.26

- added redis for collecting proxies
- deleted consumer_timeout from rabbit config
- send FromURL with html at some services

### 0.0.8 - 2021.07.23

- added flag to start microservices
- renamed microservices to services

### 0.0.7 - 2021.07.17

- added find proxies from html table
- find urls from html and add to queue

### 0.0.6 - 2021.07.16

- added findProxiesFromHTML
- added microservice to find proxies from html and send to queue

### 0.0.5 - 2021.07.16

- added microservice to get HTML from proxy source and send to queue
- added simple logger
- refactored sendRequest, publish

### 0.0.4 - 2021.07.15

- added redis and proxy source cache

### 0.0.3 - 2021.07.15

- added microservice to process body from duckduckgo search response and send found urls to queue

### 0.0.2 - 2021.07.14

- queue added
- duckduckgo search added
- added microservice to get body from duckduckgo search response and send to queue

### 0.0.1 - 2021.07.10

- project started
