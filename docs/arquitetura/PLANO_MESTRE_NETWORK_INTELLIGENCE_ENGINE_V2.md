# PLANO MESTRE V2 — NETWORK INTELLIGENCE ENGINE

**Versão proposta:** 2.0 · **Data:** 01/11/2026 · **Status:** arquitetura recomendada para implementação, ainda sem prova de produção.

## 1. Veredito e princípio de engenharia

A visão do projeto original é preservada. O objetivo não é construir outro Google nem coletar bilhões de arquivos sem critério. É permitir que o usuário indique **o problema a descobrir**, enquanto a plataforma descobre **onde pesquisar**, segue caminhos pertinentes, verifica aplicabilidade e constrói um acervo técnico cumulativo verificável.

O diferencial a validar é a união de **descoberta recursiva autônoma + memória permanente + evidências independentes + especialização técnica por fabricante/modelo/firmware**.

**Decisão V2:** instalar o mínimo, comprovar o diferencial cedo e ativar serviços pesados apenas se métricas demonstrarem necessidade. Não confundir projeto documentado com código implementado.

## 2. Produto e modos visuais obrigatórios

1. **Busca rápida:** texto exato, comandos, regex controlada, documentos, Git e filtros por fabricante, modelo, software, idioma, fonte e data.
2. **Investigação profunda:** planejamento de subperguntas, busca federada, recursão, evidências pró/contra, resposta e dossiê citável.
3. **Exploração contínua:** campanhas programadas, descoberta por links/menções/sitemaps/RSS, revisitas e alertas de documentos novos.
4. **Biblioteca permanente:** documentos originais permitidos, versões, snippets, links, hashes e relacionamento com pesquisas anteriores.
5. **Comparador técnico:** diferenças de comandos, firmware, documentação, pré-requisitos e modelos.
6. **Mapa de descobertas:** grafo visual de origem dos links, ramificações, justificativas, ganhos e motivos de parada.
7. **Auditoria de evidências:** afirmações verificáveis, conflitos e fontes derivadas da mesma origem.
8. **Administração visual:** fontes, conectores, limites, cronogramas, fila, erros, modelos IA, custos, backups e perfis de usuários.

Todas as opções usuais de campanha devem existir na UI. Arquivos de configuração devem ficar restritos a implantação, credenciais e parâmetros sensíveis; nunca exigir edição de YAML para ajustar profundidade, fontes e prioridades.

## 3. Nova stack base — escolher, não somar

| Componente | Escolha V2 | Justificativa | Obrigatório no MVP? |
|---|---|---|---|
| API/coordenação/filas/crawler | **Go + net/http + Huma** | Um binário, validação de contratos, OpenAPI; módulos internos | Sim |
| Interface | **Svelte + Vite (SPA estática)** | Bundle estático; Node somente no build | Sim |
| Banco e índice inicial | **PostgreSQL 18** | Fonte transacional, FTS, JSONB, índices, filas | Sim |
| Similaridade de palavras | **pg_trgm** | Ajuda títulos, termos e algumas buscas substring | Sim, habilitar por migração |
| Vetores | **pgvector** | Busca vetorial quando embeddings forem úteis | Não, recurso ativável |
| Crawling estático | **Go net/http seguro + Colly/HTML parser** | Filas da aplicação, limites por domínio, descoberta de links | Sim |
| Descoberta externa | **APIs Git e buscadores adaptados** | Descobre fora do corpus local | Sim, ao menos 1 adaptador real |
| Metabusca | **SearXNG** | Mais cobertura quando disponível | Extensão recomendada, não serviço inevitável |
| PDFs textuais | **Extrator simples por processo isolado** | Baixo consumo para PDFs comuns; preservar páginas | Sim |
| PDFs complexos/OCR | **Docling/Tesseract em Python worker efêmero** | Só executar em documentos que exigirem | Não, sob demanda |
| Arquivos | **Filesystem content-addressed (SHA-256)** | Sem serviço S3 obrigatório; portável | Sim |
| Backup | **restic + pg_dump/WAL conforme escala** | Cópia fora do disco principal e recuperação verificável | Sim |
| HTTPS | **Caddy** | Proxy TLS simples para acesso externo | Conforme exposição |
| Modelo de IA | **Adaptador local/cloud, substituível** | Não amarrar a um provedor ou framework | Para investigação semântica, não para busca rápida |

