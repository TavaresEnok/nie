# Documentação — Network Intelligence Engine (NIE)

Ponto de entrada da pasta `docs/`. O status de implementação fica no
[`README.md` da raiz](../README.md); aqui ficam a especificação, as decisões e
o material histórico.

## Ordem de precedência

Em caso de conflito, vale o primeiro item da lista:

1. **Código e testes** — o que está implementado de fato.
2. [`DECISOES.md`](DECISOES.md) — ADRs do que foi implementado (ADR-007 e
   ADR-008 substituem partes do ADR-004 e o ADR-005).
3. [`arquitetura/ADENDO_MESTRE_NIE_V2_1.md`](arquitetura/ADENDO_MESTRE_NIE_V2_1.md) —
   corrige a V2 no motor de coleta (Scrapy/Crawl4AI como workers).
4. [`arquitetura/PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md`](arquitetura/PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md) —
   arquitetura de referência V2.
5. [`historico/`](historico/) — concepção original; não descreve o estado atual.

## Raiz

| Arquivo | Conteúdo |
|---|---|
| [`DECISOES.md`](DECISOES.md) | Registro de decisões (ADR-001 a ADR-008) |
| [`MVP_SPEC.md`](MVP_SPEC.md) | Escopo do vertical slice da Fase 1 e critérios de aceite; escopo histórico, critérios ainda válidos como regressão |

## `arquitetura/` — especificação vigente

| Arquivo | Conteúdo |
|---|---|
| [`PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md`](arquitetura/PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md) | Plano V2 canônico: stack, evidências, segurança, fases 0–4 |
| [`PLANO_MESTRE_V2_EXPANDIDO.md`](arquitetura/PLANO_MESTRE_V2_EXPANDIDO.md) | Variante editorial do mesmo plano, com diagramas adicionais |
| [`ADENDO_MESTRE_NIE_V2_1.md`](arquitetura/ADENDO_MESTRE_NIE_V2_1.md) | V2.1: executores especializados, contrato dos executores, benchmark comportamental, critérios de aceite |
| [`DIAGRAMA_ARQUITETURA_NIE_2_0.md`](arquitetura/DIAGRAMA_ARQUITETURA_NIE_2_0.md) | Fluxo de investigação e implantação em máquina única |
| [`CATALOGO_COMPLETO_59_REGISTROS.md`](arquitetura/CATALOGO_COMPLETO_59_REGISTROS.md) | Classificação V2.1 dos 59 registros do catálogo original |
| [`MATRIZ_59_FONTES_E_PROJETOS_V2_1.csv`](arquitetura/MATRIZ_59_FONTES_E_PROJETOS_V2_1.csv) | Mesma classificação, detalhada, em CSV com `;` |
| [`FONTES_E_PROJETOS.md`](arquitetura/FONTES_E_PROJETOS.md) | URLs dos projetos e fontes mencionados |

## `historico/` — preservação da concepção (30–31/10/2026)

Material congelado; não editar além de notas de contexto.

| Arquivo | Conteúdo |
|---|---|
| [`LEIA_PRIMEIRO.md`](historico/LEIA_PRIMEIRO.md) | Guia do pacote de preservação original |
| [`ARQUIVO_MESTRE_V1.md`](historico/ARQUIVO_MESTRE_V1.md) / [`.pdf`](historico/ARQUIVO_MESTRE_V1.pdf) | Dossiê + registro cronológico + quatro anexos integrais |
| [`01_DOSSIE_COMPLETO.md`](historico/01_DOSSIE_COMPLETO.md) | Especificação original, decisões abertas, benchmark de 20 perguntas |
| [`02_REGISTRO_CRONOLOGICO_DA_CONVERSA.md`](historico/02_REGISTRO_CRONOLOGICO_DA_CONVERSA.md) | Sequência da conversa de concepção |
| [`Agente_central_de_investigacao.md`](historico/Agente_central_de_investigacao.md) | Transcrição bruta da pesquisa inicial |
| [`originais/`](historico/originais/) | Os quatro anexos originais, sem edição |
| [`CONTINUAR_NO_FUTURO.md`](historico/CONTINUAR_NO_FUTURO.md) | Prompt de retomada escrito antes do código |
| `Projeto_Busca_Autonoma_Backup_Completo.zip` | Backup original completo, com `SHA256SUMS.txt` |

## Fora de `docs/`

- Benchmark de 20 perguntas: [`internal/benchmark/20_perguntas_telecom.json`](../internal/benchmark/20_perguntas_telecom.json)
  (embutido no binário; cópia única).
- Esquema de dados: [`internal/store/migrations/`](../internal/store/migrations/).
