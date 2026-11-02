# Registro de decisões — NIE

Formato curto de ADR. A arquitetura de referência é
[`arquitetura/PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md`](arquitetura/PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md),
complementada por [`arquitetura/ADENDO_MESTRE_NIE_V2_1.md`](arquitetura/ADENDO_MESTRE_NIE_V2_1.md)
onde o adendo diverge. Este arquivo registra o que foi efetivamente
implementado; quando um ADR for substituído, o original permanece com status
atualizado.

## ADR-001 — PostgreSQL 18 como fonte única de verdade

**Status:** aceito  
**Contexto:** o MVP precisa de fila, catálogo, FTS e proveniência sem operar
Elasticsearch/OpenSearch.  
**Decisão:** usar PostgreSQL 18 com `pg_trgm`, `tsvector` e `SKIP LOCKED`.  
**Consequências:** um único serviço de dados; índices reconstruíveis; migração
para OpenSearch apenas com métrica de gargalo.

## ADR-002 — Filesystem content-addressed antes de S3

**Status:** aceito  
**Decisão:** armazenar bytes em `objects/sha256/aa/bb/<hash>` com temp + fsync +
rename atômico; hash é a identidade do conteúdo.  
**Consequências:** portabilidade e backup simples; coleta de órfãos fica para
uma fase posterior.

## ADR-003 — Huma sobre `net/http`

**Status:** aceito  
**Decisão:** usar o adaptador `humago` e o `http.ServeMux` padrão.  
**Consequências:** menos uma dependência de roteador; OpenAPI automático.

## ADR-004 — Crawler próprio com egress seguro

**Status:** parcialmente substituído pelo ADR-007 (o crawler nativo continua
como executor padrão e fallback)  
**Decisão:** não usar Colly no primeiro slice. O cliente HTTP resolve o host,
valida o IP e disca diretamente para o IP validado, revalidando redirects.  
**Consequências:** controle explícito de SSRF/DNS rebinding; recursos de
robots/rate limit são internos e testáveis.

## ADR-005 — IA fora do MVP

**Status:** substituído pelo ADR-008  
**Decisão:** descoberta e busca funcionam sem LLM; nenhum contrato de IA é
implementado antes do benchmark de 20 perguntas.  
**Consequências:** menor custo e superfície de ataque; investigação semântica
fica limitada nesta fase.

## ADR-006 — Extração PDF em processo, com limites

**Status:** aceito para o MVP, reavaliar depois  
**Decisão:** PDFs digitais usam biblioteca Go pura com limite de 50 MiB; OCR e
PDFs complexos ficam fora.  
**Consequências:** simplicidade operacional; um PDF malicioso ainda deve ser
tratado como dado não confiável e o worker pode ser isolado em fase futura.

## ADR-007 — Executores especializados (Scrapy, Crawl4AI) atrás de um roteador

**Status:** aceito (substitui parcialmente o ADR-004)  
**Contexto:** o [`ADENDO_MESTRE_NIE_V2_1.md`](arquitetura/ADENDO_MESTRE_NIE_V2_1.md) apontou que reescrever todo o
crawling em Go desperdiça projetos maduros. O control plane, a fronteira de
URLs e a proveniência continuam no Go + PostgreSQL.  
**Decisão:** `internal/crawler/router.go` define a interface `Executor` e
escolhe **um único** executor por tarefa (`SelectExecutor`):
- `native` (Go, ADR-004): padrão e fallback de qualquer falha dos workers;
- `scrapy` (`workers/scrapy_worker`, `NIE_SCRAPY_URL`): catálogos e manuais
  (`/docs/`, `/manual/`, `/wiki/`, `/reference/`, `/catalog/`);
- `crawl4ai` (`workers/crawl4ai_worker`, `NIE_CRAWL4AI_URL`, perfil Compose
  `advanced`): modo `browser` e SPAs. **Implementado com Playwright +
  BeautifulSoup; a biblioteca Crawl4AI ainda não é usada** (deep/adaptive
  crawling pendente). O nome do executor reflete o papel previsto na V2.1.

Os workers são serviços HTTP sem estado; todas as saídas voltam no contrato
`domain.FetchResult`. `RawContent` (de `raw_b64`) é a entidade HTTP
descomprimida no charset de origem e é o que vai para o CAS (ADR-002);
`RenderedContent` (DOM pós-JavaScript), texto e Markdown são derivados. O Go
repassa um teto **por documento** (`NIE_MAX_HTML_BYTES`/`NIE_MAX_PDF_BYTES`,
nunca o orçamento total da campanha) e limita o JSON devolvido pelo worker.  
**Consequências:** a validação de egress dos bridges (`validatePublicTarget`)
resolve DNS no Go, mas a conexão é feita pelo worker; a proteção contra DNS
rebinding depende dos guardas no próprio worker (`page.route` no Crawl4AI,
`workers/test_workers_ssrf.py`) e de isolamento de rede. StormCrawler/Nutch
continuam fora até haver benchmark de escala.

## ADR-008 — IA opcional, desligada por padrão

**Status:** aceito (substitui o ADR-005)  
**Contexto:** a Fase 2 exigiu planejamento de subperguntas e dossiê
estruturado; o princípio de que busca e descoberta não dependem de LLM foi
mantido.  
**Decisão:** `internal/ai` expõe um adaptador com `NIE_AI_PROVIDER` =
`none` (padrão), `ollama` ou `openai`. Com `none` ou sem `NIE_AI_BASE_URL`, o
planejamento e o dossiê usam regras determinísticas (classificador de
fabricante); o LLM só é chamado quando configurado explicitamente.  
**Consequências:** nenhum custo nem tráfego externo por padrão. Conteúdo
coletado é não confiável: o modelo não recebe ferramentas, segredos nem
capacidade de execução. O ganho da IA ainda precisa ser medido no benchmark
de 20 perguntas (`internal/benchmark/20_perguntas_telecom.json`) antes de ser
declarado.
