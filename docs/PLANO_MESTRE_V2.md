# PLANO MESTRE V2 — NETWORK INTELLIGENCE ENGINE

**Versão proposta:** 2.0  
**Data:** 01/11/2026  
**Status:** Arquitetura recomendada para implementação, ainda sem prova de produção.  
**Princípio orientador:** *Escolher, não somar. Instalar o mínimo, comprovar o diferencial cedo e ativar serviços pesados apenas se métricas demonstrarem necessidade real.*

---

## 1. Veredito e princípio de engenharia

A visão do projeto original é rigorosamente preservada. O objetivo **não é construir outro Google nem coletar bilhões de arquivos sem critério**. É permitir que o usuário indique o problema técnico a descobrir, enquanto a plataforma descobre onde pesquisar, segue caminhos pertinentes, verifica aplicabilidade e constrói um acervo técnico cumulativo verificável.

O diferencial a validar é a união de:
1. **Descoberta recursiva autônoma** (grafo de exploração guiado);
2. **Memória técnica permanente** (acervo cumulativo local);
3. **Evidências independentes** (distinguir cópias de confirmações reais);
4. **Especialização técnica multivendor por fabricante, modelo e firmware**.

> **Decisão V2:** Instalar o mínimo, comprovar o diferencial cedo e ativar serviços pesados apenas se métricas demonstrarem necessidade. Não confundir projeto documentado com código implementado.

---

## 2. Produto e modos visuais obrigatórios

Todas as opções de controle de campanha devem existir na UI. Arquivos de configuração devem ficar restritos à implantação, credenciais e parâmetros sensíveis do sistema — **nunca exigir edição de YAML para ajustar profundidade, fontes e prioridades operacionais**.

1. **Busca rápida:** Texto exato, comandos CLI, regex controlada, documentos, Git e filtros por fabricante, modelo, software, idioma, fonte e data.
2. **Investigação profunda:** Planejamento de subperguntas, busca federada, recursão, evidências pró/contra, resposta e dossiê citável.
3. **Exploração contínua:** Campanhas programadas, descoberta por links/menções/sitemaps/RSS, revisitas incrementais e alertas de novos documentos.
4. **Biblioteca permanente:** Documentos originais permitidos, versões, snippets, links, hashes SHA-256 e relacionamento com pesquisas anteriores.
5. **Comparador técnico:** Diferenças de comandos, firmware, documentação, pré-requisitos e modelos.
6. **Mapa de descobertas:** Grafo visual de origem dos links, ramificações, justificativas, ganhos e motivos de parada.
7. **Auditoria de evidências:** Afirmações verificáveis, conflitos e detecção de fontes derivadas da mesma origem.
8. **Administração visual:** Fontes, conectores, limites, cronogramas, fila, erros, modelos de IA, custos, backups e perfis de usuários.

---

## 3. Nova stack base — escolher, não somar

| Componente | Escolha V2 | Justificativa | Obrigatório no MVP? |
| :--- | :--- | :--- | :---: |
| **API / Coordenação / Filas / Crawler** | **Go + net/http + Huma** | Binário único, contratos fortemente tipados, documentação OpenAPI automática, módulos internos de alta performance. | **Sim** |
| **Interface (Frontend)** | **Svelte + Vite (SPA estática)** | Bundle estático servido pelo próprio binário Go; Node.js apenas durante o build. | **Sim** |
| **Banco e índice inicial** | **PostgreSQL 18** | Fonte de verdade transacional, FTS nativo (`to_tsvector`/`ts_rank_cd`), JSONB, índices B-Tree/GIN, filas seguras. | **Sim** |
| **Similaridade de palavras** | **pg_trgm** | Busca aproximada para títulos, termos técnicos e buscas parciais sem quebrar o banco. | **Sim** (via migração) |
| **Vetores** | **pgvector** | Busca vetorial e embeddings quando demonstrado ganho real de recall. | **Não** (ativável) |
| **Crawling estático** | **Go net/http seguro + Colly / HTML parser** | Filas da aplicação, limites por domínio, egress seguro contra SSRF e descoberta de links. | **Sim** |
| **Descoberta externa** | **APIs Git e buscadores adaptados** | Permite descobrir materiais fora do corpus inicial cadastrado. | **Sim** (ao menos 1 real) |
| **Metabusca** | **SearXNG** | Ampla cobertura da web pública quando disponível. | **Extensão recomendada** |
| **PDFs textuais** | **Extrator simples em processo isolado** | Baixo consumo de CPU/RAM para PDFs digitais normais; preserva páginas e texto. | **Sim** |
| **PDFs complexos / OCR** | **Docling / Tesseract em Python worker efêmero** | Contêiner isolado sem rede interna, ativado apenas para documentos escaneados/complexos. | **Não** (sob demanda) |
| **Armazenamento de arquivos** | **Filesystem content-addressed (SHA-256)** | `objects/sha256/xx/yy/<hash>` com escrita atômica + fsync. Portável, resiliente e sem S3 obrigatório. | **Sim** |
| **Backup** | **restic + pg_dump / WAL** | Cópia segura fora do disco principal com recuperação verificável. | **Sim** |
| **HTTPS / Proxy** | **Caddy** | Proxy TLS automático e simples para exposição externa. | Conforme exposição |
| **Modelos de IA** | **Adaptador local/cloud intercambiável** | Sem lock-in com fornecedor ou framework proprietário. | Para investigação semântica |

