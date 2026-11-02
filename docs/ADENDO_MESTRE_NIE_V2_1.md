# ADENDO MESTRE V2.1 — Network Intelligence Engine

**Reintegração dos projetos open source já identificados e preservação do rastreamento recursivo**  
**Data da revisão:** 02/11/2026  
**Status:** proposta arquitetural revisada; **nenhuma integração, desempenho ou benchmark de execução está declarado como concluído**.  
**Documentos de precedência:** `ARQUIVO_MESTRE.md` original (30/10), `PLANO_MESTRE_NETWORK_INTELLIGENCE_ENGINE_V2.md` (01/11) e arquivos originais no ZIP.

## 1. Por que existe este adendo

A V2 acertou ao reduzir a infraestrutura obrigatória, mas simplificou demais o motor de coleta: substituiu o arsenal de crawlers maduros pelo caminho Go/Colly e adiou quase todas as capacidades preexistentes. Isso pode elevar o código próprio e atrasar o diferencial da plataforma. **Recomenda-se preservar a leveza do control plane em Go e reutilizar, como workers especializados, sistemas de crawling comprovadamente existentes.**

O catálogo do backup possui **59 registros**; isso inclui produtos, APIs, índices, acervos, fóruns, fontes de documentação e referências históricas. Não representa 59 softwares independentes que devam ser instalados. O CSV anexado classifica cada registro. Os anexos textuais ainda citam ferramentas fora do catálogo principal, como **Repowise, TorBot, TorCrawl, Jackett, Prowlarr, SpiderFoot, Meilisearch e Recoll**: manter em backlog histórico, não aprovar automaticamente.

### Princípios preservados

1. O usuário define **o assunto**, não cada site.
2. O sistema deve descobrir por busca externa, links, PDF, README, RSS, sitemap, nomes sem link e citações; seguir ramificações e registrar relações.
3. O fluxo é contínuo e recuperável, mas sempre **limitado por relevância, segurança, permissões e recursos**.
4. Nenhum software pode garantir toda a web, todos os repositórios, todos os onion services ou dados privados.
5. Três URLs copiadas de uma mesma origem não são três evidências independentes.
6. A biblioteca pertence ao NIE (PostgreSQL + arquivos), não a um componente terceirizado.
7. Ferramentas maduras devem **ser aproveitadas por interfaces estáveis**, em vez de reescritas ou simplesmente empilhadas.

## 2. Escolha V2.1 — arquitetura forte, com serviços graduais

| Camada | Principal / decisão | Por quê |
|---|---|---|
| Control plane, API e direitos | **Go + Huma** | API leve, coordenação de campanhas, proteção, contrato de integração |
| Interface | **Svelte + TypeScript + Vite** | Grafo de descobertas, filas, fontes, política, evidências, biblioteca |
| Fonte de verdade | **PostgreSQL** | Campanhas, fronteira global de URLs, tarefas, relacionamentos, evidências, políticas |
| Descoberta externa | **SearXNG + GitHub/GitLab** | Oferece seeds sem exigir domínios pré-definidos |
| Descoberta complementar | **Mwmbl, Software Heritage, ecosyste.ms, Wayback, Common Crawl** | Fontes independentes, históricos e metadados; adaptadores habilitáveis |
| Crawling HTTP geral | **Scrapy**, executado em worker Python | `CrawlSpider`, links, filtros, prioridade, autotuning, pausar/retomar |
| Sites complexos e pesquisa orientada | **Crawl4AI**, isolado e invocado sob demanda | Deep crawling BFS/DFS/Best-First; prefetch, JS, transformação de páginas em Markdown |
| Escala contínua distribuída | **Apache StormCrawler**, como alternativa de execução futura | Apache Storm, modular, streaming e política de robots; não operar concorrente com Scrapy sobre as mesmas campanhas |
| Alternativas massivas/arquivo | **Nutch / Heritrix**, após prova de necessidade | Nutch batch/focused crawl; Heritrix captura WARC de qualidade arquivística |
| Busca em código | **Zoekt + APIs Git** | Pesquisa exata/regex nos repositórios selecionados; não requer clonar GitHub inteiro |
| Extração de arquivos | **Apache Tika e Docling**, por tipo e necessidade | Tika para formatos diversos; Docling para PDF/tabelas/OCR avançado |
| Arquivamento selecionado | **ArchiveBox** ou captura WARC própria | Reprodução e preservação permitidas; não substituir storage canônico |
| Pesquisa no acervo | **PostgreSQL FTS + pg_trgm** | Simplicidade inicial; índices avançados adicionados conforme carga |
| Search em grande escala | **OpenSearch**, por benchmark | Busca BM25, vetores, filtros para acervos maiores |
| Análise profunda | **GPT Researcher / Hyperresearch**, pilotos concorrentes | Reutilizar subpesquisas e auditoria de fontes sem impor dois agentes permanentes |
| RAG | **Pipeline próprio ou RAGFlow**, comparação experimental | Evitar que duas fontes de verdade governem o mesmo acervo |
| Armazenamento | **Disco local imutável por SHA-256 + restic**; adaptador S3 no futuro | Sem serviço pesado por padrão; storage S3 quando escala justificar |

