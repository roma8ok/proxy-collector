## Proxy-collector

Some services for searching, checking and saving proxies from the WEB. Works independently.

### Development

1. Start external services:

- rabbitMQ:
  `docker run --name proxy-collector-rabbit -d -p 15672:15672 -p 5672:5672 -e RABBITMQ_DEFAULT_USER=admin -e RABBITMQ_DEFAULT_PASS=admin -e RABBITMQ_SERVER_ADDITIONAL_ERL_ARGS="-rabbit consumer_timeout false" rabbitmq:3-management`
- redis for collecting sites:
  `docker run --name proxy-collector-redis-sites -d -p 6379:6379 redis:6`
- redis for collecting proxies:
  `docker run --name proxy-collector-redis-proxies -d -p 6380:6379 redis:6`
- postgreSQL:
  `docker run --name proxy-collector-postgres -d -p 5432:5432 -v ${PWD}/migrations:/migrations -e POSTGRES_USER=admin -e POSTGRES_PASSWORD=admin postgres:13`
- execute migrations:
  `for f in migrations/*.sql; do; docker exec -t proxy-collector-postgres psql -U admin -d admin -f "/$f"; done`

2. Start services (it is necessary to start at least one instance of each service for correct work):

- `go build .`
- `./proxy-collector -config=develop.json -service=fillSearchQueries`
- `./proxy-collector -config=develop.json -service=sendSearchBodyFromDDGToQueue`
- `./proxy-collector -config=develop.json -service=processSearchBodyFromDDG`
- `./proxy-collector -config=develop.json -service=sendHTMLFromProxySourceToQueue`
- `./proxy-collector -config=develop.json -service=processSourceHTML`
- `./proxy-collector -config=develop.json -service=processRawProxy`
- `./proxy-collector -config=develop.json -service=fillCheckProxiesQueue`
- `./proxy-collector -config=develop.json -service=processCheckProxies`

It is possible to start all or part of the services with one
command: `./proxy-collector -config=develop.json -service=fillSearchQueries -service=sendSearchBodyFromDDGToQueue -service=processSearchBodyFromDDG -service=sendHTMLFromProxySourceToQueue -service=processSourceHTML -service=processRawProxy -service=fillCheckProxiesQueue -service=processCheckProxies`

3. Show rabbitMQ management: `http://localhost:15672/`. Login: `admin`, password: `admin`.

4. Show processed proxies - table `proxies` in postgreSQL. User: `admin`, password: `admin`, database: `admin`.
