# Arquitetura do Sistema

## Objetivo

Construir um coletor em Go, executável localmente e em container, que:

1. leia configurações de infraestrutura via `.env`
2. leia regras de coleta e transformação via YAML
3. execute queries periódicas em um ou mais Prometheus
4. transforme cada datapoint em operações de criação/atualização de nodes e relacionamentos no Neo4j
5. mantenha comportamento idempotente, configurável e seguro para evolução

## Premissas e normalizações do contrato

O PRD, o guia `ingestao-automatica.md` e o `config.demo.yaml` têm pequenas inconsistências. Para a arquitetura, o contrato canônico deve ser:

- nodes usam `template_hashes` como lista de strings
- relacionamentos persistem `template_hashes` como lista de strings (config aceita `template_hash` singular)
- relacionamentos usam `rel_uid`, não `node_uid`
- todo node recebe a label base `Entity`
- `origin` deve ser sempre `"auto"`
- `created_at` e `updated_at` devem ser gravados em ISO 8601 UTC
- `update_policy` aceita `create` e `merge`, com `create` como default
- a configuração YAML deve normalizar `type` e `types` para um campo interno único
- typos do YAML de exemplo, como `greaterThen`, devem ser tratados como inválidos na validação

Essa decisão é necessária porque o guia de ingestão define os nomes exatos esperados pelo app principal.

## Visão de alto nível

```text
+-------------------+       +----------------------+       +------------------+
| .env              |       | config.yaml          |       | Prometheus N     |
| segredos/runtime  |       | targets/jobs/regras  |       | /api/v1/query    |
+---------+---------+       +----------+-----------+       +---------+--------+
          |                            |                             |
          +------------+---------------+                             |
                       v                                             |
               +---------------+                                     |
               | Bootstrap     |                                     |
               | load+validate |                                     |
               +-------+-------+                                     |
                       |                                             |
                       v                                             |
               +---------------+     agenda por job                  |
               | Scheduler     +-------------------------------------+
               +-------+-------+
                       |
                       v
               +---------------+     datapoints
               | Collector     +-------------------+
               | Prometheus    |                   |
               +-------+-------+                   v
                       |                    +-------------+
                       |                    | Rule Engine |
                       |                    | conditions  |
                       |                    | properties  |
                       |                    | ids         |
                       |                    +------+------+ 
                       |                           |
                       |                           v
                       |                    +-------------+
                       +------------------->| Neo4j Repo  |
                                            | upsert/match|
                                            +------+------+ 
                                                   |
                                                   v
                                             +-----------+
                                             | Neo4j     |
                                             +-----------+
```

## Estrutura de pastas

```text
neo_collector_go/
├── cmd/
│   └── neo-collector/
│       └── main.go
├── configs/
│   ├── config.demo.yaml
│   └── config.schema.yaml
├── docs/
│   ├── architecture.md
│   ├── build-and-run.md
│   └── config-reference.md
├── internal/
│   ├── app/
│   │   ├── bootstrap.go
│   │   └── runtime.go
│   ├── config/
│   │   ├── env.go
│   │   ├── yaml.go
│   │   ├── model.go
│   │   └── validate.go
│   ├── domain/
│   │   ├── datapoint.go
│   │   ├── graph.go
│   │   ├── policy.go
│   │   └── errors.go
│   ├── scheduler/
│   │   ├── scheduler.go
│   │   └── worker_pool.go
│   ├── collector/
│   │   └── prometheus/
│   │       ├── client.go
│   │       └── query.go
│   ├── engine/
│   │   ├── processor.go
│   │   ├── conditions.go
│   │   ├── properties.go
│   │   ├── identity.go
│   │   └── planner.go
│   ├── repository/
│   │   └── neo4j/
│   │       ├── client.go
│   │       ├── nodes.go
│   │       ├── relationships.go
│   │       └── tx.go
│   ├── observability/
│   │   ├── logger.go
│   │   └── metrics.go
│   └── version/
│       └── version.go
├── tests/
│   ├── integration/
│   └── fixtures/
├── .env.example
├── Dockerfile
├── Makefile
├── README.md
└── go.mod
```

## Papel de cada camada

### `cmd/neo-collector`

