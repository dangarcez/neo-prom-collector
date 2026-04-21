# PRD - Plataforma de Ingestores Configuráveis para Neo4j

## 1. Visão Geral

Este documento descreve o produto esperado para uma linha de aplicações de ingestão que consomem dados estruturados de fontes externas e os transformam em nodes e relacionamentos no Neo4j.

A proposta não é limitada a uma fonte específica nem a uma stack específica. A mesma abordagem deve servir para implementações em diferentes linguagens, como Go ou Python, e para diferentes tipos de origem, como APIs HTTP, adaptadores executáveis, arquivos estruturados, filas ou ferramentas de observabilidade.

O objetivo é preservar o modelo funcional já validado no coletor atual:

- configuração declarativa
- execução periódica por fonte e por job
- processamento item a item
- criação e atualização idempotente de nodes e relacionamentos
- comportamento de persistência previsível e extensível

Este PRD é acompanhado por [config-contract.md](./config-contract.md), que descreve o contrato de configuração genérico esperado.

## 2. Problema e Objetivos

### Problema

Hoje existe uma solução funcional voltada a um caso específico de ingestão. Ao longo da evolução do projeto, o contrato funcional amadureceu além do cenário inicial e passou a representar um padrão reaproveitável:

- fontes diferentes podem fornecer dados equivalentes
- diferentes times podem preferir stacks distintas
- o núcleo de transformação e persistência tende a se repetir
- a ausência de um PRD consolidado dificulta replicar essa arquitetura em novas automações

### Objetivo principal

Definir uma base de produto para ingestores genéricos que:

- consumam dados de fontes externas variadas
- normalizem esses dados para um modelo comum de processamento
- transformem cada item retornado em mutações configuráveis no Neo4j
- permitam evolução por configuração, sem depender de lógica hardcoded por fonte

### Objetivos secundários

- reduzir o esforço para criar novos ingestores
- manter consistência de comportamento entre implementações
- permitir compatibilidade conceitual com o coletor atual
- facilitar manutenção, observabilidade e operação em container ou execução local

## 3. Contexto de Uso e Integração com o App Principal

Essas aplicações continuam sendo soluções auxiliares que se integram a um app principal e precisam respeitar um contrato consistente de persistência no grafo.

O foco desta linha de ingestores é:

- coletar dados externos
- aplicar regras declarativas
- gravar ou atualizar entidades no Neo4j de forma controlada

O foco não é:

- descobrir definições dinamicamente em endpoints externos
- administrar lifecycle completo do grafo
- excluir entidades expiradas
- substituir o app principal

Em particular, quando o contrato gerar `expires_at`, qualquer rotina de remoção ou tratamento posterior continuará fora do escopo do ingestor.

## 4. Escopo Funcional

### 4.1 Modelo operacional

A estrutura mental padrão do produto será:

- `sources`
- `jobs`
- `nodes`
- `relationships`

Cada `source` representa uma origem de dados independente.

Cada `job` representa uma operação periódica executada dentro de uma `source`.

Cada execução de `job` produz uma lista de itens normalizados.

Cada item é processado isoladamente.

### 4.2 Modelo normalizado de item

Independentemente da origem, o pipeline deve trabalhar sobre um envelope lógico comum:

- `labels`: mapa de atributos textuais do item
- `value`: valor principal opcional, quando fizer sentido para a fonte
- `timestamp`: instante associado ao item, quando fizer sentido para a fonte

Esse modelo preserva a ergonomia do contrato já validado, mesmo quando a origem não usa o termo "label" nativamente.

### 4.3 Criação e atualização de nodes

Para cada item, o produto deve permitir configurar múltiplos templates de node com:

- `types`
- `template_hashes`
- `update_policy`
- `static_properties`
- `label_properties`
- `conditional_properties`
- `property_transforms`
- `expiration_time_min`
- `conditions`

Regras obrigatórias:

- todo node precisa ter ao menos um tipo
- todo node precisa ter `template_hashes`
- todo node precisa definir a propriedade `name`

