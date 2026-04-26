# Contrato de Configuração Genérico para Ingestores

## Objetivo

Este documento descreve um contrato de configuração genérico para aplicações de ingestão que transformam dados externos em nodes e relacionamentos no Neo4j.

O foco aqui é o contrato funcional comum entre implementações. O documento não descreve a implementação específica do coletor atual.

## Princípios do Contrato

- o top-level canônico é `sources`
- `jobs`, `nodes` e `relationships` são preservados
- cada source concreta é responsável por produzir itens normalizados
- o pipeline central opera sobre um modelo lógico comum
- a configuração deve ser suficiente para descrever transformação e persistência sem código hardcoded por caso

## Modelo Normalizado de Item

Cada source deve produzir itens que possam ser interpretados logicamente como:

- `labels`: atributos textuais do item
- `value`: valor principal opcional
- `timestamp`: timestamp opcional

Observação importante:

- o nome `label` é preservado por compatibilidade com o contrato já consolidado
- neste contexto, `label` significa atributo lógico do item normalizado, e não necessariamente label de Prometheus

## Estrutura YAML Canônica

```yaml
sources:
  - name: inventory_api
    type: http_json
    connection:
      base_url: https://inventory.example.com
      timeout_seconds: 10
      verify_tls: true
    runtime:
      default_interval_seconds: 300
      sleep_seconds: 0.1
      dry_run: false
    jobs:
      - name: assets_sync
        operation:
          kind: http_request
          method: GET
          path: /api/assets
        nodes:
          - types:
              - Asset
            template_hashes:
              - asset-v1
            update_policy: merge
            expiration_time_min: 60
            label_properties:
              name: asset_name
              owner: owner
              observed_value: __value__
              observed_at: __timestamp__
            conditional_properties:
              - type: static
                name: criticality
                value: high
                conditions:
                  - type: label
                    label: environment
                    equals: production
            property_transforms:
              - property: name
                process:
                  - type: TO_UPPER
              - property: owner
                process:
                  - type: REGEX
                    pattern: "/(\\w+)_(\\w+)/"
                    output: "$1_and_$2"

        relationships:
          - type: BELONGS_TO
            template_hash: asset-belongs-team-v1
            update_policy: merge
            source:
              type: Asset
              match_attributes:
                labels:
                  name: asset_name
            target:
              type: Team
              match_attributes:
                labels:
                  name: team_name
```

## Bloco `sources[]`

Cada item de `sources` representa uma origem de dados independente.

Campos recomendados:

| Campo | Tipo | Obrigatório | Descrição |
| --- | --- | --- | --- |
| `name` | string | sim | Nome lógico da source. |
| `type` | string | sim | Tipo do adapter da source, por exemplo `http_json`, `exec`, `prometheus`, `file_json`. |
| `connection` | objeto | não | Configuração de conectividade da source, quando fizer sentido. |
| `runtime` | objeto | não | Configuração de execução compartilhada entre jobs da source. |
| `jobs` | lista | sim | Lista de jobs executados nessa source. |

Observação:

- `connection` e `operation` são pontos de extensão específicos por adapter
- o contrato funcional do pipeline começa a partir do item normalizado produzido por cada job

## Bloco `runtime`

`runtime` controla o comportamento operacional da source.

Campos:

| Campo | Tipo | Obrigatório | Default | Descrição |
| --- | --- | --- | --- | --- |
| `default_interval_seconds` | inteiro | não | `60` | Intervalo padrão aplicado a jobs sem `interval_seconds`. |
| `sleep_seconds` | número | não | `0` | Pausa após o processamento de cada item. Aceita frações. |
| `dry_run` | boolean | não | `false` | Se `true`, executa coleta e planejamento sem gravar no Neo4j. |

## Bloco `jobs[]`

Cada job define uma operação periódica dentro de uma source.

Campos:

| Campo | Tipo | Obrigatório | Descrição |
| --- | --- | --- | --- |
| `name` | string | sim | Nome lógico do job. |
| `operation` | objeto | sim | Descrição da operação da source para produzir itens. |
| `interval_seconds` | inteiro | não | Intervalo da execução do job. |
| `nodes` | lista | não | Templates de node avaliados para cada item. |
| `relationships` | lista | não | Templates de relacionamento avaliados para cada item. |

Semântica:

