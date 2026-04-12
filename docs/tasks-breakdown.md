# Quebra do Projeto em Tarefas

## 1. Tarefa

Inicializar o projeto Go

## 2. Descricao

Criar o modulo, ponto de entrada e a estrutura base de diretorios prevista na arquitetura.

## 3. Arquivos envolvidos

- `go.mod`
- `cmd/neo-collector/main.go`
- `internal/`
- `docs/`
- `tests/`

## 4. Dependencias

- nenhuma

---

## 1. Tarefa

Implementar bootstrap da aplicacao

## 2. Descricao

Criar a inicializacao central da app, incluindo ciclo de vida, carregamento dos componentes e shutdown gracioso.

## 3. Arquivos envolvidos

- `internal/app/bootstrap.go`
- `internal/app/runtime.go`
- `cmd/neo-collector/main.go`

## 4. Dependencias

- Inicializar o projeto Go

---

## 1. Tarefa

Definir modelos de configuracao

## 2. Descricao

Criar structs canonicas para `.env` e YAML, normalizando diferencas como `type` ou `types`, `template_hash` ou `template_hashes` e aliases reservados.

## 3. Arquivos envolvidos

- `internal/config/model.go`
- `internal/config/env.go`
- `internal/config/yaml.go`

## 4. Dependencias

- Inicializar o projeto Go

---

## 1. Tarefa

Implementar carregamento de `.env` e YAML

## 2. Descricao

Ler variaveis de ambiente, arquivo YAML e montar uma configuracao interna unica para o runtime.

## 3. Arquivos envolvidos

- `internal/config/env.go`
- `internal/config/yaml.go`
- `configs/config.demo.yaml`
- `.env.example`

## 4. Dependencias

- Definir modelos de configuracao

---

## 1. Tarefa

Implementar validacao de configuracao

## 2. Descricao

Validar obrigatoriedades, operadores aceitos, presenca de `name`, selectors de relacionamentos e inconsistencias do YAML antes da aplicacao subir.

## 3. Arquivos envolvidos

- `internal/config/validate.go`
- `internal/config/model.go`

## 4. Dependencias

- Implementar carregamento de `.env` e YAML

---

## 1. Tarefa

Definir modelos de dominio

## 2. Descricao

Criar tipos de dominio para datapoint, node, relacionamento, selectors, politicas de update e erros de negocio.

## 3. Arquivos envolvidos

- `internal/domain/datapoint.go`
- `internal/domain/graph.go`
- `internal/domain/policy.go`
- `internal/domain/errors.go`

## 4. Dependencias

- Inicializar o projeto Go

---

## 1. Tarefa

Implementar cliente Prometheus

## 2. Descricao

Criar client HTTP para Prometheus com timeout, TLS, query instantanea e conversao da resposta para `Datapoint`.

## 3. Arquivos envolvidos

- `internal/collector/prometheus/client.go`
- `internal/collector/prometheus/query.go`
- `internal/domain/datapoint.go`

## 4. Dependencias

- Implementar carregamento de `.env` e YAML
- Definir modelos de dominio

---

## 1. Tarefa

Implementar cliente Neo4j

## 2. Descricao

Criar camada de conexao com Neo4j, sessao, transacao e verificacao opcional de conectividade no startup.

## 3. Arquivos envolvidos

- `internal/repository/neo4j/client.go`
- `internal/repository/neo4j/tx.go`

## 4. Dependencias

- Implementar carregamento de `.env` e YAML
- Definir modelos de dominio

---

## 1. Tarefa

Implementar gerador de identidades estaveis

## 2. Descricao

Gerar `node_uid` e `rel_uid` deterministicos com UUIDv5 a partir das chaves canonicas de negocio.

## 3. Arquivos envolvidos

- `internal/engine/identity.go`
- `internal/domain/graph.go`

## 4. Dependencias

- Definir modelos de dominio

---

## 1. Tarefa

Implementar avaliacao de condicoes

## 2. Descricao

Suportar condicoes por label e por value, com operadores `equals`, `not_equals`, `greater_than` e `less_than`.

## 3. Arquivos envolvidos

- `internal/engine/conditions.go`
- `internal/domain/datapoint.go`
- `internal/config/model.go`

## 4. Dependencias

- Implementar validacao de configuracao
- Definir modelos de dominio

---

## 1. Tarefa

Implementar resolucao de propriedades

## 2. Descricao

Resolver propriedades estaticas, dinamicas e condicionais para nodes e relacionamentos, incluindo `__value__` e `__timestamp__`.

## 3. Arquivos envolvidos

- `internal/engine/properties.go`
- `internal/domain/graph.go`
- `internal/domain/datapoint.go`

## 4. Dependencias

- Definir modelos de dominio
- Implementar avaliacao de condicoes

---

## 1. Tarefa

Implementar planner de mutacoes

## 2. Descricao

Transformar cada datapoint em um plano de persistencia contendo nodes elegiveis, relacionamentos elegiveis e campos automaticos.

## 3. Arquivos envolvidos

- `internal/engine/planner.go`
- `internal/engine/processor.go`
- `internal/engine/identity.go`
- `internal/engine/properties.go`

