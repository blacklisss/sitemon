# FROM archlinux

# WORKDIR /app

# COPY ./main ./

# CMD ["./main"]


# 1
FROM golang:latest AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o ./main ./sitemon/cmd/server

# 2

FROM scratch

WORKDIR /app

COPY --from=build /app/main /app/main
COPY --from=build /app/sitemon/cmd/server/config.yaml /app/config.yaml
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Europe/Moscow

# EXPOSE 8000

CMD ["./main"]