**Huma** funciona diretamente sobre o roteador padrão do Go; `chi` pode ser acrescentado somente se houver ganho concreto. É melhor evitar uma segunda pilha HTTP sem necessidade. Sua versão atual requer Go moderno (consultar README na implantação).

**Condição de cobertura:** o NIE só descobre novos domínios na web aberta sem seeds manuais quando há **ao menos um serviço de descoberta web funcional** (por exemplo, SearXNG próprio ou uma API de pesquisa autorizada). No perfil econômico, sem tal serviço, o sistema ainda percorre links, feeds e referências a partir de fontes que conhece, mas **não pode prometer descoberta global**. No perfil recomendado, habilitar SearXNG como primeira opção e medir disponibilidade/limites; manter fallback por provedores substituíveis.

**Não instalar no núcleo inicial:** OpenSearch, Elasticsearch, MinIO, RAGFlow, Onyx, GPT Researcher, Hyperresearch, Zoekt, Nutch, StormCrawler, Valkey, RabbitMQ, Qdrant, Kubernetes, Paperless-ngx. Eles permanecem **candidatos de extensão**, com critérios explícitos.

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

**Separação de processos:** API, crawler e trabalhadores de processamento podem usar o mesmo binário/monorepo Go, mas **execuções com permissões diferentes**. Não executar navegação ou parsing não confiável no processo/API com acesso privilegiado. Python/OCR/browser em contêiner isolado, sem rede interna ou acesso à API de administração.

## 5. Investigação autônoma já na primeira fase

Cada campanha começa com pergunta, restrições técnicas e orçamento. O motor executa um grafo **com fronteira de descoberta**:

1. **Interpretar:** identificar tema, fabricante, produto, firmware, protocolo e perguntas subordinadas.
2. **Descobrir:** consultar fontes externas permitidas, acervo próprio e resultados anteriores.
3. **Priorizar:** pontuar recursos pela relevância, novidade, evidência potencial, independência e custo previsto.
4. **Coletar:** buscar URLs públicas com políticas de domínio, robots e segurança de rede.
5. **Extrair:** capturar texto, títulos, links, citações bibliográficas, referências a manuais, versões e snippets.
6. **Expandir:** cada link/menção útil vira nova pista, inclusive referências sem hyperlinks, e possui `discovered_from` e justificativa.
7. **Confrontar:** relacionar claims com evidências, verificar independência e compatibilidade.
8. **Persistir:** gravar conteúdo permitido, hashes, logs, versão, razões da prioridade e decisões de recusa.
9. **Encerrar/revisitar:** parar quando custo/novidade/qualidade não justifica continuar; agendar revisita se autorizado.

**O motor inicial pode usar scoring determinístico, sem LLM:** textos-chave, classificação de tipo, correspondência de fabricante/firmware, URL, histórico de fonte e diferença de hash. IA pode melhorar a expansão sem se tornar requisito para que as filas funcionem.

### Orçamentos iniciais configuráveis (valores de partida, não garantias)

| Limite por campanha inicial | Valor proposto |
|---|---:|
| Profundidade recursiva | 3 níveis |
| URLs tentadas | 200 |
| Domínios diferentes | 20 |
| Concorrência por domínio | 1 |
| Taxa por domínio | no máximo 6 req/min, reduzir conforme política/sinais |
| HTML bruto por recurso | 5 MiB |
| PDF bruto por recurso | 50 MiB |
| Volume total baixado | 1 GiB |
| Retries transitórios | até 3, com backoff exponencial e jitter |
| Custo IA | limite configurável; zero quando IA desativada |

Limites de domínio e políticas são **compartilhados entre workers**, não apenas contadores locais; preservar logs de consumo real. Em presença de `429`, `Retry-After`, bloqueio ou erro de acesso, reduzir/coibir acesso — nunca tentar contornar.