**Importante:** escolher dois motores de crawl com *especializações diferentes* não significa que ambos devem percorrer a mesma URL. Para cada tarefa, o orquestrador seleciona **um único executor** (Scrapy, Crawl4AI ou StormCrawler). A fronteira de URLs, os hashes e as relações continuam no PostgreSQL e todas as saídas passam pelo mesmo contrato.

## 3. A “árvore que cresce sozinha”: reutilizar o que já existe

### Três operações distintas

**1. Encontrar sementes inéditas**: SearXNG, GitHub/GitLab, Mwmbl, índices históricos e links previamente coletados.  
**2. Percorrer caminhos**: Scrapy/Crawl4AI/StormCrawler seguem links de páginas e alimentam o índice de candidatos.  
**3. Aumentar inteligência**: motor de relevância e LLM opcional transformam referências, nomes citados sem hyperlinks, lacunas e descobertas em novas consultas.

Um crawler capaz de seguir links não cria sozinho uma investigação técnica confiável: precisa de **pontuação por relevância, rastreamento da origem da descoberta, exclusão de loops e orçamento por campanha**. Por outro lado, não se deve escrever um `CrawlSpider` inteiro do zero em Go quando essa capacidade já existe.

### Exemplo conceitual

```text
    Pergunta: NE8000 + redundância PPPoE + BGP
                             │
                             ▼
            SearXNG / Git / Mwmbl / acervo
                             │
                  URLs e pistas iniciais
                             ▼
        ┌──────── FRONTEIRA GLOBAL (PostgreSQL) ────────┐
        │ URL, origem, campanha, prioridade, profundidade│
        └──────────────────────┬─────────────────────────┘
                               ▼
                  ROTEADOR DE EXECUTORES
                   │          │          │
                   ▼          ▼          ▼
                 Scrapy    Crawl4AI   StormCrawler*
               HTML/link   JS/deep     escala futura
                   └──────────┼──────────┘
                              ▼
                    LINKS + MENÇÕES + PDFs
                              │
           ┌──────────────────┼───────────────────┐
           ▼                  ▼                   ▼
     Manual oficial     Fórum independente     Repositório Git
           │                  │                   │
           ▼                  ▼                   ▼
      Comandos VRP     Relatos e contexto       Configuração
           └──────────────────┼───────────────────┘
                              ▼
                  DEDUPE + CLASSIFICAÇÃO
                              ▼
                   EVIDÊNCIAS / CLAIMS
                              ▼
               DOSSIÊ + ACERVO PERMANENTE
                              │
           lacunas, novos links, versões e nomes
                              └─────────► FRONTEIRA

    * habilitar quando escala medida justificar
```

### Contrato dos executores

O motor deve trabalhar com tipos internos compartilhados, por exemplo:

- `DiscoveryCandidate`: URL canônica, documento de origem, tipo de descoberta (link, sitemap, menção, citação, API), campanha, profundidade, score, razões.
- `FetchTask`: destino permitido, modo (HTTP, browser, Git, arquivo), deadline, limite de bytes, política de domínio, identidade de campanha.
- `FetchResult`: URL efetiva, status HTTP, conteúdo bruto, metadados, tempo/custo e relações extraídas.
- `DocumentVersion`: hash, parser/versão, timestamps, localização e licença; preservação seletiva.
- `DiscoveryEdge`: `from_document_id`, `to_url_or_document_id`, método, confiança de extração, timestamp e campanha.
- `Evidence`: claim ID, documento e versão, página/offset, apoio/refutação, derivação da fonte e escopo técnico.

O crawler retorna **dados**, nunca autorização para executar ações ou instalar ferramentas.

## 4. Decisões específicas para evitar redundância