> **Nota sobre o roteador Go e Huma:** O Huma funciona diretamente sobre o roteador padrão do Go (`net/http`). `chi` só deve ser adicionado se houver ganho concreto evidente.
>
> **Condição de cobertura web:** O NIE só descobre novos domínios na web aberta sem sementes (*seeds*) manuais quando há ao menos um serviço de descoberta web funcional (ex.: SearXNG próprio ou API de pesquisa externa). No perfil econômico sem tal serviço, o crawler percorre links e referências a partir das fontes que já conhece.
>
> **Excluídos do núcleo inicial:** *OpenSearch, Elasticsearch, MinIO (repositório arquivado em 04/2026), RAGFlow, Onyx, GPT Researcher, Hyperresearch, Zoekt, Nutch, StormCrawler, Valkey, RabbitMQ, Qdrant, Kubernetes e Paperless-ngx*. Permanecem como candidatos de extensão condicionada a métricas.

---

## 4. Arquitetura física e lógica

```text
USUÁRIO → Caddy (se exposto) → Go API + painel Svelte compilado
                                 │
                   ┌─────────────┴──────────────────┐
                   │                                │
          Orquestrador/Planner                 Busca local
       campanhas, limites, políticas       FTS/exata/filtros
                   │                                │
       PostgreSQL (catálogo, filas, evidências, histórico)
                   │
          ┌────────┴────────┐
          │                 │
       Git Eye           Web Eye
  APIs/seleção repos   metabusca/APIs → filas URLs/menções
          │                 │
          └────────┬────────┘
                   ▼
        Crawler Go com egress seguro
                   │
            Extratores por tipo
      HTML leve / PDF leve / Python efêmero
                   │
          SHA-256 + versão + proveniência
                   │
    Filesystem imutável → PostgreSQL (chunks/evidências)
                   │
          Recuperação híbrida opcional
                   │
       Motor de claims + citações + dossiê
                   │
           Novas pistas → fila
```

### Separação de processos e segurança:
- **API, crawler e agendador:** Podem ser compilados no mesmo binário Go (monorepo), mas com execução em perfis de privilégios e processos separados.
- **Isolamento de parsers:** Nunca executar navegação não confiável ou parsing de arquivos arbitrários com privilégios administrativos. O worker de OCR/Python roda em contêiner descartável, sem rede interna e com sistema de arquivos somente leitura.

---

## 5. Investigação autônoma já na primeira fase

Cada campanha é disparada com uma pergunta técnica, restrições operacionais e um orçamento máximo. O motor percorre um grafo com fronteira de descoberta em 9 etapas:

1. **Interpretar:** Identificar tema, fabricante, produto, firmware, protocolo e subperguntas.
2. **Descobrir:** Consultar fontes externas permitidas, acervo local e resultados de campanhas passadas.
3. **Priorizar:** Pontuar candidatos por relevância temática, novidade, probabilidade de evidência, independência e custo estimado.
4. **Coletar:** Requisitar URLs públicas com políticas rígidas de domínio, robots.txt e segurança de egress.
5. **Extrair:** Capturar texto limpo, títulos, links, referências a manuais, versões e snippets de código/CLI.
6. **Expandir:** Cada link ou menção relevante vira uma nova pista na fila (`discovered_from` + justificativa auditável).
7. **Confrontar:** Relacionar claims com evidências, avaliar independência de origens e checar compatibilidade de versões.
8. **Persistir:** Gravar bytes imutáveis (SHA-256), metadados no Postgres, logs de execução e justificativas de recusa.
9. **Encerrar ou revisitar:** Concluir quando o orçamento esgotar ou a novidade decair; agendar revisita periódica se configurado.

