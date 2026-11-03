# NIE — Network Intelligence Engine

MVP auto-hospedável para **descobrir, coletar, preservar e buscar** documentação
técnica com proveniência verificável. O foco inicial é telecom/ISPs: comandos de
CLI, configurações, RFCs, datasheets, READMEs e notas de versão.

Cada resultado de busca aponta para o documento original, o trecho exato, a
página (quando PDF) e o hash SHA-256 do conteúdo armazenado.

> **Como ler o status.** *Testado* = coberto por teste automatizado que roda
> neste repositório (unidade ou integração). *Sem teste* = código existe, mas
> nenhum teste o exercita. *Não validado E2E* = nenhuma campanha real de ponta a
> ponta foi registrada. Nenhum item abaixo foi validado em campanha real.

## Matriz de status

| Componente | Status | Evidência / limitação |
|---|---|---|
| **Motor de evidências** (P0-01) | Testado (regras + conjunto rotulado) | Análise por oração: um recurso só é sustentado pela oração que o afirma e só é refutado pela oração que o nega (negar Telnet não refuta BGP); recursos pedidos juntos precisam aparecer afirmados na mesma oração. Firmware só conta quando a release é citada explicitamente (o modelo NE8000 não implica VRP v8). Negação em PT/EN/ES refuta; condicional/parcial e listas são menção; outro fabricante ou release não prova nem refuta. Refutação só vale com fabricante e recurso pedidos. Conjunto rotulado de 15 frases em `assertion_test.go`. Continua sendo análise lexical, não compreensão da proposição |
| **Schema de claims ↔ motor** (P0-02) | Testado com PostgreSQL 16 | Migração `0005`; `TestP002_SaveClaimsWithDocumentedAndApplicableSchema`; a rota `/investigate` falha com 500 se não persistir |
| **Fila PostgreSQL + `SKIP LOCKED` + lease** | Testado com PostgreSQL 16 | `internal/store/store_integration_test.go` (requer `NIE_TEST_DATABASE_URL`); PostgreSQL 18 não testado |
| **Integridade do original no CAS** (P1-04) | Testado | Workers guardam a entidade HTTP descomprimida no charset de origem (gzip, ISO-8859-1, `<meta charset>`); DOM renderizado segue como derivado (`RenderedContent`). `workers/test_workers_ssrf.py`, `internal/crawler/router_test.go` |
| **Limites de bytes por documento** | Testado | Teto por documento (`NIE_MAX_HTML_BYTES`/`NIE_MAX_PDF_BYTES`) repassado aos workers e aplicado ao corpo descomprimido; resposta JSON dos workers limitada |
| **Worker Scrapy** | Parcial | Usa `HtmlResponse`/`LinkExtractor` do Scrapy para extrair links; a coleta é `requests`, sem o scheduler/`CrawlSpider` do Scrapy |
| **Worker de navegador** (`crawl4ai_worker`) | Parcial, browser sem teste | Playwright + BeautifulSoup; **a biblioteca Crawl4AI não é usada**. Só o fallback HTTP é testado |
| **SSRF no fetcher Go** | Testado | `internal/netguard`, conexão ao IP validado e revalidação de redirects |
| **SSRF nos workers Python** | Parcial | Validação de destino e `page.route` no navegador, testados em unidade; não há isolamento de rede/egress no Compose, então DNS rebinding não está coberto. Chromium roda com `--no-sandbox` |
| **Limites de recursos dos workers** | Configurado, não validado | `mem_limit`, `cpus`, `pids_limit`, `no-new-privileges` no `docker-compose.yml` |
| **Roteamento entre executores** (P1-01) | Testado (heurística) | Escolha por padrões de URL (`/app/`, `/docs/`…); não detecta páginas JS por conteúdo |
| **Sitemap / RSS / Atom** | Testado | `internal/extract/feed.go`: `urlset`, `sitemapindex`, RSS, Atom e `<link rel="alternate">`; arestas `sitemap`/`feed`. Sitemaps do `robots.txt` ainda não são lidos |
| **Menções sem link** | Testado (unidade) | `internal/extract/mentions.go`: referências genéricas (CVE, DOI, arXiv, ISO/IEC/IEEE/ITU-T/NIST/ABNT) e vocabulário de telecom geram consultas de descoberta |
| **Priorização da fronteira** (P1-05) | Testado (unidade) | `computeFrontierPriority`: termos da campanha antes, páginas de ruído (login, share…) depois |
| **Origin clustering** (P1-06) | Testado com PostgreSQL 16 | Fingerprint de conteúdo persistido em `origin_clusters`; evidências gravam `cluster_id`, que sobrevive ao recarregamento. Fingerprint idêntico = 1 origem; independência real de autoria não é inferida |
| **Ciclo adaptativo de pistas** | Implementado e testado com descoberta HTTP controlada; sem E2E público | Fila ociosa agenda avaliação persistente. Lacunas de claims geram buscas primeiro; quando elas fecham, a rodada envia subperguntas determinísticas ainda não pesquisadas (`PlanSubquestions`, sem LLM no ciclo). Cobertura por termos no mesmo trecho não certifica a resposta. Histórico em `/api/campaigns/{id}/rounds` e subperguntas em `/api/campaigns/{id}/questions`. |
| **Git Eye** (P1-07) | Parcial | Varredura local com proveniência verdadeira: `Revision` só é um SHA se o trecho existe no blob daquele commit; alteração não commitada ou arquivo não rastreado vem como `working_tree`/`uncommitted`/`dirty` com `base_revision`. Busca de repositórios e READMEs no GitHub; sem busca por conteúdo remoto, GitLab ou Zoekt |
| **Claims idempotentes** | Testado com PostgreSQL 16 | Índice único (campanha, título). A migração 0006 **mescla** duplicatas antigas no claim validado (ou no mais antigo): evidências e contradições são movidas, nada é apagado, e cada fusão fica em `claim_merges`. Upsert seguro sob concorrência; claims com validação operacional são imutáveis |
| **robots.txt** | Testado (unidade + servidor local) | RFC 9309: robots.txt inalcançável bloqueia por 1 min; 404/410 libera; 5xx bloqueia. Cada redirecionamento é conferido contra o robots.txt do destino; o redirecionamento do próprio robots.txt não reentra na verificação |
| **Orçamento de bytes** | Testado com PostgreSQL 16 | Reserva atômica no PostgreSQL (`byte_reservations`) antes de cada download: 12 reservas simultâneas nunca excedem o saldo. O teto concedido é aplicado em todos os executores, inclusive o nativo. Reservas expiram se o worker cair; saldo ocupado adia o job em vez de descartá-lo |
| **Original preservado sem texto** | Testado com PostgreSQL 16 | Documento sem texto extraível vai para o CAS como `extraction_pending`; uma extração posterior do mesmo SHA-256 preenche texto e trechos sem nova coleta. Reprocessar com outro motor um documento que já tem trechos (versões de extração) ainda não é suportado. OCR/Docling não existem |
| **Orçamento de pesquisas** | Testado com PostgreSQL 16 | Pesquisas externas (máx. 50 por campanha) não consomem `max_urls`, que conta só downloads |
| **Recuperação para o dossiê** | Testado (unidade) | A pergunta é decomposta em consultas curtas (fabricante, modelo, firmware, recurso) e os resultados são fundidos; recall@k nunca foi medido |
| **Tipos de aresta** | Testado com PostgreSQL 16 | Arestas `sitemap` e `feed` são preservadas no grafo |
| **Links em PDFs** (P1-08) | Testado (unidade) | URLs encontradas no texto; anotações de link embutidas no PDF não são lidas |
| **Dossiê** | Testado (unidade) | Toda afirmação contestada, só documentada ou sem evidência suficiente recebe advertência; o resumo informa quando nenhuma conclusão é sustentada |
| **Arquitetura genérica** (ADR-009) | Parcial | Descoberta e dossiê funcionam fora de telecom; classificador e requisitos do motor de evidências ainda são de telecom |
| **Benchmark de 20 perguntas** | Testado com fixtures sintéticas | Mede o avaliador, não a pesquisa real na internet |
| **SearXNG** | Configurado | `config/searxng/settings.yml` (JSON); resultados reais não testados |
| **OCR** | Não integrado | `internal/extract/ocr.go` existe, mas o pipeline não o chama |
| **Busca semântica (pgvector)** | Planejado | Hoje: exata + FTS `tsvector` + `pg_trgm` |

