FROM golang:latest AS build
ADD . /src
RUN cd /src \
 && http_proxy=http://10.76.6.181:8118 https_proxy=http://10.76.6.181:8118 go build ./...

FROM alpine:latest
RUN apk update \
 && apk upgrade \
 && apk add --no-cache ca-certificates libc6-compat \
 && update-ca-certificates 2>/dev/null || true
WORKDIR /app
COPY --from=build /src/srvmon /app/
USER nobody
ENTRYPOINT ["/app/srvmon"]