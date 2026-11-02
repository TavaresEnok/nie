# NIE — Network Intelligence Engine

MVP auto-hospedável para **descobrir, coletar, preservar e buscar** documentação
técnica com proveniência verificável. O foco inicial é telecom/ISPs: comandos de
CLI, configurações, RFCs, datasheets, READMEs e notas de versão.

Cada resultado de busca aponta para o documento original, o trecho exato, a
página (quando PDF) e o hash SHA-256 do conteúdo armazenado.

> **Status de Auditoria e Conformidade:**
> O NIE adota transparência estrita entre o que está implementado, o que foi testado em laboratório e o que está planejado para fases futuras.

## Matriz de Conformidade e Status dos Requisitos

| Requisito / Componente | Escopo | Status Real | Evidência / Testes |
|---|---|---|---|
| **Go 1.24+ / Huma / pgxpool** | Framework API & HTTP | **Implementado** | `cmd/nie/main.go`, `internal/api/api.go` |
| **PostgreSQL 18 + SKIP LOCKED** | Fila e Storage | **Implementado & Testado** | `internal/store/store.go`, `store_integration_test.go` |
| **Proteção Anti-SSRF & Netguard** | Egress Efetivo (P0-03) | **Implementado & Testado** | `internal/netguard/ip.go`, `internal/crawler/router.go`, workers Python |
| **Validação Canônica de Domínio Oficial** | Anti-Spoofing (P0-02) | **Implementado & Testado** | `internal/classifier/classifier.go`, `classifier_test.go` |
| **Motor de Evidências Epistemológico** | Rigor Técnico (P0-01) | **Implementado & Testado** | `internal/evidence/engine.go`, `engine_test.go` (sem rótulos infundados de 'verificado') |
| **Origin Clustering Anti-Cópia** | Thread-Safe (P1-06) | **Implementado & Testado** | `internal/evidence/clustering.go` (`sync.RWMutex`, N réplicas = 1 cluster) |
| **Preservação de Todas as Arestas** | Grafo Multi-Origens (P1-03) | **Implementado & Testado** | `internal/store/store.go`, `TestP103_MultipleOriginsPreserveEdges` |
| **Advisory Lock por Conexão Dedicada** | Migração Segura (P1-02) | **Implementado & Testado** | `internal/store/migrate.go`, `TestP102_ConcurrentMigrations` |
| **Fronteira com Focused Crawling Dinâmico** | Priorização por Score (P1-04) | **Implementado & Testado** | `internal/worker/worker.go` (`computeFrontierPriority`) |
| **Grafo Visual Interativo com Multi-Origens** | UI de Descoberta (P1-04) | **Implementado** | `web/src/App.svelte` (visualização de ligações `A->C`, `B->C` e convergência) |
| **Agendamento com Fase de Descoberta** | Scheduler Autônomo (P1-01) | **Implementado** | `internal/scheduler/scheduler.go` (dispara adaptadores mesmo sem seeds) |
| **Configuração SearXNG JSON** | API SearXNG (P0-04) | **Implementado** | `config/searxng/settings.yml`, `docker-compose.yml` |
| **Git Eye / Descoberta Local e Remota** | Código Técnico (P1-05) | **Implementado & Testado** | `internal/discovery/git_local.go`, `git_local_test.go` (.cfg, .rsc rastreados) |
| **Workers Scrapy e Crawl4AI com Egress Guard** | Bridges Isoladas (P0-03) | **Implementado & Protegido**| `workers/scrapy_worker/app.py`, `workers/crawl4ai_worker/app.py` |
| **Docling / Tika / OCR Avançado** | Extração Especializada | **Parcial / Planejado** | Extração nativa Go de PDF/HTML ativa; OCR sob demanda planejado para worker dedicado |
| **Busca Semântica com Embeddings (pgvector)** | Vetores e Reranker | **Planejado para V3** | Atualmente exact + FTS tsvector + trigramas pg_trgm ativo no PostgreSQL |

## O que já funciona

