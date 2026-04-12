# Neo Collector Go

Automacao em Go para consultar Prometheus periodicamente e criar ou atualizar nodes e relacionamentos no Neo4j a partir de um YAML configuravel.

## O que o MVP entrega

- leitura de configuracao operacional via `.env`
- leitura de regras de coleta via YAML
- queries periodicas em um ou mais targets Prometheus
- processamento isolado de cada datapoint
- limite configuravel de concorrencia no processamento de datapoints
- criacao e atualizacao de nodes e relacionamentos no Neo4j
- suporte a `update_policy` `create` e `merge`
- modo `dry_run` por target
- logs em `text` por default (ou `json` via `APP_LOG_FORMAT`)

## Estrutura principal

- `cmd/neo-collector`: ponto de entrada
- `internal/config`: carga e validacao de `.env` e YAML
- `internal/collector/prometheus`: cliente HTTP do Prometheus
- `internal/engine`: avaliacao de condicoes, resolucao de propriedades e planejamento de mutacoes
- `internal/repository/neo4j`: persistencia no Neo4j
- `internal/scheduler`: agendamento por intervalo
- `configs/config.demo.yaml`: configuracao canonica de exemplo
- `docs/`: arquitetura, backlog e referencia de configuracao

## Execucao local

1. Copie `.env.example` para `.env` e ajuste as credenciais do Neo4j.
2. Ajuste `APP_MAX_DATAPOINT_WORKERS` se quiser controlar o paralelismo de processamento.
3. Ajuste `configs/config.demo.yaml` para o seu Prometheus e suas definicoes.
4. Execute:

```bash
make run-once
```

Para rodar continuamente:

```bash
make run
```

## Testes

Testes unitarios:

```bash
make test
```

Testes de integracao com Prometheus e Neo4j reais em containers:

```bash
make test-integration
```

As imagens podem ser sobrescritas via:

- `INTEGRATION_PROMETHEUS_IMAGE`
- `INTEGRATION_NEO4J_IMAGE`
- `DOCKER_HOST`
- `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE`

## Flags

- `-env`: caminho para o arquivo `.env`
- `-config`: caminho para o arquivo YAML
- `-once`: executa cada job uma unica vez e encerra

## Observacao sobre os arquivos de exemplo

O arquivo `config.demo.yaml` na raiz foi mantido como referencia original do PRD. O arquivo canonico para execucao do MVP e `configs/config.demo.yaml`.

## Documentacao

- [Arquitetura](./docs/architecture.md)
- [Quebra de tarefas](./docs/tasks-breakdown.md)
- [Backlog de execucao](./docs/backlog-execucao.md)
- [Build e execucao](./docs/build-and-run.md)
- [Referencia de configuracao](./docs/config-reference.md)
- [Troubleshooting](./docs/troubleshooting.md)