Ponto de entrada. Inicializa logger, carrega configuração, sobe scheduler e trata shutdown gracioso.

### `internal/config`

Responsável por:

- carregar `.env` para segredos e parâmetros de ambiente
- carregar YAML com targets, jobs, nodes e relacionamentos
- validar regras obrigatórias antes da aplicação subir
- normalizar diferenças de schema do YAML para um modelo interno estável

### `internal/scheduler`

Responsável por executar cada job no intervalo definido. O scheduler cria uma rotina por job e impede sobreposição de execuções do mesmo job.

### `internal/collector/prometheus`

Encapsula comunicação com Prometheus, inclusive timeout, TLS, tratamento de erro e conversão da resposta em `Datapoint`.

### `internal/engine`

É o núcleo da regra de negócio. Para cada datapoint:

- avalia condições de criação
- resolve propriedades estáticas, dinâmicas e condicionais
- garante presença de `name` em nodes
- calcula identidades estáveis
- gera um plano de mutação para Neo4j

### `internal/repository/neo4j`

Executa as operações de persistência com transação, obedecendo as regras:

- node primeiro
- relacionamento depois
- relacionamento nunca cria node
- `merge` atualiza
- `create` só insere se não existir
- `updated_at` sempre muda em mutações

### `internal/observability`

Centraliza logs estruturados, métricas operacionais e eventualmente health checks.

## Fluxo de dados

### 1. Inicialização

1. A aplicação lê `.env`.
2. Lê o YAML de configuração.
3. Valida a configuração inteira antes de iniciar o loop.
4. Cria clientes de Prometheus e Neo4j.
5. Registra um scheduler para cada combinação `target + job`.

### 2. Execução de job

1. O scheduler dispara um job pelo intervalo configurado.
2. O coletor chama o endpoint `/api/v1/query` do target Prometheus.
3. A resposta é convertida em uma lista de datapoints padronizados, contendo labels, value e timestamp.
4. Cada datapoint é processado isoladamente.

### 3. Processamento de datapoint

1. O engine avalia as condições dos nodes.
2. Para cada node elegível, resolve propriedades, injeta campos automáticos, calcula `node_uid` e gera o plano de persistência.
3. O engine avalia as condições dos relacionamentos.
4. Para cada relacionamento elegível, resolve source e target, resolve propriedades, calcula `rel_uid` e gera o plano de persistência sem criar nodes.

### 4. Persistência no Neo4j

1. Os nodes do datapoint são persistidos primeiro.
2. O repositório verifica existência por identidade de negócio.
3. Aplica `create` ou `merge`.
4. Os relacionamentos são persistidos em seguida.
5. Source e target são localizados por tipo e atributos definidos na configuração.
6. Se um dos lados não existir, o relacionamento é ignorado.
7. Se houver ambiguidade de match em source ou target, o datapoint falha e é logado como erro de regra.

### 5. Observabilidade

Para cada execução, a aplicação registra:

- target
- job
- duração
- quantidade de datapoints
- quantidade de nodes criados ou atualizados
- quantidade de relacionamentos criados ou atualizados
- erros por fase

## Decisões técnicas justificadas

### 1. Separar `.env` de YAML

`.env` fica com segredos e parâmetros operacionais. YAML fica com comportamento de coleta e mapeamento. Isso evita misturar credenciais com regra de negócio e permite promover a mesma configuração funcional entre ambientes.

### 2. Usar `cmd` + `internal`

Essa é a convenção idiomática em Go para encapsular regras internas e manter o ponto de entrada mínimo. Melhora manutenção e reduz acoplamento acidental.

### 3. Usar scheduler por intervalo, não cron

O PRD define intervalos em segundos por job, não janelas de calendário. `time.Ticker` com controle de sobreposição é mais simples, previsível e suficiente para esse caso.

### 4. Processar datapoints isoladamente

O PRD exige que a lógica seja aplicada separadamente para cada resultado da query. Isso simplifica regras condicionais, rastreabilidade de erro e idempotência.

### 5. Gerar IDs estáveis com UUID determinístico

`node_uid` e `rel_uid` devem ser estáveis. A melhor abordagem é UUIDv5 sobre uma chave canônica:

