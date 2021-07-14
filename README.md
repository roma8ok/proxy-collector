Start rabbitmq:
`docker run --name proxy-collector-rabbit -d -p 15672:15672 -p 5672:5672 -e RABBITMQ_DEFAULT_USER=admin -e RABBITMQ_DEFAULT_PASS=admin rabbitmq:3-management`

Show rabbitmq management: `http://localhost:15672/`