| Dupla ou grupo | Escolher | Não fazer |
|---|---|---|
| Scrapy / Crawl4AI | Scrapy para HTTP normal, Crawl4AI para deep crawling/JS focalizado | Duas requisições de ambos ao mesmo URL sem necessidade |
| Scrapy / StormCrawler / Nutch | Scrapy no início; StormCrawler ou Nutch quando benchmark determinar padrão de escala | Três fronteiras autônomas independentes |
| Zoekt / OpenGrok / Livegrep / Hound | Zoekt para busca Git; OpenGrok só se referências cruzadas agregarem valor | Quatro índices iguais |
| PostgreSQL / OpenSearch / Solr | PostgreSQL como verdade; OpenSearch como índice derivado opcional | Dois motores documentais redundantes sem benchmark |
| pgvector / Qdrant | Começar com nenhum ou pgvector; Qdrant só se volume exigir | Embeddings e catálogos duplicados |
| GPT Researcher / Hyperresearch | Pilotos sobre a mesma bateria de testes, com qualidade, custo e permissões | Instalar dois sistemas de agentes obrigatórios |
| RAGFlow / Onyx / Open WebUI / AnythingLLM | No máximo um RAG pronto, e só se melhorar sobre pipeline próprio | Quatro interfaces divergentes |
| ArchiveBox / Heritrix | ArchiveBox em arquivamento seletivo; Heritrix se houver acervo arquivístico de grande porte | Downloads indiscriminados de tudo |
| YaCy / Mwmbl / SearXNG | SearXNG e Mwmbl como fontes; YaCy como índice próprio opcional | Tratar três mecanismos como três verificações independentes de fato |

## 5. Segurança e legalidade incorporadas na mecânica

**Crawler:** respeitar robots.txt, termos, `429`/`Retry-After`, limites por domínio, timeout, content-type e permissões. Bloquear SSRF (IPs internos/metadata/loopback/link-local, IPv4 e IPv6), DNS rebinding, redirecionamentos para destinos proibidos, downloads excessivos, bombs de compactação e loop de calendário/paginação. Impor barreira de egress em rede, além de validação de URL. Workers de browser/parsing com privilégios limitados, sem acesso à rede de gerenciamento.

**Pesquisa com IA:** HTML, README, fórum e PDF são entradas não confiáveis: bloquear prompt injection por permissões reais e separação de papéis. Nunca fornecer segredos, execução shell, credenciais ou acesso a equipamentos de produção ao agente pesquisador. Respostas devem referenciar trechos reais e registrar hipóteses não confirmadas.

**Arquivamento:** guardar íntegra somente quando permissão/licença permitir; senão, manter metadados, referência e citações curtas permitidas. Redigir dados pessoais e credenciais antes de indexar ou expor a APIs externas. Permitir pedidos de exclusão/retificação.

**Orçamento:** limites por campanha, tipo de documento, domínio, custo IA, número de páginas, tempo de execução e profundidade. Deduplicação global de URL e conteúdo; revisita incremental e `ETag`/`Last-Modified` quando disponíveis; cancelamento e recuperação após falhas.

## 6. Como medir se reaproveitar motores é realmente melhor

O catálogo de **20 casos técnicos** do arquivo mestre será preservado. Acrescentar ao benchmark testes de comportamento:

1. Descobrir domínios que o usuário não indicou manualmente a partir de buscas externas.
2. Seguir cinco níveis de links artificiais sem loops e sem ultrapassar orçamento.
3. Extrair URL de PDF e README, e gerar busca por menção **sem hyperlink**.
4. Buscar caminhos por sitemap/RSS e registrar arestas de descoberta.
5. Priorizar material oficial e relevante em vez de milhões de páginas genéricas.
6. Percorrer fórum público com paginação permitida, respeitando robots e limites.
7. Preservar a campanha ao reiniciar um worker e não repetir download desnecessário.
8. Demonstrar que 3 republicações de um texto não viram 3 origens independentes.
9. Provar correspondência de comando por fabricante, modelo e release.
10. Bloquear URLs internas, redirecionamentos maliciosos, links infinitos e instruções hostis na página.

Comparar *recall de documentos relevantes, novas fontes úteis por 100 downloads, taxa de duplicação, latência, RAM pico, CPU, custo por campanha, precisão de citações e reprodutibilidade*. Não afirmar desempenho de qualquer motor sem execução no hardware real.

## 7. Infraestrutura em perfis, não 20 serviços fixos

**Perfil de prova, a dimensionar:** 4 vCPU, 8 GB RAM, SSD 160–250 GB, volume de arquivos separado. Go + Svelte estático + PostgreSQL + um Scrapy worker + SearXNG; sem LLM local, Docling pesado, Chromium ou outros índices sempre ligados. Ajustar após medição.

**Perfil avançado, a dimensionar:** 8 vCPU, 16–32 GB RAM, SSD NVMe, disco para acervo. Usar Crawl4AI/Docling isoladamente sob demanda, Zoekt seletivo e recursos de IA remotos/local dimensionados separadamente.