### Orçamentos iniciais padrão por campanha:

| Parâmetro | Limite de Partida |
| :--- | :--- |
| **Profundidade recursiva** | Máximo de 3 níveis |
| **URLs tentadas** | Até 200 URLs |
| **Domínios distintos** | Até 20 domínios |
| **Concorrência por domínio** | 1 conexão simultânea |
| **Taxa de requisição** | No máximo 6 requisições/minuto (reduzir sob sinais de saturação) |
| **Tamanho máximo de HTML** | 5 MiB por recurso |
| **Tamanho máximo de PDF** | 50 MiB por recurso |
| **Volume total baixado** | Até 1 GiB por campanha |
| **Retries transitórios** | Até 3 tentativas com backoff exponencial e jitter |
| **Custo de IA** | Teto orçamentário configurável (zero com IA desativada) |

---

## 6. Motor de evidências — impedir que três cópias virem três provas

O motor modela o conhecimento através de entidades relacionais estritas:
* `Claim` (afirmação técnica isolada e testável)
* `Evidence` (trecho específico com citação exata de página/offset)
* `OriginCluster` (agrupamento de fontes derivadas da mesma fonte original)
* `DocumentVersion` (versão capturada vinculada a um hash SHA-256)
* `Passage` (bloco textual de contexto)
* `Contradiction` (registro explícito de divergência entre fontes)
* `Vendor`, `DeviceModel`, `FirmwareRelease`, `Command`, `ValidationRun`, `DiscoveryEdge`

### Estados epistemológicos separados:
1. **Status Epistemológico:** Sem evidência suficiente | Sustentado por documento | Corroborado por origens independentes | Contestado.
2. **Aplicabilidade Técnica:** Modelo e versão confirmados | Incompatível | Indeterminado.
3. **Validação Operacional:** Não testado | Testado em laboratório autorizado | Falhou no teste.

> **Exemplo mandatório de telecom:** A continuidade de sessões PPPoE em alta disponibilidade de BNG/BRAS não pode ser inferida a partir de redundância de BGP, failover de RADIUS ou VRRP. São propriedades funcionais independentes que exigem evidências separadas.

---

## 7. Busca híbrida para comandos CLI e termos técnicos

O motor adota camadas complementares de recuperação de informação:
* **Busca Exata:** Comparação canônica literal de comandos CLI, caminhos de menu, OIDs de SNMP, códigos de erro e blocos de configuração. Preserva símbolos (`/`, `:`, `-`, `"`, espaços e quebras).
* **Busca Lexical Documental:** PostgreSQL `to_tsvector` com `ts_rank` e `ts_rank_cd`, segmentada por idioma e tipologia de documento.
* **Busca Aproximada:** `pg_trgm` para tolerância a pequenos erros de digitação em títulos, modelos e nomes técnicos.
* **Busca Semântica (Opcional):** `pgvector` acionado quando embeddings forem expressamente homologados via benchmark de recall.
* **Busca em Código:** APIs de Git e clone seletivo no MVP; Zoekt reservado para quando o acervo de código atingir dezenas de gigabytes.
* **Fusão de Resultados:** Reciprocal Rank Fusion (RRF) combinando pontuações com validação posterior dos filtros de firmware e hardware.

---

## 8. Segurança e conformidade como restrições do sistema

* **Prevenção contra SSRF e DNS Rebinding:** O crawler em Go aceita estritamente conexões HTTP/HTTPS públicas. A resolução de IP (IPv4/IPv6) valida e rejeita loopback (`127.0.0.0/8`), IPs privados (RFC 1918), link-local (`169.254.0.0/16`), metadados de nuvem e multicast antes do estabelecimento do socket TCP. A validação é repetida obrigatoriamente a cada redirecionamento HTTP.
* **Respeito a Robots.txt e Termos de Uso:** Implementação estrita da RFC 9309, respeitando diretivas de bloqueio, taxas de requisição e cabeçalhos `Retry-After`. Resposta com status 429 ou 403 coíbe o acesso ao domínio; jamais contornar bloqueios via rotação de proxy.
* **Tratamento de Dados Externos:** Todo conteúdo HTML, PDF, Markdown ou texto de fórum é tratado como dado não confiável. O modelo de IA nunca recebe permissão para disparar comandos em equipamentos ou acessar credenciais internas.
* **Arquivos Seguros:** Proibição absoluta de execução de scripts, binários ou firmwares baixados. Limites contra ZIP/PDF bombs, recursão de links e alocação desmedida de memória.
* **Sanitização de Segredos:** Dados sensíveis (senhas, hashes, chaves de API, dados pessoais) são censurados antes de serem gravados ou enviados a modelos externos.