- **Campanhas** com pergunta, seeds, orçamento, pausa, retomada e agendamento contínuo.
- **Descoberta** por links internos recursivos, SearXNG (opcional) e GitHub (opcional).
- **Grafo Visual de Descoberta** na UI representando as ramificações em árvore ("água e eletricidade").
- **Crawler seguro** com fila persistente no PostgreSQL 18 (`SKIP LOCKED`), robots.txt e proteção anti-SSRF/DNS rebinding.
- **Coleta e extração** de HTML, texto e PDFs digitais (com fallback OCR).
- **Storage content-addressed** imutável por SHA-256 com escrita atômica.
- **Classificador Telecom Multivendor** (Huawei VRP, MikroTik RouterOS v6/v7, Cisco IOS-XR/XE, Datacom DmOS, ZTE, FiberHome, VSOL, SonicWall, Hillstone).
- **Origin Clustering & Anti-Cópia** (agrupa blogs e espelhos duplicados para não inflar confirmações falsas).
- **Motor de Claims e Auditoria Epistemológica** (`sustained`, `corroborated`, `contested`, `unconfirmed`).
- **Investigação com IA & Dossiê Técnico** estruturado (matriz de compatibilidade, comandos CLI com proveniência, rollback e checklist).
- **Busca Híbrida** exata (preserva pontuação de CLI), lexical (`tsvector`), trigramas (`pg_trgm`) e busca em Git local.
- **Agendamentos e Alertas** de mudança de firmware/documentação técnica.
- **Painel Svelte moderno** embutido diretamente no binário Go.
- **Docker Compose** pronto para subir com PostgreSQL 18 e aplicação.

## Arquitetura

```text
┌──────────────┐     HTTP/JSON      ┌──────────────────────────────┐
│ Painel Svelte│ ─────────────────▶ │ API Go + Huma (net/http)     │
└──────────────┘                    │  campanhas, busca, docs, jobs│
                                    └──────────────┬───────────────┘
                                                   │ pgx
                                    ┌──────────────▼───────────────┐
                                    │ PostgreSQL 18                │
                                    │ fila SKIP LOCKED, FTS, trgm  │
                                    └──────────────▲───────────────┘
                                                   │
┌──────────────────────────────────────────────────┴───────────────┐
│ Worker Go                                                        │
│ claim → throttle/robots → fetch seguro → extrair → SHA-256       │
│       → documento/versão/trechos → enfileirar novos links        │
└───────────────────────────────┬──────────────────────────────────┘
                                │ bytes imutáveis
                     ┌──────────▼──────────┐
                     │ objects/sha256/...  │
                     └─────────────────────┘
```

Decisões registradas em [`docs/DECISOES.md`](docs/DECISOES.md).

## Subir com Docker Compose

Requisitos: Docker Engine/Desktop 29+ com Compose v2.

```bash
docker compose up --build
```

Abra:

- Painel: <http://127.0.0.1:8080>
- Documentação da API: <http://127.0.0.1:8080/api/docs>
- OpenAPI: <http://127.0.0.1:8080/api/openapi.json>

O Compose cria dois volumes:

- `nie_pgdata`: banco PostgreSQL.
- `nie_data`: objetos SHA-256 e dados do runtime.

Para encerrar sem perder dados:

```bash
docker compose down
```

Para apagar também banco e objetos (irreversível):

```bash
docker compose down -v
```

O PostgreSQL fica exposto apenas em `127.0.0.1:5432`. A aplicação também escuta
apenas no loopback por padrão. A senha padrão do banco é `nie`; para alterar,
crie um `.env` a partir de `.env.example` e defina `NIE_DB_PASSWORD`.

## Desenvolvimento local

Requisitos: Go 1.27+, Node 24+, PostgreSQL 18+.

1. Suba o PostgreSQL (ou use um existente):

   ```bash
   docker run -d --name nie-postgres \
     -e POSTGRES_USER=nie -e POSTGRES_PASSWORD=nie -e POSTGRES_DB=nie \
     -p 127.0.0.1:5432:5432 \
     -v nie_pgdata:/var/lib/postgresql \
     postgres:18-alpine
   ```

2. Configure o ambiente e rode a API/worker:

   ```bash
   export NIE_DATABASE_URL='postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable'
   export NIE_DATA_DIR='./data'
   export NIE_ADDR='127.0.0.1:8080'
   go run ./cmd/nie
   ```

   No PowerShell:

   ```powershell
   $env:NIE_DATABASE_URL='postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable'
   $env:NIE_DATA_DIR='./data'
   $env:NIE_ADDR='127.0.0.1:8080'
   go run ./cmd/nie
   ```

