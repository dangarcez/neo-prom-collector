FROM docker.io/golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /out/neo-collector ./cmd/neo-collector

FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=build /out/neo-collector /app/neo-collector
COPY configs /app/configs

ENTRYPOINT ["/app/neo-collector"]
CMD ["-config", "/app/configs/config.demo.yaml"]
