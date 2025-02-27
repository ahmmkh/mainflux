# HTTP adapter

HTTP adapter provides an HTTP API for sending messages through the platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                       | Description                                         | Default                        |
| ------------------------------ | --------------------------------------------------- | ------------------------------ |
| MF_HTTP_ADAPTER_LOG_LEVEL      | Log level for the HTTP Adapter                      | debug                          |
| MF_HTTP_ADAPTER_HOST           | HTTP adapter listening host                         |                                |
| MF_HTTP_ADAPTER_PORT           | Service HTTP port                                   | 80                             |
| MF_HTTP_ADAPTER_SERVER_CERT    | Service server certificate                          |                                |
| MF_HTTP_ADAPTER_SERVER_KEY     | Service server key                                  |                                |
| MF_THINGS_AUTH_GRPC_URL        | Things service Auth gRPC URL                        | localhost:7000                 |
| MF_THINGS_AUTH_GRPC_TIMEOUT    | Things service Auth gRPC request timeout in seconds | 1s                             |
| MF_THINGS_AUTH_GRPC_CLIENT_TLS | Flag that indicates if TLS should be turned on      | false                          |
| MF_THINGS_AUTH_GRPC_CA_CERTS   | Path to trusted CAs in PEM format                   |                                |
| MF_BROKER_URL                  | Message broker instance URL                         | nats://localhost:4222          |
| MF_JAEGER_URL                  | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY              | Send telemetry to mainflux call home server         | true                           |
| MF_HTTP_ADAPTER_INSTANCE_ID    | HTTP Adapter instance ID                            |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`http-adapter`](https://github.com/mainflux/mainflux/blob/master/docker/docker-compose.yml#L245-L262) service section in
docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the http
make http

# copy binary to bin
make install

# set the environment variables and run the service
MF_HTTP_ADAPTER_LOG_LEVEL=[HTTP Adapter Log Level] \
MF_HTTP_ADAPTER_HOST=[Service HTTP host] \
MF_HTTP_ADAPTER_PORT=[Service HTTP port] \
MF_HTTP_ADAPTER_SERVER_CERT=[Path to server certificate] \
MF_HTTP_ADAPTER_SERVER_KEY=[Path to server key] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MF_THINGS_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_THINGS_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_BROKER_URL=[Message broker instance URL] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_HTTP_ADAPTER_INSTANCE_ID=[HTTP Adapter instance ID] \
$GOBIN/mainflux-http
```

Setting `MF_HTTP_ADAPTER_CA_CERTS` expects a file in PEM format of trusted CAs. This will enable TLS against the Things gRPC endpoint trusting only those CAs that are provided.

## Usage

HTTP Authorization request header contains the credentials to authenticate a Thing. The authorization header can be a plain Thing key
or a Thing key encoded as a password for Basic Authentication. In case the Basic Authentication schema is used, the username is ignored.
For more information about service capabilities and its usage, please check out
the [API documentation](https://api.mainflux.io/?urls.primaryName=http.yml).

[doc]: https://docs.mainflux.io