## 6. Motor de evidências — impedir que três cópias virem três provas

Entidades mínimas: `Claim`, `Evidence`, `OriginCluster`, `DocumentVersion`, `Passage`, `Contradiction`, `Vendor`, `DeviceModel`, `FirmwareRelease`, `Command`, `ValidationRun` e `DiscoveryEdge`.

Cada **claim** deve conter a afirmação testável e seu escopo: fabricante, família, modelo, revisão, sistema operacional, versão/release, recurso, cenário, modo CLI e pré-requisitos. Uma evidência deve apontar **o documento concreto e versão** e **o trecho/página/offset** que sustenta ou refuta o claim, com hash e política de retenção.

Estados separados:
- **Status epistemológico:** sem evidência suficiente / sustentado por documento / corroborado por origens independentes / contestado.
- **Aplicabilidade:** modelo-versão confirmados / incompatível / indeterminados.
- **Validação operacional:** não testado / testado em laboratório autorizado / falhou no teste.

Um manual oficial sustenta a condição **documentada**, mas não comprova que a solução funciona no contexto operacional inteiro. Não produzir percentuais de certeza sem calibração. Um claim pode permanecer contestado mesmo com muitas citações.

**Exemplo obrigatório:** continuidade de sessões PPPoE não pode ser inferida de failover BGP, VRRP ou redundância RADIUS. São propriedades independentes que precisam de evidências distintas.

## 7. Busca híbrida de verdade, especialmente para comandos CLI

Camadas de busca:

- **Exata:** comparação literal/canônica de comandos, mensagens de erro, caminho CLI, OID, versão e blocos de configuração. Preservar pontuação, barras, aspas, espaçamento e quebras originais onde semanticamente importantes.
- **Lexical documental:** PostgreSQL `to_tsvector` e ranking, separado por idioma e tipo de documento. Não afirmar que a busca padrão fornece BM25; usa `ts_rank`/`ts_rank_cd`.
- **Aproximada:** `pg_trgm` em títulos, nomes, termos e atributos selecionados. Ele não preserva todos os símbolos como unidade de trigramas e pode ser ruim para padrões curtos.
- **Semântica:** pgvector **somente após** seleção de um embedding local/cloud, versionamento do modelo e benchmark de ganho de recall.
- **Code search especializado:** primeiro Git API/clone seletivo e busca literal; Zoekt apenas quando o corpus de código exigir.

Combinar candidatos por regras + fusão de rankings (p.ex., Reciprocal Rank Fusion) com pesos testados, não substituir fonte por similaridade. Após recuperação sempre conferir correspondência exata do trecho e suas restrições de firmware/modelo.

## 8. Segurança e compliance são restrições do sistema, não prompts

**Crawler/SSRF:** aceitar apenas HTTP(S) público e fontes explicitamente autorizadas. Validar URL e scheme; resolver IPv4/IPv6; rejeitar loopback, link-local, redes privadas, multicast, metadados cloud e destinos não permitidos; **conectar ao IP validado** para evitar DNS rebinding; repetir a validação para cada redirecionamento; e impor **bloqueio de egress em rede** fora do perímetro permitido. Sem cabeçalhos com tokens de administração.

**Robots e uso justo:** implementar RFC 9309, observar termos, quotas/APIs, códigos `429/403`, exclusão e autorizações. **Colly anuncia suporte a robots, mas a configuração deve explicitamente ativar o respeito a robots.txt** e ser coberta por teste; sua implementação possui opção/valor padrão que pode ignorá-lo. Não mascarar identidade para ultrapassar restrições.

**Conteúdo externo:** HTML, README, PDF, texto de fórum e saída de ferramentas são dados **não confiáveis**. Um modelo jamais recebe capacidade irrestrita de executar URLs arbitrárias, fazer chamadas administrativas, acessar segredos, instalar software ou executar comandos de roteadores.