3. Frontend em modo dev (opcional; o binário usa o build embutido):

   ```bash
   cd web
   npm ci
   npm run dev
   ```

   O Vite faz proxy de `/api` para `http://127.0.0.1:8080`.

4. Build de produção do SPA e do binário:

   ```bash
   cd web && npm ci && npm run build && cd ..
   go build -o bin/nie ./cmd/nie
   ./bin/nie
   ```

## Configuração

Todas as variáveis são opcionais; os padrões funcionam para uso local.

| Variável | Padrão | Descrição |
|---|---|---|
| `NIE_DATABASE_URL` | `postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable` | DSN do PostgreSQL |
| `NIE_ADDR` | `127.0.0.1:8080` | Endereço HTTP |
| `NIE_DATA_DIR` | `./data` | Diretório de objetos e temporários |
| `NIE_WORKER` | `true` | Liga/desliga o worker neste processo |
| `NIE_WORKERS` | `2` | Número de goroutines de coleta |
| `NIE_POLL_INTERVAL` | `2s` | Intervalo de polling da fila |
| `NIE_REQUEST_TIMEOUT` | `30s` | Timeout por requisição HTTP |
| `NIE_MAX_HTML_BYTES` | `5242880` | Limite por resposta HTML |
| `NIE_MAX_PDF_BYTES` | `52428800` | Limite por resposta PDF |
| `NIE_MAX_TEXT_BYTES` | `5242880` | Limite por resposta de texto |
| `NIE_RATE_LIMIT_PER_MINUTE` | `6` | Default de intervalo por domínio (6 req/min = 10 s) |
| `NIE_SEARXNG_URL` | vazio | URL do SearXNG para descoberta web |
| `NIE_GITHUB_ENABLED` | `false` | Habilita descoberta via GitHub |
| `NIE_GITHUB_TOKEN` | vazio | Token para elevar rate limit do GitHub |
| `NIE_API_TOKEN` | vazio | Se definido, exige `X-API-Token` ou `Bearer` nas rotas `/api/` privadas |
| `NIE_WEB_DIR` | vazio | Diretório alternativo para o SPA em desenvolvimento |
| `NIE_USER_AGENT` | `NIEBot/0.1 (+https://localhost)` | User-Agent e token de `robots.txt` |
| `NIE_LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error` |

Limites de **campanha** (profundidade, URLs, domínios, bytes e intervalo por
domínio) são definidos na UI/API, não por variáveis de ambiente.

## API

Exemplos com `curl`. Se `NIE_API_TOKEN` estiver definido, acrescente
`-H 'X-API-Token: SEU_TOKEN'`. Healthcheck, metadados, documentação, OpenAPI e
schemas permanecem públicos; as rotas de dados exigem o token.

### Healthcheck e capacidades

```bash
curl http://127.0.0.1:8080/api/health
curl http://127.0.0.1:8080/api/meta
```

### Criar campanha

```bash
curl -X POST http://127.0.0.1:8080/api/campaigns \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "configuração de BGP em NE8000",
    "seed_urls": ["https://example.com/"],
    "max_depth": 2,
    "max_urls": 50,
    "max_domains": 10,
    "max_bytes": 104857600,
    "min_domain_interval_seconds": 5
  }'
```

Sem seeds, habilite `use_web_discovery` (requer SearXNG) ou
`use_git_discovery` (requer GitHub habilitado).

### Acompanhar e controlar

```bash
curl http://127.0.0.1:8080/api/campaigns
curl http://127.0.0.1:8080/api/campaigns/SEU_ID
curl -X POST http://127.0.0.1:8080/api/campaigns/SEU_ID/pause
curl -X POST http://127.0.0.1:8080/api/campaigns/SEU_ID/resume
curl http://127.0.0.1:8080/api/campaigns/SEU_ID/edges
curl 'http://127.0.0.1:8080/api/jobs?campaign_id=SEU_ID&status=pending'
```

### Buscar

```bash
curl 'http://127.0.0.1:8080/api/search?q=display%20bgp%20peer'
curl 'http://127.0.0.1:8080/api/search?q=ip%20route%2010.0.0.0%2F8&limit=10'
curl 'http://127.0.0.1:8080/api/search?q=bgp&campaign_id=SEU_ID'
```

