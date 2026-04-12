# Guia de Ingestao Automatica

Este guia define o contrato minimo para criar nodes e relacionamentos fora do app, diretamente no Neo4j, de modo que eles aparecam corretamente na interface.

## Convencao de origem

- `origin = "manual"`: reservado para objetos criados pelo app.
- `origin = "auto"`: usado por automacoes externas.

Nao use outros valores se quiser compatibilidade total com a sinalizacao visual do frontend.

## Regra geral

Automacoes externas nao devem criar propriedades genericas como `uuid` ou `hash` esperando que o app as interprete.

Os nomes esperados pelo sistema sao exatos:

- nodes: `node_uid`, `name`, `template_hashes`, `origin`
- relacionamentos: `rel_uid`, `template_hashes`, `origin`

## Como obter os hashes corretos

Os hashes usados na automacao devem ser os mesmos cadastrados nas definicoes do sistema.

Fontes possiveis:

- pagina `Definicoes` no frontend
- endpoint `GET /api/v1/definitions`

## Renomeacao de definicoes

O `template_hash` e a identidade estavel da definicao. O `name` pode mudar ao longo do tempo.

Exemplo:

- uma definicao de node pode sair de `RECURSO` para `RESOURCE`
- uma definicao de relacionamento pode sair de `UTILIZA` para `USES`

Quando isso acontece pelo app:

- o SQLite mantem o mesmo `template_hash`
- os nodes existentes com esse `template_hash` recebem o novo label tecnico
- os relacionamentos existentes com esse `template_hash` recebem o novo tipo tecnico

Para automacoes externas, a regra pratica e:

1. trate `template_hash` como chave canonica
2. consulte periodicamente o nome tecnico atual da definicao
3. ao criar novos nodes ou relacionamentos, use sempre o nome tecnico mais recente retornado pelo app

## Contrato para nodes automaticos

Cada node criado fora do app deve ter:

- label base `Entity`
- labels adicionais com os nomes tecnicos das definicoes aplicadas
- `node_uid`: UUID estavel em formato string
- `name`: nome exibido e pesquisavel
- `template_hashes`: lista com um ou mais hashes validos de definicoes de node
- `origin = "auto"`

Campos recomendados:

- `created_at` em ISO 8601
- `updated_at` em ISO 8601

Exemplo:

```cypher
CREATE (n:Entity:Person {
  node_uid: "7f0ec5a9-4e76-4cd4-a75e-bfc8f3dbcd55",
  name: "Alice",
  template_hashes: ["aaaaaaaaaaaaaaaa"],
  origin: "auto",
  created_at: "2026-04-05T12:00:00Z",
  updated_at: "2026-04-05T12:00:00Z",
  email: "alice@example.com",
  department: "Graph"
})
```

## Contrato para relacionamentos automaticos

Cada relacionamento criado fora do app deve ter:

- origem e destino ligados a nodes compativeis com o contrato acima
- tipo do relacionamento igual ao nome tecnico da definicao
- `rel_uid`: UUID estavel em formato string
- `template_hashes`: lista com o hash valido da definicao de relacionamento
- `origin = "auto"`

Campos recomendados:

- `created_at` em ISO 8601
- `updated_at` em ISO 8601

Exemplo:

```cypher
MATCH (source:Entity {node_uid: "7f0ec5a9-4e76-4cd4-a75e-bfc8f3dbcd55"})
MATCH (target:Entity {node_uid: "ee5d3568-47c8-4897-8e30-c8018fe0f2d8"})
CREATE (source)-[:WorksFor {
  rel_uid: "17c7c1aa-2ec7-4027-b6b9-f12a02f98893",
  template_hashes: ["bbbbbbbbbbbbbbbb"],
  origin: "auto",
  created_at: "2026-04-05T12:10:00Z",
  updated_at: "2026-04-05T12:10:00Z",
  role: "Engineer"
}]->(target)
```

## Observacoes importantes

- `template_hashes` de node e de relacionamento devem existir no SQLite do app.
- O tipo do relacionamento no Neo4j deve bater com o `name` atual da definicao.
- Os labels extras do node devem bater com os nomes tecnicos atuais das definicoes aplicadas.
- Se voce omitir `origin`, o item pode ate aparecer, mas ficara sem classificacao visual de origem.
- Se voce omitir `node_uid` ou `rel_uid`, o app nao conseguira detalhar e operar corretamente sobre esses itens.

## Recomendacao de estrategia

Para automacoes externas, prefira este fluxo:

1. Consultar as definicoes ativas no app e cachear `template_hash` e `name`.
2. Criar nodes com `:Entity`, `node_uid`, `template_hashes` e `origin = "auto"`.
3. Criar relacionamentos com `rel_uid`, `template_hashes`, tipo Neo4j correto e `origin = "auto"`.
4. Reconciliar o cache local de nomes tecnicos quando uma definicao for renomeada no app.
5. Atualizar `updated_at` quando houver mutacao posterior.