**Execução de arquivos:** jamais executar scripts, firmware, binários ou comandos coletados. Parsers/OCR/browser em ambiente sem privilégio, limites de CPU/RAM/tempo, filesystem somente leitura e rede/volumes restritos. Antibombas ZIP/PDF, limites de quantidade/tamanho/recursão de anexos.

**Privacidade/licenças:** classificar fonte como `somente link/trecho`, `arquivo permitido` ou `retenção negada`. Censurar dados pessoais, senhas e tokens antes de indexar e antes de enviar a modelo externo; manter política de remoção e registro de fundamento.

**Aplicação:** RBAC, autenticação, cookies seguros, CSRF conforme autenticação, sanitização de HTML/Markdown e CSP contra XSS, rate limit, logs sem segredos, TLS quando acessível remotamente e política de retenção. Implementar autorização no código da ferramenta, não somente por prompt.

## 9. Dados, armazenamento e resiliência

- **PostgreSQL:** fonte de verdade de campanhas, usuários, tarefas, metadados, documentos, chunks, claims, evidências e arestas de descoberta.
- **Filesystem:** `objects/sha256/aa/bb/<hash>` para bytes imutáveis. Atualizar referência no PostgreSQL **após** escrita temporária + fsync + rename atômico; órfãos são coletados com janela de segurança.
- **Documento:** hash do conteúdo, URL, data, ETag/Last-Modified, licença, fabricante/modelo, parser e versão, páginas, blocos e histórico.
- **Índices:** materiais recuperáveis/reconstruíveis; versionar transformações e embeddings, nunca destruir o original permitido.
- **Backup:** restic para objetos/configuração, backups consistentes de PostgreSQL; restauração testada, cópia física diferente e política de retenção. Em escala, WAL/PITR.
- **Fila PostgreSQL:** `FOR UPDATE SKIP LOCKED` + lease, idempotência, tentativas limitadas, backoff, dead-letter, cancelamento e recuperação de jobs interrompidos.

MinIO **não** entra como dependência padrão: seu repositório público foi arquivado em abril/2026. Um adaptador `ObjectStore` permitirá filesystem agora e S3 compatível depois, avaliando alternativa mantida e licenças.

## 10. Inteligência e modelos sem lock-in

Orquestrador próprio com contratos:
- `ResearchPlanner` → subperguntas e objetivos de evidência;
- `DiscoveryProvider` → fontes candidatas e cobertura;
- `Fetcher`/`Extractor` → conteúdo bruto e estrutura;
- `EvidenceResolver` → claims, contradições e compatibilidade;
- `ReportWriter` → dossiê **vinculado a evidências existentes**.

Usar saída estruturada validada, versões de prompts/modelos, timeout e orçamento. Expor opções `Sem IA`, `Modelo local` e `API externa`. **IA local não faz parte do dimensionamento mínimo** e pode exigir RAM/GPU adicionais. Sem modelo, a busca exata e descoberta por links/menções estruturadas continuam funcionando; investigação semântica multietapas fica limitada.

Não obrigatório: GPT Researcher, Hyperresearch, RAGFlow ou Onyx. Avaliar cada um em um benchmark controlado, com `feature flags` e adaptadores; integrar apenas quando melhorar uma métrica sem duplicar repositórios de dados.

## 11. Funcionalidades de interface e experiência

Painel inicial com pesquisa única e modos simples. Para uma campanha, mostrar: objetivo, estados, fontes consultadas, gasto de recursos, novas URLs, grafo de ramificações, causas de prioridade, evidências, contradições, fila e botão de pausa/retomada. Configuração visual de limites, domínio, tema, agenda, modelo IA e política de retenção.

Busca deve aceitar texto exato com pontuação intacta e filtros de fabricante/modelo/release. Ao clicar na resposta: abrir **documento real, versão imutável, página/seção e trecho**. Dossiê exportável em Markdown (PDF posterior, se necessário), com lacunas e afirmações não comprovadas explicitadas.

## 12. Escalabilidade por evidência, não por ansiedade

**Não criar um serviço apenas para cumprir um desenho.** Indicadores para reconsiderar escolhas:

| Sintoma medido | Extensão a testar | Pré-condição |
|---|---|---|
| Pesquisa de código em repositórios selecionados acima do SLO | Zoekt | medir ganho vs. `ripgrep`/API Git |
| Pesquisa documental PostgreSQL lenta ou índice inviável no hardware | OpenSearch ou outro índice dedicado | provar carga, consulta e custo |
| Embeddings se tornam gargalo separados da busca textual | Reavaliar pgvector/índice ANN ou banco vetorial | benchmark de recall, latência, manutenção |
| Muitas tarefas, bloqueio de filas ou elevado write amplification | NATS JetStream/filas dedicadas | métricas de throughput + estratégia outbox |
| Arquivos precisam de acesso S3 multinó | SeaweedFS ou outro S3 mantido | teste de durabilidade/compatibilidade/backup |
| Crawling distribuído de grande porte passa a ser obrigatório | StormCrawler/Nutch | taxa de descoberta útil e custo comprovados |
| RAG documental próprio torna-se caro de manter | Piloto RAGFlow/Onyx | evitar segundo catálogo como fonte de verdade |

Limiares quantitativos devem ser estabelecidos **após medir em corpora representativos**; não prometer desempenho universal por tamanho de acervo.

## 13. Ordem de execução — nova proposta

### Fase 0 — Baseline e dados de referência
- Fazer dataset de 20 perguntas originais + documentos-ouro; fontes públicas autorizadas.
- Verificar versões, licenças e APIs de cada dependência.
- Montar teste de busca manual vs. busca federada vs. investigação autônoma.
**Aceite:** perguntas, referências e critérios revisados antes de codificar.

### Fase 1 — Vertical slice real com recursão mínima
- API/painel básico, PostgreSQL, descoberta Git/web, fila persistente, crawler seguro e processamento HTML/PDF simples.
- Salvar ao menos um fluxo completo **busca → descoberta de link/menção → extração → arquivo → evidência citável → nova pista**, com pausa/retomada.
- Busca exata por comandos, filtros e documento original.
**Aceite:** caminho ponta a ponta real e reproduzível, não mockado.

### Fase 2 — Evidência e investigação profunda
- Claims e clusters de independência, firmware, contradicts/supports e relatório com citações.
- IA conectável (local/cloud), pesquisa semântica opcional e políticas de saída verificáveis.
**Aceite:** problemas difíceis sem misturar firmwares e sem alegar validação inexistente.

### Fase 3 — Autonomia contínua e controles operacionais
- Agendas, revisitas por ETag/Last-Modified/hash, orçamento por fonte, inspeção do grafo, observabilidade, alertas e dedup durável.
**Aceite:** campanhas recuperam após interrupções e descobrem novas fontes sem intervenção constante.

### Fase 4 — Ampliação condicionada
- Zoekt, OpenSearch, workers distribuídos, processamento pesado, armazenamento S3, fontes especializadas e múltiplos usuários **somente se comprovadamente necessários**.
**Aceite:** benefício quantificado superior ao custo de manutenção.

## 14. Excluído do MVP deliberadamente

- Índice universal da internet e espelhamento integral de GitHub/GitLab.
- Tor e metadados BitTorrent (fontes opcionais futuras, com uso legítimo e aprovação).
- Execução automática de comandos CLI em equipamentos em produção.
- Clone cego de repositórios sem seleção e cota.
- Cluster Kubernetes, vários bancos vetoriais, múltiplos frameworks RAG no mesmo pipeline.
- SaaS multi-tenant completo e treinamento de LLM do zero.

## 15. Sucesso mensurável

O projeto será considerado bom quando puder recuperar uma informação difícil e **mostrar a rota auditável** que levou até ela; quando diferenciar firmware incompatível; quando não contar cópias como fontes independentes; quando puder interromper/reiniciar uma campanha sem perder trabalho; e quando não ultrapassar permissões, rede e recursos autorizados.

**Nenhuma função deste documento foi declarada como implementada.** O plano corrige o projeto no nível de arquitetura, requisitos e critérios verificáveis. O código precisa ser criado e testado.
