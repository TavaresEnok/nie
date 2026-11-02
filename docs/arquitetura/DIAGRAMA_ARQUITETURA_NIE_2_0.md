# Network Intelligence Engine 2.0 — Diagrama de arquitetura

> Proposta arquitetural do Plano Mestre V2 (01/11/2026). Representa componentes planejados, não funcionalidades já implementadas.

## Fluxo completo de investigação

```text
                        ┌────────────────────────────────────┐
                        │      USUÁRIO (PC / CELULAR)        │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ PAINEL SVELTE + VITE               │
                        │ busca, campanhas, grafo, biblioteca│
                        │ Caddy/HTTPS se acesso externo      │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │       NÚCLEO GO + HUMA            │
                        │ API, orquestrador, autenticação,   │
                        │ planejamento, cotas e políticas   │
                        └─────────────────┬──────────────────┘
                                          │
                    ┌─────────────────────┼──────────────────────┐
                    ▼                     ▼                      ▼
       ┌──────────────────────┐ ┌──────────────────────┐ ┌──────────────────────┐
       │ GIT EYE              │ │ WEB EYE              │ │ BUSCA NO ACERVO      │
       │ GitHub/GitLab APIs   │ │ SearXNG/APIs*        │ │ PostgreSQL FTS      │
       │ repositórios úteis   │ │ links, fóruns, RSS   │ │ pg_trgm + filtros   │
       └──────────┬───────────┘ └──────────┬───────────┘ └──────────┬───────────┘
                  └────────────────────────┼────────────────────────┘
                                           ▼
                        ┌────────────────────────────────────┐
                        │ FILA PERSISTENTE NO POSTGRESQL     │
                        │ pistas, prioridades, leases,       │
                        │ retries, limites e retomada        │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ CRAWLER / WORKERS EM GO             │
                        │ Colly + HTTP(S) seguro             │
                        │ robots, SSRF, cotas por domínio    │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ EXTRAÇÃO SEGURA                    │
                        │ HTML / PDF textual / referências   │
                        │ Docling + OCR só quando necessário │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ NORMALIZAÇÃO E PROVENIÊNCIA        │
                        │ SHA-256, deduplicação, versões,    │
                        │ fabricante, modelo e firmware      │
                        └─────────────────┬──────────────────┘
                                          ▼
                    ┌─────────────────────┴──────────────────────┐
                    ▼                                            ▼
       ┌─────────────────────────────┐            ┌─────────────────────────────┐
       │ POSTGRESQL                  │            │ ARQUIVOS LOCAIS             │
       │ catálogo, trechos, FTS,     │            │ originais imutáveis         │
       │ claims, evidências, grafo   │            │ endereçados por SHA-256     │
       └──────────────┬──────────────┘            └──────────────┬──────────────┘
                      └────────────────────┬─────────────────────┘
                                           ▼
                        ┌────────────────────────────────────┐
                        │ MOTOR DE VERIFICAÇÃO CRUZADA       │
                        │ fontes independentes, contradições│
                        │ modelo / firmware / pré-requisitos │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ DOSSIÊ TÉCNICO COM EVIDÊNCIAS      │
                        │ comandos, páginas, URLs, riscos,   │
                        │ testes pendentes e incertezas      │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ NOVAS PISTAS RELEVANTES            │
                        │       ↖ VOLTAM À FILA ACIMA       │
                        └────────────────────────────────────┘
```

**Observações:** o acervo local também pode alimentar diretamente a verificação, sem baixar novamente documentos já preservados. A expansão recursiva só percorre fontes autorizadas e respeita cotas e políticas de acesso. A IA é opcional no núcleo: modos **sem IA**, **modelo local** ou **API externa**. SearXNG ou uma API web de descoberta funcional é necessário para ampliar a busca para domínios desconhecidos, além dos links das fontes já conhecidas.

## Implantação leve (máquina única / VM no Proxmox)

```text
┌────────────────────────────────────────────────────────────────────┐
│                       VM LINUX / SERVIDOR                         │
│                                                                    │
│  Caddy* ──► Painel Svelte + Vite  /  API Go + Huma                │
│                                          │                         │
│                                   Workers Go                      │
│                                          │                         │
│               ┌──────────────────────────┼───────────────────┐     │
│               ▼                          ▼                   ▼     │
│        PostgreSQL                 Acervo em disco      Worker OCR* │
│        FTS + pg_trgm              SHA-256 + versões    isolado     │
│               │                          │                         │
│               └────────────┬─────────────┘                         │
│                            ▼                                       │
│                    Backup restic + PG                              │
└────────────────────────────┬───────────────────────────────────────┘
                             ▼
                   CÓPIA EXTERNA / RESTAURAÇÃO

* Serviços condicionais: Caddy, SearXNG, Docling/OCR, IA local.
  Expansões futuras: pgvector, Zoekt, OpenSearch e storage S3.
```

**Infra inicial prevista:** teste econômico com 2 vCPU/4 GB; perfil recomendado com 4 vCPU/8 GB e SSD. São metas de dimensionamento a validar, não medições. Modelos de IA locais e OCR complexo exigem recursos adicionais.