- node: `sorted(types) + name + sorted(template_hashes)`
- relacionamento: `source_identity + rel_type + target_identity + template_hash`

Isso evita duplicação entre execuções e mantém a exigência de estabilidade do contrato.

### 6. Persistir primeiro nodes, depois relacionamentos

O PRD determina que relacionamentos não criem nodes. Logo, a ordem de persistência precisa respeitar dependência explícita.

### 7. Usar `MERGE` com critério de negócio e `CREATE` para política estrita

Para `merge`, o repositório faz upsert com atualização de propriedades e `updated_at`. Para `create`, faz verificação prévia e não altera registros existentes. Isso materializa exatamente a política pedida no PRD.

### 8. Validar configuração no startup

Erros como ausência de `name`, uso de operador inválido, `template_hash` ausente na config, ou source ou target incompletos devem falhar antes da aplicação entrar em loop. Isso reduz falhas silenciosas em produção.

### 9. Usar drivers oficiais

- Prometheus: cliente HTTP oficial do ecossistema Prometheus
- Neo4j: `neo4j-go-driver/v5`

A decisão reduz risco de incompatibilidade, simplifica manutenção e melhora suporte a autenticação, timeout e retry.

### 10. Logs estruturados com `log/slog`

`slog` já faz parte da stdlib moderna do Go. Evita dependência desnecessária e facilita emitir logs JSON para container e agregadores.

### 11. Concorrência com limite

Cada job pode rodar em sua própria goroutine, mas o processamento de datapoints deve passar por pool limitado. Isso mantém escalabilidade sem saturar Neo4j ou Prometheus quando houver queries volumosas.

### 12. Tratamento explícito de ambiguidades

Se a configuração de match de source ou target encontrar mais de um node, o sistema não deve escolher arbitrariamente. A decisão correta é falhar aquele relacionamento e registrar erro, preservando integridade do grafo.

## Contrato interno sugerido

### Modelo canônico de node

```text
NodeTemplate
- Types []string
- TemplateHashes []string
- UpdatePolicy create|merge
- StaticProperties map[string]any
- DynamicProperties []DynamicProperty
- ConditionalProperties []ConditionalProperty
- Conditions []Condition
```

### Modelo canônico de relacionamento

```text
RelationshipTemplate
- Type string
- TemplateHash string
- UpdatePolicy create|merge
- Source NodeSelector
- Target NodeSelector
- StaticProperties map[string]any
- DynamicProperties []DynamicProperty
- ConditionalProperties []ConditionalProperty
- Conditions []Condition
```

## Estratégia de persistência no Neo4j

### Nodes

Critério de existência:

- label base `Entity`
- labels definidos em `Types`
- `name`

Campos automáticos:

- `node_uid`
- `origin`
- `created_at`
- `updated_at`

### Relacionamentos

Critério de existência:

- `source`
- `target`
- tipo do relacionamento
- `template_hashes`

Campos automáticos:

- `rel_uid`
- `origin`
- `created_at`
- `updated_at`

## Regras de validação obrigatórias

- cada job deve ter `name`, `query` e `interval_seconds > 0`
- cada node deve ter pelo menos um tipo
- cada node deve expor `name` em propriedade estática ou dinâmica
- cada node deve ter ao menos um `template_hashes`
- cada relacionamento deve ter `type`, `template_hash` (na config), `source` e `target`
- source e target devem ter `type` e pelo menos um atributo de match
- operadores válidos para label: `equals`, `not_equals`
- operadores válidos para value: `equals`, `not_equals`, `greater_than`, `less_than`
- aliases internos como `__value__` e `__timestamp__` são reservados

## Estratégia de testes

- testes unitários para validação de config
- testes unitários para avaliação de condições e resolução de propriedades
- testes unitários para geração de identidade estável
- testes de integração com Neo4j e Prometheus em containers
- testes de contrato para exemplos de YAML em `tests/fixtures`

## Resultado esperado

Essa arquitetura entrega:

- modularidade para crescer sem virar script monolítico
- segurança de contrato com o app principal
- idempotência nas reexecuções
- suporte a múltiplos targets e múltiplos jobs
- clareza para operação local e em container