A resposta traz `match` (`exact`, `fts` ou `fuzzy`), `snippet` com destaque,
`url`, `page`, `ordinal`, `sha256` e `fetched_at`.

### Biblioteca

```bash
curl 'http://127.0.0.1:8080/api/documents?limit=20'
curl 'http://127.0.0.1:8080/api/documents?campaign_id=SEU_ID'
curl http://127.0.0.1:8080/api/documents/ID
```

## Como o fluxo funciona

1. A criação da campanha grava seeds como jobs `fetch` e, opcionalmente, jobs de
   descoberta.
2. O worker reivindica jobs com `FOR UPDATE SKIP LOCKED` e lease de 2 minutos.
   Jobs `running` com lease expirado voltam a ser elegíveis, com incremento de
   tentativa e limite de `max_attempts`.
3. Antes de buscar, o worker aplica o throttle por domínio e consulta o
   `robots.txt` do host (cache de 1 h; 401/403/429 e 5xx bloqueiam
   temporariamente).
4. O cliente HTTP resolve o host, valida todos os IPs e disca **diretamente**
   para um IP público; redirects são revalidados. Proxy de ambiente é ignorado.
5. O conteúdo é limitado por tipo, extraído e armazenado por SHA-256. Somente
   HTML, texto e PDF digital são aceitos.
6. Documento, versão, trechos e arestas de descoberta são gravados em uma
   transação. Links elegíveis viram novos jobs dentro do orçamento.
7. A busca combina `strpos` case-insensitive com `ts_rank_cd` e `pg_trgm`,
   deduplicando por trecho.

## Segurança

- Bloqueio de loopback, redes privadas, link-local, CGNAT, multicast, faixas de
  documentação/reservadas e endpoints de metadata.
- Resolução DNS + conexão ao IP validado (mitiga DNS rebinding); redirects
  revalidados; sem proxy de ambiente.
- `robots.txt` respeitado; 401/403/429 e falhas 5xx tratados como bloqueio
  temporário.
- Conteúdo externo é dado não confiável: nada é executado. O SPA não usa
  `innerHTML` de conteúdo coletado; snippets têm apenas o `<mark>` gerado pelo
  servidor e o texto é renderizado em `<pre>`.
- Headers de segurança no SPA (CSP, `nosniff`, `DENY`, `no-referrer`).
- Token único opcional; API escuta no loopback por padrão.
- Limites de tamanho e tempo por requisição.

Riscos residuais do MVP: parser de PDF roda no mesmo processo do worker (sem
sandbox/OCR), não há RBAC multiusuário, e a política de egress depende do
sistema operacional e da rede do host.

## Testes

```bash
go test ./...
go vet ./...
cd web && npm run build
```

O teste de integração do store usa um PostgreSQL real e é ignorado por padrão:

```bash
export NIE_TEST_DATABASE_URL='postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable'
go test ./internal/store -run TestStoreIntegration -v
```

Ele cobre migrações, normalização de seeds, documento/versão/trechos, busca
exata/FTS/aproximada, fila, arestas, throttle e estatísticas.

## Estrutura

```text
cmd/nie/                 bootstrap, HTTP server, worker
internal/api/            rotas Huma, auth e handlers
internal/config/         variáveis de ambiente
internal/crawler/        HTTP seguro, SSRF, robots.txt
internal/discovery/      SearXNG e GitHub
internal/domain/         tipos de domínio
internal/extract/        HTML, texto e PDF
internal/netguard/       política de IP público
internal/storage/        storage content-addressed
internal/store/          PostgreSQL, fila, busca e migrações
internal/urlutil/        normalização e hash de URL
internal/worker/         processamento da fila
internal/webui/          SPA embutido
web/                     Svelte 5 + Vite
docs/                    especificação e decisões
```

## Limitações e próximos passos

- Sem OCR para PDFs escaneados e sem execução de JavaScript.
- Sem extração dedicada de DOCX/XLSX/PPTX ou arquivos compactados.
- Sem grafo visual completo, comparador de equipamentos ou claims formais.
- Sem multiusuário/RBAC, métricas Prometheus ou tracing distribuído.
- Próxima fase sugerida: métricas, fixtures públicas de crawler, OCR isolado e
  camada de evidências/claims.
