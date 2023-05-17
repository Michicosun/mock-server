# mock-server
Simple server for REST API and message queue mocks

## Usage scope
With our service you can
- Create REST API mocks - set up route with either of three handlers:
  - __Static mocks__: request on route will respond with the predefined body
  - __Proxy mocks__: request on route will be proxied to the external service forwarding all request headers and body
  - __Dynamic mocks__: request on route will launch the predefined python script accepting request headers and body as its arguments
- Create messages queues mocks - set up broker queue mock or link two queues in an ESB pair:
  - __Rabbitmq mocks__: you can send messages to the writing end of the mocked queue and read messages from the reading one
  - __Kafka mocks__: same as previous but instead of Rabbitmq queues you are mocking the Kafka topics
  - __ESB__: you can connect two existing queues together, send messages to the first queue and read them from the second

## Interface
The service can be used through the REST API or through [mock-server-front](https://github.com/fdr896/mock-server-front) ReactJS UI

## Deploy
mock-server is implied to be deployed locally on your machine
### Prerogatives
You need to install and run [docker-daemon](https://www.docker.com/) on your computer. Also, install [golang](https://go.dev/) (in future versions, that step will be discarded)
### Linux deploy
In Linux machine the only thing you should do is to run the [deploy.sh](https://github.com/Michicosun/mock-server/blob/main/deploy/deploy.sh) script
```bash
$ git clone git@github.com:Michicosun/mock-server.git
$ cd mock-server/deploy
$ chmod +x deploy.sh
$ ./deploy.sh
```
Script sets up the `system-daemon` service `mock-server.service` and runs the `nginx` web-server

If everything went well, you'll see
```bash
Mock server successfully deployed and running. Available on http://your-hostname.domain
```
### Rest OS
Deploy script uses the `systemd` Linux service manager and Linux command line commands, so currently automatic deploy is unavailable. But it is possible to deploy the service manually
- run the message queue brokers docker containers with `$ cd deploy && docker compose up -d`
- init the golang packages executing `$ go mod init`
- run the service `$ go run cmd/server/main.go`
### Configuration
To configure the service components (e.g. adjust the broker's configuration or change the default service port) you can modify the [service config](https://github.com/Michicosun/mock-server/blob/main/configs/config.yaml)

## Service architecture
- All configs are locating in [configs/](https://github.com/Michicosun/mock-server/blob/main/configs/) dir. Adjust them for your needs
- Executable files lay in [cmd/](https://github.com/Michicosun/mock-server/blob/main/cmd/)
- [deploy/](https://github.com/Michicosun/mock-server/blob/main/deploy/) dir consists of deploy scripts and systemd, docker and nginx config files
- Service components implementation locates in [internal/](https://github.com/Michicosun/mock-server/blob/main/internal/)
  - [internal/server](https://github.com/Michicosun/mock-server/blob/main/internal/server): HTTP server implementation based on [gin web framework](https://github.com/gin-gonic/gin)
  - [internal/database](https://github.com/Michicosun/mock-server/blob/main/internal/database): [MongoDB](https://www.mongodb.com/) connect and database interface
  - [internal/brokers](https://github.com/Michicosun/mock-server/blob/main/internal/brokers): [Rabbitmq](https://www.rabbitmq.com/) and [Kafka](https://kafka.apache.org) connectors and ESB
  - [internal/coderun](https://github.com/Michicosun/mock-server/blob/main/internal/coderun): Docker workers that executes python scripts + watcher that distributes requests between theme + docker-provider that serve the docker containers
  - [internal/control](https://github.com/Michicosun/mock-server/blob/main/internal/control): Control service
  - [internal/util](https://github.com/Michicosun/mock-server/blob/main/internal/util): Support classes (e.g. blocking queue, file storage connector)

## Examples
Examples of mock-server usage can be found either in [cmd/playground](https://github.com/Michicosun/mock-server/blob/main/cmd/playground) directory or in [tests](https://github.com/Michicosun/mock-server/blob/main/tests)