Decisões de arquitetura vigentes: [`docs/DECISOES.md`](docs/DECISOES.md).

## O que já funciona

- **Campanhas** com pergunta, seeds, orçamento, pausa, retomada e agendamento contínuo.
- **Descoberta** por links recursivos, sitemaps, RSS/Atom, menções sem link, SearXNG (opcional), Mwmbl, GitHub e Git local.
- **Roteador de executores** por padrão de URL (navegador Playwright para `/app/` e afins, worker Scrapy para `/docs/` e afins, Go nativo no restante).
- **Grafo Visual de Descoberta** na UI representando as ramificações em árvore e convergência multi-origens.
- **Crawler seguro** com fila persistente no PostgreSQL (`SKIP LOCKED`), robots.txt e proteção anti-SSRF/DNS rebinding no fetcher Go; os workers Python validam destinos, mas dependem de isolamento de rede para DNS rebinding.
- **Coleta e extração** de HTML, texto e PDFs digitais com extração de citações e hyperlinks embutidos.
- **Storage content-addressed (CAS)** imutável por SHA-256 do conteúdo bruto original com escrita atômica.
- **Classificador Telecom Multivendor** (Huawei VRP, MikroTik RouterOS v6/v7, Cisco IOS-XR/XE, Datacom DmOS, ZTE, FiberHome, VSOL, SonicWall, Hillstone).
- **Origin Clustering & Anti-Cópia** (agrupa espelhos e reposts para não inflar confirmações falsas).
- **Motor de Evidências Epistemológico** estruturado (`sustained`, `corroborated`, `contested`, `documented`, `unconfirmed`) com verificação atômica de requisitos.
- **Investigação com IA & Dossiê Técnico** estruturado (matriz de compatibilidade, comandos CLI com proveniência, rollback e checklist).
- **Busca Híbrida** exata (preserva pontuação de CLI), lexical (`tsvector`), trigramas (`pg_trgm`) e busca em Git local com hash de commit.
- **Agendamentos** de campanhas (a detecção de mudança em documentos ainda não está ligada ao fluxo).
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

Ao esvaziar a fila, `NIE_AUTOMATIC_RESEARCH=true` agenda a avaliação das evidências
e permite buscas derivadas das lacunas encontradas. `NIE_MAX_RESEARCH_ROUNDS=3`
limita o total de avaliações, incluindo a final (máximo aceito: 10; zero desativa).
O histórico de decisões fica na campanha e em `GET /api/campaigns/{id}/rounds`.
Nenhuma lacuna detectada significa somente que as regras atuais não propuseram
outra busca; não comprova cobertura completa do assunto. Veja
[o registro de execução do Plano V3](docs/EXECUCAO_PLANO_V3.md).

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
