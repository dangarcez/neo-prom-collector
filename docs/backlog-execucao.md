# Backlog de Execucao

## Objetivo

Organizar a execucao do projeto em fases pequenas e ordenadas, priorizando primeiro a entrega funcional minima, depois robustez operacional e por fim validacao completa e documentacao final.

Este backlog foi derivado de [tasks-breakdown.md](/home/garcez/neo_collector_go/docs/tasks-breakdown.md) e da arquitetura definida em [architecture.md](/home/garcez/neo_collector_go/docs/architecture.md).

## Criterio de priorizacao

- primeiro o que destrava o fluxo fim a fim
- depois o que reduz risco operacional
- por ultimo o que aumenta confianca, empacotamento e acabamento

## MVP

### Objetivo da fase

Entregar uma versao funcional que rode localmente, leia `.env` e YAML, consulte Prometheus em intervalos configurados e crie ou atualize nodes e relacionamentos no Neo4j de forma idempotente.

### Ordem de execucao

1. Inicializar o projeto Go
   - cria modulo, estrutura base e ponto de entrada
2. Definir modelos de configuracao
   - estabiliza o contrato interno antes da implementacao
3. Definir modelos de dominio
   - fixa as entidades centrais do processamento
4. Implementar bootstrap da aplicacao
   - prepara inicializacao e ciclo de vida do processo
5. Implementar carregamento de `.env` e YAML
   - habilita entrada real de configuracao
6. Implementar validacao de configuracao
   - impede subida com regras invalidas
7. Implementar cliente Prometheus
   - habilita coleta de datapoints
8. Implementar cliente Neo4j
   - habilita persistencia no grafo
9. Implementar gerador de identidades estaveis
   - garante `node_uid` e `rel_uid` deterministicos
10. Implementar avaliacao de condicoes
   - habilita elegibilidade de nodes e relacionamentos
11. Implementar resolucao de propriedades
   - materializa campos estaticos, dinamicos e condicionais
12. Implementar planner de mutacoes
   - transforma datapoint em plano de escrita
13. Implementar persistencia de nodes
   - cobre `create` e `merge` para entidades
14. Implementar persistencia de relacionamentos
   - cobre `create` e `merge` sem criar nodes
15. Implementar orquestracao por datapoint
   - conecta engine e repositorios no fluxo correto
16. Implementar scheduler por job
   - executa `target + job` em intervalos configurados
17. Preparar execucao local
   - adiciona `.env.example`, `Makefile` e instrucoes minimas de uso

### Saida esperada

- aplicacao sobe localmente
- configuracao valida antes de iniciar
- queries sao executadas em Prometheus
- cada datapoint e processado isoladamente
- nodes e relacionamentos sao escritos no Neo4j conforme `update_policy`
- projeto pode ser testado manualmente de ponta a ponta

## Fase 2

### Objetivo da fase

Endurecer o sistema para uso mais confiavel em ambiente de execucao continuo, com melhor controle operacional, empacotamento e cobertura basica automatizada.

### Ordem de execucao

1. Implementar observabilidade
   - adiciona logs estruturados, contadores e visibilidade do runtime
2. Implementar limite de concorrencia
   - evita saturacao de Prometheus e Neo4j sob volume maior
3. Preparar containerizacao
   - empacota a aplicacao para execucao padronizada
4. Escrever testes unitarios
   - cobre validacao, condicoes, propriedades, identidade e planner

### Saida esperada

- aplicacao gera logs uteis para diagnostico
- processamento possui limite de paralelismo configuravel
- container Docker constroi e executa corretamente
- componentes centrais possuem cobertura unitaria

## Fase 3

### Objetivo da fase

Fechar a entrega com validacao integrada real e documentacao consolidada para operacao, manutencao e evolucao.

### Ordem de execucao

1. Escrever testes de integracao
   - valida Prometheus e Neo4j reais em ambiente controlado
2. Finalizar documentacao funcional
   - consolida arquitetura, configuracao, build, run, deploy e testes

### Saida esperada

- fluxo fim a fim validado por automacao
- documentacao suficiente para desenvolvimento, operacao e handoff

## Sequencia recomendada resumida

### Semana ou bloco 1

- estrutura do projeto
- modelos de configuracao
- modelos de dominio
- bootstrap

### Semana ou bloco 2

- carregamento e validacao de configuracao
- cliente Prometheus
- cliente Neo4j
- identidade estavel

### Semana ou bloco 3

- condicoes
- propriedades
- planner
- persistencia de nodes

### Semana ou bloco 4

- persistencia de relacionamentos
- orquestracao por datapoint
- scheduler
- execucao local

### Semana ou bloco 5

- observabilidade
- limite de concorrencia
- containerizacao
- testes unitarios

### Semana ou bloco 6

- testes de integracao
- documentacao final

## Riscos que merecem atencao cedo

- ambiguidade no criterio de existencia de nodes e relacionamentos
- normalizacao incompleta entre `type` e `types`
- diferenca entre `template_hash` e `template_hashes`
- geracao incorreta de IDs estaveis
- match ambiguo de source e target em relacionamentos
- queries Prometheus retornando alto volume sem controle de concorrencia

## Definicao de pronto por fase

### MVP

- executa localmente com configuracao de exemplo
- escreve nodes e relacionamentos no Neo4j
- respeita `create` e `merge`
- atualiza `updated_at` corretamente

### Fase 2

- gera logs estruturados suficientes para operacao
- empacota em container
- possui testes unitarios dos componentes criticos

### Fase 3

- possui testes de integracao passando
- documentacao cobre build, run, deploy e troubleshooting basico
