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

## ADR-009 — Arquitetura genérica; telecom como especialização

**Status:** aceito (implementação parcial)  
**Contexto:** o objetivo original é investigar qualquer assunto público; redes
e ISPs são a especialização inicial. Se a descoberta depender de termos como
fabricante, BGP ou firmware, o NIE vira apenas um buscador de telecom.  
**Decisão:** fila, coleta, CAS, busca, menções sem link e dossiê devem
funcionar para qualquer tema. O vocabulário de telecom entra como camada
adicional, nunca como condição para a descoberta.  
**Estado em 02/11/2026:**
- genérico e testado: menções de CVE, DOI, arXiv e normas (ISO, IEC, IEEE,
  ITU-T, NIST, ABNT) geram novas consultas (`internal/worker/worker_test.go`);
  priorização da fronteira usa os termos da própria campanha; o dossiê omite
  rollback/checklist de redes quando não há fabricante de rede;
- ainda específico de telecom: `internal/classifier`, `evidence.ParseRequirements`
  (requisitos de fabricante, modelo, firmware e recurso) e os padrões de
  modelos e protocolos em `internal/extract/mentions.go`. Fora de telecom, o
  motor de evidências não extrai requisitos e as afirmações ficam
  `documented`/`unconfirmed` com aplicabilidade `indeterminate`.  
**Consequências:** um benchmark fora de telecom deve integrar a validação de
ponta a ponta. Tornar os requisitos do motor de evidências configuráveis por
domínio é trabalho pendente.

## ADR-010 — O PostgreSQL manda na fila; executores só coletam

**Status:** aceito  
**Contexto:** a fila persistente, a fronteira de URLs, a dedupe e os orçamentos
já vivem no PostgreSQL. Usar também o scheduler do Scrapy criaria duas
estruturas concorrentes de prioridade, deduplicação e recuperação.  
**Decisão:** o NIE decide o que coletar e em que ordem; Scrapy, Playwright (e
Crawl4AI, se for integrado) executam uma coleta por tarefa e devolvem dados.
Aproveitar `LinkExtractor`, `CrawlSpider` ou middlewares do Scrapy é permitido,
desde que a fronteira continue no PostgreSQL.  
**Consequências:** o worker Scrapy atual (que usa `requests` e o
`LinkExtractor`) é coerente com esta decisão. Não se espera migrar para o
scheduler/`JOBDIR` do Scrapy.

## ADR-011 — Regras de evidência, rodadas e proveniência (auditoria 3)

**Status:** aceito (implementado em 02/11/2026)  
- **Evidência:** `ResolveClaimStatus` avalia cada evidência isoladamente. Só
  `supports` sustenta; `mentions` é relevância lexical. `sustained/applicable`
  exige uma evidência oficial que comprove, ela mesma, fabricante, modelo,
  firmware e recurso. Duas origens independentes com escopo comprovado dão
  `corroborated/applicable`. Um trecho que apenas lista os termos não afirma
  nada e vira `mentions`.
- **Rodadas:** `EnqueueDiscovery` é idempotente por campanha+provedor+consulta
  normalizada e reabre a campanha concluída; pesquisas externas têm teto próprio
  e não consomem `max_urls`.
- **Persistência:** claims são upsert por (campanha, título); `origin_clusters`
  e `cluster_id` são gravados na mesma transação.
- **Proveniência Git:** um SHA só é citado se o trecho existe no blob daquele
  commit; o resto é `working_tree`.
- **Originais:** o documento é arquivado no CAS antes de qualquer extração; sem
  texto, fica `extraction_pending`.  
**Limites conhecidos:** a classificação ainda é lexical; o ciclo adaptativo
continua manual; PostgreSQL 18, navegador real, isolamento de rede dos workers,
backup/restauração e a campanha de ponta a ponta continuam sem prova.