- cada job produz zero ou mais itens
- a lógica de nodes e relacionamentos é aplicada item a item
- um item não deve interferir diretamente no processamento do outro

## Bloco `nodes[]`

Cada template de node define como um item pode gerar ou atualizar um node.

Campos:

| Campo | Tipo | Obrigatório | Default | Descrição |
| --- | --- | --- | --- | --- |
| `type` | string | condicional | - | Alias para um único tipo. |
| `types` | lista de string | condicional | - | Lista de labels técnicas do node. |
| `template_hashes` | lista de string | sim | - | Hashes da definição do node. |
| `update_policy` | string | não | `create` | `create`, `merge` ou `merge_at_change`. |
| `expiration_time_min` | inteiro | não | ausente | Gera `expires_at` em `create` e `merge`. |
| `static_properties` | mapa | não | `{}` | Propriedades literais. |
| `label_properties` | mapa string->string | não | `{}` | Propriedades resolvidas a partir de labels ou tokens. |
| `conditional_properties` | lista | não | `[]` | Propriedades aplicadas quando condições passam. |
| `property_transforms` | lista | não | `[]` | Processamentos aplicados sobre propriedades já resolvidas. |
| `conditions` | lista | não | `[]` | Condições para o template ser aplicado. |

Regras:

- deve existir ao menos um tipo
- `template_hashes` é obrigatório
- a propriedade `name` é obrigatória no resultado do template

## Bloco `relationships[]`

Cada template de relacionamento define como um item pode gerar ou atualizar um relacionamento entre nodes existentes.

Campos:

| Campo | Tipo | Obrigatório | Default | Descrição |
| --- | --- | --- | --- | --- |
| `type` | string | sim | - | Tipo técnico do relacionamento. |
| `template_hash` | string | sim | - | Hash canônico da definição do relacionamento. |
| `update_policy` | string | não | `create` | `create`, `merge` ou `merge_at_change`. |
| `expiration_time_min` | inteiro | não | ausente | Gera `expires_at` em `create` e `merge`. |
| `static_properties` | mapa | não | `{}` | Propriedades literais. |
| `label_properties` | mapa string->string | não | `{}` | Propriedades resolvidas do item. |
| `conditional_properties` | lista | não | `[]` | Propriedades aplicadas quando condições passam. |
| `property_transforms` | lista | não | `[]` | Processamentos aplicados sobre propriedades já resolvidas. |
| `conditions` | lista | não | `[]` | Condições para o relacionamento ser gerado. |
| `source` | objeto | sim | - | Seletor do node de origem. |
| `target` | objeto | sim | - | Seletor do node de destino. |

Regras:

- relacionamento não cria nodes implicitamente
- `source` e `target` precisam localizar nodes existentes
- se um lado não encontrar match, o relacionamento é ignorado
- se houver múltiplos matches válidos em `source` e `target`, o sistema cria o produto cartesiano entre os pares

## Blocos `source` e `target`

Esses blocos definem como localizar nodes existentes para um relacionamento.

Campos:

| Campo | Tipo | Obrigatório | Descrição |
| --- | --- | --- | --- |
| `type` | string | sim | Tipo técnico do node a ser encontrado. |
| `match_attributes.static` | mapa | não | Atributos fixos para o match. |
| `match_attributes.labels` | mapa string->string | não | Atributos resolvidos a partir do item normalizado. |

Regras:

- pelo menos um atributo de match deve existir
- ausência de label usada em `match_attributes.labels` continua sendo erro de processamento

## Semântica dos Blocos de Propriedade

### `static_properties`

Mapa de valores literais copiados diretamente.

### `label_properties`

Mapa de `propriedade -> token`.

O token pode apontar para:

- uma label do item normalizado
- `__value__`
- `__timestamp__`

Se a label não existir:

- a propriedade é omitida

### `conditional_properties`

Permite aplicar propriedades quando um conjunto de condições passa.

Tipos suportados:

- `static`
- `label`

### `property_transforms`

Executa uma lista ordenada de processors sobre uma propriedade já resolvida.

Schema:

```yaml
property_transforms:
  - property: name
    process:
      - type: TO_UPPER
  - property: metric_name
    process:
      - type: REGEX
        pattern: "/(\\w+)_(\\w+)/"
        output: "$1_and_$2"
```

Processors suportados na base do contrato:

- `TO_UPPER`
- `TO_LOWER`
- `REGEX`

Campos adicionais de `REGEX`:

