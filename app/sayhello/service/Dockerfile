# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26.0-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH
ARG SERVICE_NAME=sayhello

RUN apk add --no-cache make git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /src/bin/${SERVICE_NAME} ./app/${SERVICE_NAME}/service/cmd/server

# Runtime stage
FROM alpine:3.19

ARG SERVICE_NAME=sayhello

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /src/bin/${SERVICE_NAME} /app/${SERVICE_NAME}

VOLUME /app/configs

ENV TZ=Asia/Shanghai
ENV SERVICE_NAME=${SERVICE_NAME}

CMD ["/bin/sh", "-c", "/app/${SERVICE_NAME} -conf /app/configs"]