---

## 9. Dados, armazenamento e resiliência

* **PostgreSQL:** Fonte única da verdade para campanhas, filas, metadados, chunks, claims, evidências e arestas do grafo.
* **Filesystem Imutável:** Padrão `objects/sha256/aa/bb/<hash>` para armazenamento dos bytes originais. Gravação temporária + `fsync` + `rename` atômico; coleta de lixo segura com janela de tolerância para órfãos.
* **Filas Transacionais:** `SELECT ... FOR UPDATE SKIP LOCKED` com locação (*lease*), expiração de prazo, idempotência, limites de retries e tratamento de dead-letter jobs.
* **Backup e Recuperação:** `restic` para os arquivos em disco e `pg_dump` ou arquivamento de WAL para o banco, com rotina de teste de restauração periódica.

---

## 10. Inteligência e modelos de IA sem lock-in

O sistema opera com interfaces e contratos agnósticos:
* `ResearchPlanner`: Decompõe consultas técnicas complexas em subperguntas.
* `DiscoveryProvider`: Interface unificada para adaptadores de busca (Web, Git, Acervo).
* `Fetcher` / `Extractor`: Interface para obtenção e conversão de formatos brutos.
* `EvidenceResolver`: Cruza afirmações com trechos documentais e detecta contradições.
* `ReportWriter`: Gera o dossiê técnico estruturado com referências auditáveis.

**Modos de IA na UI:**
1. *Modo Sem IA:* Executa buscas exatas, filtros determinísticos e expansão por links/menções normalmente.
2. *Modo IA Local:* Conexão com Ollama / vLLM em infraestrutura própria com placa de vídeo dedicada.
3. *Modo API Externa:* OpenAI, Anthropic, Google Gemini ou provedores compatíveis com OpenAPI.

---

## 11. Funcionalidades da interface (Svelte + Vite)

* **Interface Central:** Painel limpo com busca rápida universal e modo de investigação profunda.
* **Acompanhamento de Campanha em Tempo Real:** Visualização do objetivo, fontes consultadas, grafo de ramificação das URLs descobertas, justificativas de prioridade e consumo de cotas.
* **Navegação com Citação Precisa:** Clicar em qualquer trecho de um dossiê abre o documento original imutável na página e parágrafo exatos.
* **Exportação:** Geração de relatórios completos em formato Markdown (com indicação expressa de lacunas e incertezas não comprovadas).

---

## 12. Critérios para expansões futuras (Escalabilidade por evidência)

Nenhum componente pesado será instalado antecipadamente sem métricas que comprovem a necessidade:

| Sintoma Medido | Extensão a Avaliar | Pré-condição Obrigatória |
| :--- | :--- | :--- |
| Busca em repositórios Git ultrapassar os limites de tempo/recursos | **Zoekt** | Demonstrar ganho concreto em relação ao ripgrep e APIs Git. |
| Busca documental no PostgreSQL ficar lenta ou índice inviável em disco | **OpenSearch** | Comprovar gargalo de latência e consumo em corpus de grande escala. |
| Busca puramente textual perder recall em conceitos de telecom | **pgvector** | Validar ganho medido em benchmark com embeddings específicos. |
| Fila do Postgres apresentar saturação extrema de I/O em escrita | **NATS JetStream** | Comprovar gargalo de throughput e adotar padrão outbox. |
| Necessidade de storage distribuído para múltiplos nós de servidores | **SeaweedFS / S3** | Testar durabilidade, compatibilidade com backup e facilidade de manutenção. |
| Crawling massivo em nível de web aberta tornar-se indispensável | **StormCrawler** | Comprovar retorno de investimento e taxa de páginas raras encontradas. |

---

## 13. Roteiro de execução (Fases 0 a 4)

### Fase 0 — Baseline e dados de referência
- Montar o conjunto de 20 perguntas técnicas reais de telecom + documentos gabarito (gold standard).
- Homologar versões, compatibilidade e licenças das bibliotecas do ecossistema Go/Svelte/Postgres.
- Definir as métricas de aceitação para busca manual vs. automatizada.

