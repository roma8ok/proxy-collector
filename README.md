## Proxy-collector

Some services for searching, checking and saving proxies from the WEB.

### How it works?

1. Start external and proxy-collector services.
2. Get proxies in postgreSQL in table `proxies`. These proxies are fresh because they are checked every 10 minutes.

### Production

1. Start external services:

- rabbitMQ. Add this to advanced.config:

```
[
  {rabbit, [
    {consumer_timeout, undefined}
  ]}
].
```

- redis for collecting sites.
- redis for collecting proxies.
- postgreSQL (within migrations).
- grafana loki / promtail / grafana.

2. Create config file config-prod.json similar to config-develop.json.

- In `ip_api_url` you can use `https://public-ip.dev/`, `https://api64.ipify.org/` or other similar services.

3. Start services (workers number is recommended for AWS t4g.micro). It is recommended to start services at the same
   time or in reverse order.

- `go build .`
- `./proxy-collector -config=config-prod.json -service=fillSearchQueries -workers=1`
- `./proxy-collector -config=config-prod.json -service=sendSearchBodyFromDDGToQueue -workers=1`
- `./proxy-collector -config=config-prod.json -service=processSearchBodyFromDDG -workers=1`
- `./proxy-collector -config=config-prod.json -service=sendHTMLFromProxySourceToQueue -workers=12`
- `./proxy-collector -config=config-prod.json -service=processSourceHTML -workers=56`
- `./proxy-collector -config=config-prod.json -service=processRawProxy -workers=128`
- `./proxy-collector -config=config-prod.json -service=fillCheckProxiesQueue -workers=1`
- `./proxy-collector -config=config-prod.json -service=processCheckProxies -workers=64`

|Queue                   |Consumers                     |Publishers                                 |
|------------------------|------------------------------|-------------------------------------------|
|1_search_queries        |sendSearchBodyFromDDGToQueue  |fillSearchQueries                          |
|2_search_bodies_from_DDG|processSearchBodyFromDDG      |sendSearchBodyFromDDGToQueue               |
|3_proxy_sources         |sendHTMLFromProxySourceToQueue|processSearchBodyFromDDG, processSourceHTML|
|4_proxy_source_html     |processSourceHTML             |sendHTMLFromProxySourceToQueue             |
|5_raw_proxies           |processRawProxy               |processSourceHTML                          |
|6_check_proxies         |processCheckProxies           |fillCheckProxiesQueue                      |

### Local development

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
- grafana loki / promtail / grafana:
    1. `docker run --name proxy-collector-loki -d -v $(pwd):/mnt/config -p 3100:3100 grafana/loki:2.2.1 -config.file=/mnt/config/config-loki.yaml`
    2. `docker run --name proxy-collector-promtail -d -v $(pwd):/mnt/config -v /var/log:/var/log grafana/promtail:2.2.1 -config.file=/mnt/config/config-promtail.yaml`
    3. `docker run --name proxy-collector-grafana -d -p 3000:3000 grafana/grafana:latest`
    4. go to http://localhost:3000 (login: `admin`, password: `admin`) and add data source -> Loki -> set url
       to http://localhost:3100 (on macOS - http://docker.for.mac.localhost:3100)

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
