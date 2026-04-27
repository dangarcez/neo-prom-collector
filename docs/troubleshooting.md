# Troubleshooting

## O binario `go` nao e encontrado

Verifique se o Go esta no `PATH`.

Exemplo:

```bash
export PATH=/usr/local/go/bin:$PATH
go version
```

## `make test-integration` falha sem conseguir falar com Docker

Os testes de integracao dependem de um daemon Docker acessivel pelo usuario atual.

Verifique:

- se o Docker esta em execucao
- se o usuario tem permissao para acessar o socket Docker
- se a variavel `DOCKER_HOST` nao esta apontando para um endpoint invalido

Em ambientes onde `XDG_RUNTIME_DIR` faz o `testcontainers-go` escolher um socket rootless inexistente, force o socket padrao:

```bash
DOCKER_HOST=unix:///var/run/docker.sock \
TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock \
make test-integration
```

## `make test-integration` falha ao baixar imagens

Isso normalmente indica:

- falta de acesso a internet
- rate limit no registry
- nome ou tag de imagem invalido

As imagens podem ser sobrescritas com:

- `INTEGRATION_PROMETHEUS_IMAGE`
- `INTEGRATION_NEO4J_IMAGE`

## O Prometheus responde, mas a query volta vazia

Nos testes de integracao a query usada e `prometheus_build_info`.

Se o Prometheus acabou de subir, pode levar alguns segundos ate haver datapoints disponiveis. O teste ja faz retry, mas em ambientes lentos pode ser necessario aumentar timeout futuramente.

## O Neo4j sobe, mas a aplicacao nao conecta

Verifique:

- URI Bolt correta
- usuario e senha corretos
- database configurado
- `NEO4J_VERIFY_CONNECTIVITY=true` falhando por indisponibilidade temporaria

## Duplicacao inesperada de nodes ou relacionamentos

Revise:

- `update_policy`
- criterio de identidade por `type` e `name` nos nodes
- `z4j_template_hash` do relacionamento persistido
- `z4j_node_uid`, `z4j_rel_uid` e `z4j_origin` nos registros gerados pelo app
- match de `source` e `target`
- selectors muito amplos podem gerar fan-out e criar muitos relacionamentos

## Logs ajudam a diagnosticar

Defina:

```bash
APP_LOG_LEVEL=debug
```

Isso aumenta a visibilidade das execucoes e facilita localizar falhas por target, job e fase de processamento.