- `pattern`: padrão regex com ao menos um grupo de captura
- `output`: saída gerada a partir dos grupos, usando `$1`, `$2`, etc.

Regras:

- roda depois de `static_properties`, `label_properties` e `conditional_properties`
- roda antes dos campos automáticos
- se a propriedade não existir, o transform é ignorado
- se o valor não for string, `TO_UPPER`, `TO_LOWER` e `REGEX` são ignorados
- se `REGEX` não encontrar match, a propriedade fica inalterada
- `REGEX` aceita padrões com ou sem delimitadores `/.../`
- `REGEX` precisa referenciar ao menos um grupo existente no `output`

### `expiration_time_min`

Campo opcional para node ou relacionamento.

Semântica:

- gera `expires_at = agora_utc + expiration_time_min`
- `expires_at` é gerado apenas na persistência
- `expires_at` só entra em `create` e `merge`
- `merge_at_change` não deve renovar `expires_at`

## Condições

Condições suportadas:

- `label`
- `label_exists`
- `value`

### `label`

Usa uma label do item normalizado com operadores:

- `equals`
- `not_equals`

### `label_exists`

Verifica apenas a existência da label.

### `value`

Usa o `value` do item normalizado com operadores:

- `equals`
- `not_equals`
- `greater_than`
- `less_than`

## Tokens Reservados

Tokens reservados do contrato:

| Token | Semântica |
| --- | --- |
| `__value__` | Valor principal do item normalizado. |
| `__timestamp__` | Timestamp do item normalizado em UTC. |

Se uma source não produzir `value` ou `timestamp`, jobs que dependam desses tokens não devem ser considerados válidos para aquela source.

## Semântica de `update_policy`

### `create`

- cria apenas se a entidade equivalente não existir
- se `expiration_time_min` existir, cria `expires_at` na inserção

### `merge`

- cria quando não existe
- atualiza quando existe
- se `expiration_time_min` existir, cria ou renova `expires_at`

### `merge_at_change`

- cria quando não existe
- só atualiza quando propriedades de negócio mudam
- ignora campos automáticos na comparação
- não renova `expires_at`

## Campos Automáticos Gerados pela Persistência

### Nodes

- label base `Entity`
- `node_uid`
- `template_hashes`
- `origin = "auto"`
- `created_at`
- `updated_at`
- `expires_at` quando aplicável

### Relacionamentos

- `rel_uid`
- `template_hash`
- `origin = "auto"`
- `created_at`
- `updated_at`
- `expires_at` quando aplicável

Observação:

- para relacionamentos, a entrada canônica é `template_hash`
- a persistência usa `template_hash` como string

## Regras de Validação Importantes

O contrato deve falhar antes da execução contínua em casos como:

- source sem `name`
- source sem `type`
- source sem jobs
- job sem `name`
- job sem `operation`
- `interval_seconds <= 0` após normalização
- node sem tipo
- node sem `template_hashes`
- node sem `name`
- relacionamento sem `type`
- relacionamento sem `template_hash`
- `source` ou `target` sem `type`
- `source` ou `target` sem atributos de match
- `update_policy` fora de `create`, `merge`, `merge_at_change`
- `property_transforms` inválido
- `property_transforms[].process[]` do tipo `REGEX` sem `pattern`, sem `output`, sem referencia a grupo, sem grupo de captura ou com referencia a grupo inexistente
- `expiration_time_min <= 0` quando informado

## Observações de Compatibilidade com o Coletor Atual

Este contrato é genérico, mas mantém compatibilidade conceitual com o coletor já existente.

Mapeamentos:

- `prom_targets` -> `sources`
- query Prometheus -> um tipo específico de `operation`
- datapoint Prometheus -> item normalizado com `labels`, `value` e `timestamp`

Compatibilidades preservadas:

- `jobs`, `nodes` e `relationships` permanecem como nomes centrais
- `label_properties`, `conditional_properties`, `property_transforms`, `update_policy` e `match_attributes` permanecem
- `template_hashes` em relacionamento pode ser mencionado apenas como alias de compatibilidade, mas `template_hash` continua sendo o campo canônico de entrada

## Observação Final

Cada implementação concreta pode:

- escolher linguagem
- escolher runtime
- definir adapters específicos de source
- especializar o bloco `operation`

Mas deve preservar este contrato funcional para manter consistência entre ingestores.
