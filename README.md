Start rabbitmq:
`docker run --name proxy-collector-rabbit -d -p 15672:15672 -p 5672:5672 -e RABBITMQ_DEFAULT_USER=admin -e RABBITMQ_DEFAULT_PASS=admin -e RABBITMQ_SERVER_ADDITIONAL_ERL_ARGS="-rabbit consumer_timeout false" rabbitmq:3-management`

Show rabbitmq management: `http://localhost:15672/`

Start redis for collecting sites:
`docker run --name proxy-collector-redis-sites -d -p 6379:6379 redis:6`

Start redis for collecting proxies:
`docker run --name proxy-collector-redis-proxies -d -p 6380:6379 redis:6`

Show all keys and values in redis: `for i in $(redis-cli KEYS '*'); do echo $i; redis-cli GET $i; done`

Start postgres:
`docker run --name proxy-collector-postgres -d -p 5432:5432 -e POSTGRES_USER=admin -e POSTGRES_PASSWORD=admin postgres:13`