### Fase 1 — Vertical Slice Real (MVP Funcional)
- API Go com Huma + painel SPA básico em Svelte.
- Banco de dados PostgreSQL 18 com migrações iniciais.
- Fila persistente com `SKIP LOCKED` e crawler Go com proteção de egress (SSRF).
- Extrator simples de HTML e PDF textual.
- Armazenamento de arquivos endereçado por SHA-256.
- Execução de ponta a ponta: *busca → descoberta de link → extração → armazenamento → evidência citável → nova pista*.
- Busca exata por comandos e visualizador de trecho original.

### Fase 2 — Motor de evidências e investigação profunda
- Implementação das entidades `Claim`, `Evidence`, `OriginCluster` e detecção de cópias.
- Verificação de compatibilidade por modelo e firmware de equipamento.
- Integração de adaptadores de IA com fallback e geração de dossiês em Markdown.

### Fase 3 — Autonomia contínua e observabilidade
- Agendamento de campanhas recorrentes e revisitas com `ETag` / `Last-Modified`.
- Visualização interativa do grafo de descoberta no Svelte.
- Observabilidade de taxas, quotas, erros e retenção de arquivos.

### Fase 4 — Ampliações condicionadas
- Avaliação de Zoekt, OpenSearch ou storage distribuído somente mediante necessidade comprovada por métricas.

---

## 14. O que está deliberadamente fora do MVP

1. Clonagem massiva de toda a internet ou espelhamento completo de GitHub/GitLab.
2. Varreduras em redes Tor e metadados de torrent no núcleo inicial.
3. Qualquer capacidade de enviar comandos ou interagir com roteadores de produção.
4. Orquestração complexa com Kubernetes ou múltiplos bancos vetoriais redundantes.
5. Treinamento de modelos de linguagem do zero.

---

## 15. Diagramas da arquitetura V2

### Visão Geral dos Componentes:

```text
                        ┌────────────────────────────────────┐
                        │      USUÁRIO (PC / CELULAR)        │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ PAINEL SVELTE + VITE               │
                        │ Busca, campanhas, grafo, biblioteca│
                        │ Caddy/HTTPS se acesso externo      │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │       NÚCLEO GO + HUMA             │
                        │ API, orquestrador, autenticação,   │
                        │ planejamento, cotas e políticas   │
                        └─────────────────┬──────────────────┘
                                          │
                    ┌─────────────────────┼──────────────────────┐
                    ▼                     ▼                      ▼
       ┌──────────────────────┐ ┌──────────────────────┐ ┌──────────────────────┐
       │ GIT EYE              │ │ WEB EYE              │ │ BUSCA NO ACERVO      │
       │ GitHub/GitLab APIs   │ │ SearXNG / APIs       │ │ PostgreSQL FTS       │
       │ Repositórios úteis   │ │ Links, fóruns, RSS   │ │ pg_trgm + filtros    │
       └──────────┬───────────┘ └──────────┬───────────┘ └──────────┬───────────┘
                  └────────────────────────┼────────────────────────┘
                                           ▼
                        ┌────────────────────────────────────┐
                        │ FILA PERSISTENTE NO POSTGRESQL     │
                        │ Pistas, prioridades, leases,       │
                        │ retries, limites e retomada        │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ CRAWLER / WORKERS EM GO            │
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
       │ Catálogo, trechos, FTS,     │            │ Originais imutáveis         │
       │ claims, evidências, grafo   │            │ endereçados por SHA-256     │
       └──────────────┬──────────────┘            └──────────────┬──────────────┘
                      └────────────────────┬─────────────────────┘
                                           ▼
                        ┌────────────────────────────────────┐
                        │ MOTOR DE VERIFICAÇÃO CRUZADA       │
                        │ Fontes independentes, contradições│
                        │ modelo / firmware / pré-requisitos │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ DOSSIÊ TÉCNICO COM EVIDÊNCIAS      │
                        │ Comandos, páginas, URLs, riscos,   │
                        │ testes pendentes e incertezas      │
                        └─────────────────┬──────────────────┘
                                          ▼
                        ┌────────────────────────────────────┐
                        │ NOVAS PISTAS RELEVANTES            │
                        │       ↖ VOLTAM À FILA ACIMA       │
                        └────────────────────────────────────┘
```

### Topologia de Implantação em Servidor Único (VM Proxmox / Baremetal):

```text
┌────────────────────────────────────────────────────────────────────┐
│                       VM LINUX / SERVIDOR                         │
│                                                                    │
│  Caddy* ──► Painel Svelte + Vite  /  API Go + Huma                 │
│                                          │                         │
│                                   Workers Go                       │
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
```
