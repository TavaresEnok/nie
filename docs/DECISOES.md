# Registro de decisões — NIE

Formato curto de ADR. A fonte canônica de arquitetura continua sendo
`PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md`; este arquivo registra o que
foi efetivamente implementado.

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

**Status:** aceito  
**Decisão:** não usar Colly no primeiro slice. O cliente HTTP resolve o host,
valida o IP e disca diretamente para o IP validado, revalidando redirects.  
**Consequências:** controle explícito de SSRF/DNS rebinding; recursos de
robots/rate limit são internos e testáveis.

## ADR-005 — IA fora do MVP

**Status:** aceito  
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
