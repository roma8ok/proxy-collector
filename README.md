## Proxy-collector

Some services for searching, checking and saving proxies from the WEB.

### Development

1. Start external services:

- rabbitMQ:
  `docker run --name proxy-collector-rabbit -d -p 15672:15672 -p 5672:5672 -e RABBITMQ_DEFAULT_USER=admin -e RABBITMQ_DEFAULT_PASS=admin -e RABBITMQ_SERVER_ADDITIONAL_ERL_ARGS="-rabbit consumer_timeout false" rabbitmq:3-management`
- redis for collecting sites:
  `docker run --name proxy-collector-redis-sites -d -p 6379:6379 redis:6`
- redis for collecting proxies:
  `docker run --name proxy-collector-redis-proxies -d -p 6380:6379 redis:6`
- postgreSQL:
    1. `docker run --name proxy-collector-postgres -d -p 5432:5432 -v ${PWD}/migrations:/migrations -e POSTGRES_USER=admin -e POSTGRES_PASSWORD=admin postgres:13`
    2. execute migrations:
       `for f in migrations/*.sql; do; docker exec -t proxy-collector-postgres psql -U admin -d admin -f "/$f"; done`
- grafana loki (https://grafana.com/docs/loki/latest/installation/docker/):
    1. `docker run --name proxy-collector-loki -d -v $(pwd):/mnt/config -p 3100:3100 grafana/loki:2.2.1 -config.file=/mnt/config/config-loki.yaml`
    2. `docker run --name proxy-collector-promtail -d -v $(pwd):/mnt/config -v /var/log:/var/log grafana/promtail:2.2.1 -config.file=/mnt/config/config-promtail.yaml`
    3. `docker run --name proxy-collector-grafana -d -p 3000:3000 grafana/grafana:latest`
    4. go to http://localhost:3000 (login: `admin`, password: `admin`) and add data source -> Loki -> set url
       to http://localhost:3000 (on macos - http://docker.for.mac.localhost:3100)

2. Start services (it is necessary to start at least one instance of each service for correct work):

- `go build .`
- `./proxy-collector -config=config-develop.json -service=fillSearchQueries`
- `./proxy-collector -config=config-develop.json -service=sendSearchBodyFromDDGToQueue`
- `./proxy-collector -config=config-develop.json -service=processSearchBodyFromDDG`
- `./proxy-collector -config=config-develop.json -service=sendHTMLFromProxySourceToQueue`
- `./proxy-collector -config=config-develop.json -service=processSourceHTML`
- `./proxy-collector -config=config-develop.json -service=processRawProxy`
- `./proxy-collector -config=config-develop.json -service=fillCheckProxiesQueue`
- `./proxy-collector -config=config-develop.json -service=processCheckProxies`

3. Show rabbitMQ management: http://localhost:15672 (login: `admin`, password: `admin`).

4. Show logs at http://localhost:3000/explore (login: `admin`, password: `admin`), query - `{source="proxy-collector"}`.

5. Show processed proxies - table `proxies` in postgreSQL. User: `admin`, password: `admin`, database: `admin`.