## 4. Dependencias

- Implementar gerador de identidades estaveis
- Implementar resolucao de propriedades

---

## 1. Tarefa

Implementar persistencia de nodes

## 2. Descricao

Criar a logica de `create` e `merge` para nodes, respeitando `Entity`, `name`, `template_hashes`, `origin`, `created_at` e `updated_at`.

## 3. Arquivos envolvidos

- `internal/repository/neo4j/nodes.go`
- `internal/repository/neo4j/tx.go`

## 4. Dependencias

- Implementar cliente Neo4j
- Implementar planner de mutacoes

---

## 1. Tarefa

Implementar persistencia de relacionamentos

## 2. Descricao

Criar a logica de `create` e `merge` para relacionamentos sem criar nodes, com match explicito de source e target.

## 3. Arquivos envolvidos

- `internal/repository/neo4j/relationships.go`
- `internal/repository/neo4j/tx.go`

## 4. Dependencias

- Implementar cliente Neo4j
- Implementar planner de mutacoes
- Implementar persistencia de nodes

---

## 1. Tarefa

Implementar orquestracao por datapoint

## 2. Descricao

Executar o fluxo completo por datapoint: avaliar regras, persistir nodes primeiro, depois relacionamentos, e tratar erros por item.

## 3. Arquivos envolvidos

- `internal/engine/processor.go`
- `internal/repository/neo4j/nodes.go`
- `internal/repository/neo4j/relationships.go`

## 4. Dependencias

- Implementar planner de mutacoes
- Implementar persistencia de nodes
- Implementar persistencia de relacionamentos

---

## 1. Tarefa

Implementar scheduler por job

## 2. Descricao

Criar execucao periodica por `target + job`, evitando sobreposicao e suportando cancelamento por contexto.

## 3. Arquivos envolvidos

- `internal/scheduler/scheduler.go`
- `internal/app/runtime.go`

## 4. Dependencias

- Implementar cliente Prometheus
- Implementar orquestracao por datapoint

---

## 1. Tarefa

Implementar limite de concorrencia

## 2. Descricao

Adicionar worker pool para processar datapoints com paralelismo controlado sem saturar Neo4j ou Prometheus.

## 3. Arquivos envolvidos

- `internal/scheduler/worker_pool.go`
- `internal/engine/processor.go`

## 4. Dependencias

- Implementar scheduler por job

---

## 1. Tarefa

Implementar observabilidade

## 2. Descricao

Adicionar logs estruturados, contadores de execucao, erros por fase e metricas basicas do runtime.

## 3. Arquivos envolvidos

- `internal/observability/logger.go`
- `internal/observability/metrics.go`
- `internal/app/runtime.go`

## 4. Dependencias

- Implementar bootstrap da aplicacao
- Implementar scheduler por job

---

## 1. Tarefa

Preparar execucao local

## 2. Descricao

Criar arquivos auxiliares para desenvolvimento local, incluindo exemplos de ambiente e comandos de build e run.

## 3. Arquivos envolvidos

- `.env.example`
- `Makefile`
- `README.md`

## 4. Dependencias

- Implementar bootstrap da aplicacao
- Implementar carregamento de `.env` e YAML
- Implementar scheduler por job

---

## 1. Tarefa

Preparar containerizacao

## 2. Descricao

Criar `Dockerfile` e instrucoes para empacotar e executar a aplicacao em container.

## 3. Arquivos envolvidos

- `Dockerfile`
- `README.md`
- `docs/build-and-run.md`

## 4. Dependencias

- Preparar execucao local

---

## 1. Tarefa

Escrever testes unitarios

## 2. Descricao

Cobrir validacao de config, condicoes, resolucao de propriedades, identidade estavel e planner.

## 3. Arquivos envolvidos

- `tests/fixtures/`
- arquivos `*_test.go` em `internal/config/`
- arquivos `*_test.go` em `internal/engine/`

## 4. Dependencias

- Implementar validacao de configuracao
- Implementar gerador de identidades estaveis
- Implementar avaliacao de condicoes
- Implementar resolucao de propriedades
- Implementar planner de mutacoes

---

## 1. Tarefa

Escrever testes de integracao

## 2. Descricao

Validar consultas reais ao Prometheus e persistencia real no Neo4j com containers de teste.

## 3. Arquivos envolvidos

- `tests/integration/`
- arquivos `*_test.go`
- setup de containers de teste

## 4. Dependencias

- Implementar cliente Prometheus
- Implementar persistencia de nodes
- Implementar persistencia de relacionamentos
- Implementar scheduler por job
- Preparar containerizacao

---

## 1. Tarefa

Finalizar documentacao funcional

## 2. Descricao

Documentar arquitetura, formato do YAML, build, execucao local, deploy e estrategia de testes.

## 3. Arquivos envolvidos

- `docs/architecture.md`
- `docs/config-reference.md`
- `docs/build-and-run.md`
- `README.md`

## 4. Dependencias

- Preparar execucao local
- Preparar containerizacao
- Escrever testes unitarios
- Escrever testes de integracao