### 4.4 Criação e atualização de relacionamentos

Para cada item, o produto deve permitir configurar múltiplos templates de relacionamento com:

- `type`
- `template_hash`
- `update_policy`
- `static_properties`
- `label_properties`
- `conditional_properties`
- `property_transforms`
- `expiration_time_min`
- `conditions`
- `source`
- `target`

Os blocos `source` e `target` devem localizar nodes existentes por tipo e atributos de match. Relacionamentos não devem criar nodes implicitamente.

### 4.5 Policies de persistência

O contrato funcional deve suportar:

- `create`
- `merge`
- `merge_at_change`

Semântica esperada:

- `create`: cria apenas quando a entidade equivalente ainda não existe
- `merge`: cria quando não existe e atualiza quando existe
- `merge_at_change`: cria quando não existe e atualiza apenas quando propriedades de negócio mudarem

### 4.6 Condições e propriedades derivadas

O produto deve suportar:

- condições do tipo `label`
- condições do tipo `label_exists`
- condições do tipo `value`
- propriedades condicionais do tipo `static`
- propriedades condicionais do tipo `label`

Também deve suportar `property_transforms`, com arquitetura extensível, incluindo na primeira versão:

- `TO_UPPER`
- `TO_LOWER`

### 4.7 Expiração lógica

Nodes e relacionamentos podem declarar `expiration_time_min`.

Quando configurado:

- o repositório deve gerar `expires_at`
- `expires_at` deve ser calculado como `agora_utc + expiration_time_min`
- `expires_at` só deve ser criado ou renovado em `create` e `merge`
- `merge_at_change` não deve renovar `expires_at`

### 4.8 Modos operacionais e runtime

O produto deve permitir:

- execução periódica por `job`
- `dry_run`
- controle de paralelismo
- `sleep_seconds` para amortecer throughput entre itens
- logs estruturados ou textuais

## 5. Modelo de Configuração Genérico

O contrato canônico deve ser genérico e orientado a fontes:

- top-level: `sources`
- dentro de cada source: `jobs`
- dentro de cada job: `nodes` e `relationships`

O contrato deve permitir campos específicos por adapter de source, mas o pipeline central deve continuar operando sobre o mesmo modelo lógico.

Princípios do contrato:

- o contrato central descreve transformação e persistência
- a origem concreta descreve apenas como produzir itens normalizados
- a configuração deve ser validada antes do loop de execução
- implementações concretas podem especializar partes do schema desde que preservem a semântica funcional

## 6. Semântica de Processamento e Persistência

### 6.1 Pipeline por execução

Para cada `source` e `job`:

1. executar a operação configurada
2. obter uma coleção de itens normalizados
3. processar cada item de forma isolada
4. gerar nodes elegíveis
5. gerar relacionamentos elegíveis
6. persistir nodes antes de relacionamentos

### 6.2 Resolução de propriedades

A ordem de resolução deve ser:

1. `static_properties`
2. `label_properties`
3. `conditional_properties`
4. `property_transforms`
5. injeção de campos automáticos na persistência

### 6.3 Regras para ausência de dados

O comportamento esperado deve seguir o contrato atual:

- se uma propriedade em `label_properties` referenciar uma label ausente, a propriedade é omitida
- se uma propriedade condicional do tipo `label` referenciar uma label ausente, a propriedade é omitida
- se `match_attributes.labels` referenciar uma label ausente, isso continua sendo erro de processamento do seletor

### 6.4 Regras de identidade

Para nodes:

- identidade de existência baseada em tipo e `name`
- `node_uid` estável gerado automaticamente

Para relacionamentos:

- `template_hash` é a entrada canônica de configuração
- a persistência usa `template_hashes` com um único item
- `rel_uid` estável deve ser gerado automaticamente
- quando houver múltiplos matches em `source` e `target`, o sistema deve criar o produto cartesiano entre os pares

### 6.5 Campos automáticos

O pipeline de persistência deve gerar ou manter:

- label base `Entity` para nodes
- `origin = "auto"`
- `node_uid`
- `rel_uid`
- `created_at`
- `updated_at`
- `expires_at` quando aplicável

### 6.6 Semântica de `merge_at_change`

`merge_at_change` deve comparar apenas propriedades de negócio declaradas na configuração e ignorar campos automáticos, incluindo:

- ids automáticos
- `origin`
- `created_at`
- `updated_at`
- `expires_at`
- hashes automáticos persistidos

Quando nada mudar:

- a mutação deve ser tratada como `skipped`
- `updated_at` não deve ser renovado
- `expires_at` não deve ser renovado

## 7. Requisitos Não Funcionais

### 7.1 Extensibilidade

A solução deve facilitar:

- adição de novos tipos de `source`
- adição de novos processors em `property_transforms`
- novas implementações em stacks diferentes

### 7.2 Operabilidade

Toda implementação concreta deve prever:

- execução local
- execução em container
- documentação operacional mínima
- configuração externa por arquivo e ambiente

### 7.3 Segurança e previsibilidade

O sistema deve:

- validar configuração antes da execução contínua
- evitar criação implícita de nodes a partir de relacionamentos
- ser idempotente em reprocessamentos equivalentes
- expor comportamento consistente em logs e métricas

### 7.4 Observabilidade

As implementações devem registrar, no mínimo:

- source
- job
- duração da execução
- volume de itens processados
- quantidade de nodes criados, atualizados ou ignorados
- quantidade de relacionamentos criados, atualizados ou ignorados
- erros de processamento

## 8. Compatibilidade com o Coletor Atual

Este PRD é genérico por design, mas foi escrito para ser compatível conceitualmente com o coletor existente.

Mapeamento conceitual:

- `prom_targets` -> `sources`
- query Prometheus -> uma forma específica de `job.operation`
- datapoint Prometheus -> um caso natural do item normalizado com `labels`, `value` e `timestamp`

Compatibilidades esperadas:

- o vocabulário de `jobs`, `nodes`, `relationships`, `conditions`, `update_policy` e `property_transforms` é preservado
- `label_properties` continua existindo como nome de contrato, mas deve ser interpretado como atributos do item normalizado, e não apenas labels Prometheus
- Prometheus passa a ser tratado como um adapter entre vários possíveis

Este material não redefine o coletor atual. Ele define um padrão mais amplo para novas soluções semelhantes.

## 9. Entregáveis Esperados de Qualquer Nova Implementação

Qualquer implementação concreta derivada deste PRD deve entregar:

- código funcional
- estrutura de projeto clara
- configuração externa por ambiente e por arquivo declarativo
- documentação mínima de build, execução local, testes e operação em container
- documentação do contrato funcional adotado

## 10. Casos de Referência que a Solução Deve Atender

O produto deve ser capaz de sustentar implementações para cenários como:

- fonte HTTP que retorna itens estruturados
- fonte baseada em script ou adapter externo, inclusive em Python
- processamento periódico por source e job
- criação de node com `name` vindo da fonte
- relacionamento entre nodes já existentes
- fan-out de relacionamentos quando múltiplos matches forem válidos
- `merge` versus `merge_at_change`
- `label_exists`
- `property_transforms`
- `expiration_time_min`
- `dry_run`

## 11. Fora de Escopo

Estão fora do escopo deste PRD:

- exclusão automática de nodes ou relacionamentos expirados
- descoberta dinâmica obrigatória de schema por fonte
- definição de um único adapter obrigatório
- obrigatoriedade de uma linguagem específica
- substituição das regras do app principal

## 12. Critérios de Aceite do Documento

Este PRD será considerado adequado quando:

- permitir desenhar um novo ingestor sem depender de Prometheus
- permitir desenhar um novo ingestor sem depender de Go
- preservar as funcionalidades maduras já consolidadas no contrato atual
- deixar explícito o modelo `sources -> jobs -> nodes -> relationships`
- ser compreensível sem depender do `project.prd` original
- servir como base para diferentes implementações concretas
