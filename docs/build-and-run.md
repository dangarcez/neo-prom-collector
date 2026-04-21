# Build e Execucao

## Requisitos

- Go 1.25 ou superior
- acesso a um Prometheus
- acesso a um Neo4j

## Preparar ambiente local

1. Copie `.env.example` para `.env`.
2. Ajuste as credenciais do Neo4j no `.env`.
3. Ajuste `APP_MAX_DATAPOINT_WORKERS` se quiser controlar o paralelismo.
4. Ajuste os targets e jobs em `configs/config.demo.yaml`.

## Rodar localmente uma vez

```bash
make run-once
```

Comando equivalente:

```bash
go run ./cmd/neo-collector -env .env -config configs/config.demo.yaml -once
```

## Rodar localmente em loop

```bash
make run
```

Comando equivalente:

```bash
go run ./cmd/neo-collector -env .env -config configs/config.demo.yaml
```

## Gerar binario

```bash
make build
```

## Rodar testes

```bash
make test
```

## Rodar testes de integracao

Os testes de integracao sobem um Prometheus e um Neo4j reais em containers Docker.

```bash
make test-integration
```

Se quiser trocar as imagens usadas nos testes:

```bash
INTEGRATION_PROMETHEUS_IMAGE=prom/prometheus:v2.49.1 \
INTEGRATION_NEO4J_IMAGE=neo4j:5-community \
make test-integration
```

Se o `testcontainers-go` escolher o socket errado do Docker, force o socket padrao:

```bash
DOCKER_HOST=unix:///var/run/docker.sock \
TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock \
make test-integration
```

## Build do container

```bash
docker build -t neo-collector-go .
```

Observacoes:

- o `Dockerfile` compila direto `./cmd/neo-collector` e nao roda `go mod download`
- isso evita baixar dependencias usadas apenas nos testes de integracao, como `testcontainers-go`
- no primeiro build, o `go build` ainda pode baixar os modulos realmente usados pelo binario

## Build com Podman

Com Podman, se o build parecer travado enquanto o Go baixa modulos, o problema normalmente nao e o projeto, e sim DNS/rede do ambiente de build. O sintoma comum e a resolucao de `proxy.golang.org` falhar ou demorar muito.

Comando recomendado:

```bash
podman build --network=host -t neo_prom_collector .
```

Se ainda falhar:

- revise a conectividade DNS do Podman
- confirme que `proxy.golang.org` e `sum.golang.org` sao acessiveis no ambiente do builder
- se estiver em ambiente corporativo, configure os proxies do Podman antes do build

## Rodar com Docker Compose

O projeto inclui um `docker-compose.yml` para subir uma stack local completa com:

- `neo4j`
- `prometheus`
- `neo-collector`

Arquivos usados nesse fluxo:

- `docker-compose.yml`
- `configs/config.compose.yaml`
- `configs/prometheus/prometheus.compose.yml`

### Preparar

1. Copie `.env.example` para `.env`.
2. Ajuste `NEO4J_PASSWORD` no `.env`.
3. Se quiser mudar a regra de coleta do compose, edite `configs/config.compose.yaml`.

### Subir a stack

```bash
docker compose up --build -d
```

### Acompanhar logs

```bash
docker compose logs -f neo-collector
```

### Endpoints uteis

- Prometheus: `http://localhost:9090`
- Neo4j Browser: `http://localhost:7474`
- Neo4j Bolt: `bolt://localhost:7687`

### Parar a stack

```bash
docker compose down
```

### Parar e remover volumes

```bash
docker compose down -v
```

Observacao:

- o collector no compose usa `configs/config.compose.yaml`
- o Prometheus do compose faz self-scrape, entao a query `prometheus_build_info` ja funciona sem configuracao extra
- `NEO4J_URI` e sobrescrito no container para `bolt://neo4j:7687`, entao o `.env` local pode continuar usando `localhost` para execucao fora do compose

## Rodar container

```bash
docker run --rm \
  --env-file .env \
  neo-collector-go \
  -config /app/configs/config.demo.yaml
```

## Deploy em container

Fluxo minimo recomendado:

1. gerar a imagem com `docker build -t neo-collector-go .`
2. montar um arquivo `.env` com credenciais do Neo4j
3. fornecer o YAML de configuracao da coleta
4. executar o container com restart policy do seu orquestrador

Exemplo simples:

```bash
docker run -d \
  --name neo-collector-go \
  --restart unless-stopped \
  --env-file .env \
  -v $(pwd)/configs:/app/configs \
  neo-collector-go \
  -config /app/configs/config.demo.yaml
```
