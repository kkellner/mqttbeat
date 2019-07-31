# 2 stage build for minimal image size

## 1 Builder Stage
FROM golang:alpine AS builder

COPY . /go/src/github.com/tmechen/mqttbeat
RUN GOOS=linux GOARCH=amd64 && \
    apk add --no-cache g++ glide git && \
    cd /go/src/github.com/tmechen/mqttbeat && \
    glide install && \
    go build -ldflags "-linkmode external -extldflags -static" -a main.go

## 2 Running Stage
FROM scratch
COPY --from=builder /go/src/github.com/tmechen/mqttbeat/main /app/mqttbeat
WORKDIR /config
# Running with debugging output, for less logs remove "-d", "*" flags
CMD ["/app/mqttbeat", "--path.config", "/config", "-e", "-d", "*"]