**Perfil distribuído:** definir por benchmark de URLs/dia, número de domínios, retenção, índices e latências antes de adicionar StormCrawler, Nutch, OpenSearch e storage multinó. Não há números universais de consumo que substituam teste representativo.

## 8. Estado dos projetos verificados em 02/11/2026

| Projeto | Registro objetivo e fonte oficial |
|---|---|
| Scrapy | `CrawlSpider` extrai e segue links definidos por regras; `JOBDIR` persiste requests/visitas/estado para pausa/retomada. https://docs.scrapy.org/en/master/topics/spiders.html e /topics/jobs.html |
| Crawl4AI | Deep crawling BFS, DFS, Best-First; prefetch e modo adaptativo (estatístico ou embeddings). É preciso habilitar respeito a robots conforme configuração. https://docs.crawl4ai.com/core/deep-crawling/ |
| Apache StormCrawler | v3.7.0 nas docs, streaming distribuído, módulos OpenSearch/Solr/SQL; requer armazenamento de URLs para recursão. https://stormcrawler.apache.org/docs/ |
| Apache Nutch | Release 1.23 em 08/09/2026; JDK 17 e Hadoop 3.5.0 se usar cluster. https://nutch.apache.org/news/nutch-1.23-release/ |
| Heritrix | Changelog da versão 3.17.1 em 10/09/2026. https://github.com/internetarchive/heritrix3/blob/master/CHANGELOG.md |
| Crawlee | RequestQueue para recursão; enqueueLinks nos contexts. https://crawlee.dev/js/api/core/class/RequestQueue |
| YaCy | crawler próprio, índice, API e rede P2P. https://github.com/yacy/yacy_search_server |
| Hyperresearch | vault permanente, verificações de citações/independência e execuções retomáveis, mas o formato de harness/workflow requer prova de integração. https://github.com/jordan-gibbs/hyperresearch |
| GPT Researcher | planejamento e pesquisa profunda, com suporte a relatórios; testar como executor de investigação, não crawler universal. https://github.com/assafelovic/gpt-researcher |
| Sourcebot | Fonte oficial anunciou desenvolvimento fechado a partir da v6 em 08/10/2026. https://www.sourcebot.dev/blog/sourcebot-is-now-closed-source |
| LangChain Open Deep Research | Repositório arquivado em 21/08/2026. https://github.com/langchain-ai/open_deep_research |
| MinIO (repo público) | Arquivado em 25/04/2026. https://github.com/minio/minio |

**Estas verificações não constituem auditoria de código, licença de cada dependência, segurança de deploy ou teste de integração.** Respeitar licenças de versões exatas antes de fork, distribuição ou SaaS.

## 9. Prioridades executáveis

**P0 — desde o protótipo:** Go + PostgreSQL + Svelte, SearXNG, APIs Git, Scrapy recursivo, fronteira persistente, limites, proveniência, hash, busca textual e dossiê com citações.

**P1 — integração funcional no primeiro ciclo:** Crawl4AI sob demanda, Docling/Tika por tipo, Zoekt para corpus Git selecionado, Mwmbl/Wayback/Software Heritage/Common Crawl como adaptadores, ArchiveBox opcional e exploração por menções.

**P2 — pilotos sem compromisso de lock-in:** GPT Researcher versus Hyperresearch; RAGFlow versus RAG próprio; YaCy para índice independente, Heritrix para preservação; OpenSearch se o PostgreSQL não atingir SLO.

**P3 — prova de escala:** StormCrawler ou Nutch como motor de execução distribuído (escolher um pelo workload), ampliar workers e índices.

**P4 — opcionais com justificativa e controles próprios:** Ahmia para material onion público e bitmagnet para metadados DHT legítimos; evitar recursos sem demanda e autorização. Nenhuma dessas redes garante cobertura universal.

## 10. Critérios de aceite antes de dizer 'poderoso'

O primeiro protótipo somente estará demonstrado se, para um tema de redes escolhido, o sistema:

- encontrar ao menos um domínio que não foi informado diretamente pelo usuário;
- seguir ramificações de links além da primeira página e capturar de onde cada descoberta veio;
- produzir nova busca derivada de menção sem link;
- localizar texto verificável em um documento e atribuir versão/página/URL;
- distinguir cópias de uma mesma evidência e apontar incertezas;
- permitir pausar, retomar e cancelar sem perder o estado;
- respeitar limites de campanha/domínio e falhar de forma segura diante de links maliciosos.

A V2.1 **não substitui** os arquivos históricos. Ela corrige o subaproveitamento de tecnologias maduras, mantendo o compromisso com pesquisa verificável, segurança, infraestrutura enxuta e crescimento medido.
