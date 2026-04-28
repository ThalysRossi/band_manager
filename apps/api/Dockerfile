FROM golang:1.23-alpine AS build

WORKDIR /src

COPY apps/api/go.mod ./apps/api/go.mod
WORKDIR /src/apps/api
RUN go mod download

WORKDIR /src
COPY apps/api ./apps/api
WORKDIR /src/apps/api
RUN go build -o /bin/band-manager-api ./cmd/api

FROM alpine:3.21

RUN adduser -D -H -u 10001 appuser
USER appuser

COPY --from=build /bin/band-manager-api /bin/band-manager-api

EXPOSE 8080

ENTRYPOINT ["/bin/band-manager-api"]
