# Inventário do backup — 59 registros preservados

Este inventário classifica o catálogo original sem confundir software instalado com fonte de pesquisa. Os estados são **recomendações**, não resultados de testes nem licenças verificadas para todos os itens.

| Nome | Papel | Decisão V2.1 |
|---|---|---|
| Zoekt | Motor de índice Git | Integrar em Git Eye de forma seletiva |
| OpenGrok | Motor Git alternativo | Piloto opcional |
| Sourcegraph | Plataforma comercial/serviço | Referência externa |
| Sourcebot | Ferramenta de busca Git | Não usar como núcleo aberto |
| Livegrep | Motor Git alternativo | Não instalar inicialmente |
| Hound | Motor Git alternativo | Não instalar inicialmente |
| Software Heritage | Acervo público externo | Integrar via adaptador |
| Software Heritage Archive | Interface do acervo | Integrar via adaptador |
| ecosyste.ms | Fonte de metadados | Integrar via adaptador |
| grep.app | Busca de código pública | Avaliar API/limites |
| GitHub Code Search | Busca pública | Integrar como fonte |
| GitHub REST Search | API pública | Integrar como fonte |
| GitLab Search | API pública | Integrar como fonte |
| GH Archive | Dataset de eventos | Opcional |
| World of Code | Corpus de pesquisa | Estudar como fonte |
| SearXNG | Metabusca | Integrar no MVP |
| Mwmbl | Índice web independente | Integrar como fonte |
| Mwmbl Search | Interface de busca | Unificar com Mwmbl |
| Common Crawl | Corpus web histórico | Integrar sob demanda |
| Wayback Machine | Arquivo histórico | Integrar via adaptador |
| Scrapy | Crawler recursivo | Promover a motor principal de crawling |
| Crawl4AI | Crawler inteligente/extrator | Integrar sob demanda |
| Apache Nutch | Crawler massivo | Escala futura |
| Apache StormCrawler | Crawler contínuo distribuído | Preservar como eixo de expansão |
| Heritrix | Crawler de arquivamento | Opcional especializado |
| YaCy | Buscador com crawler próprio | Experimento opcional |
| Crawlee | Crawler JS/TS | Alternativa a Scrapy/Crawl4AI |
| Ahmia crawler | Crawler onion | Fase opcional restrita |
| bitmagnet | Índice de metadados DHT | Fase opcional restrita |
| Apache Tika | Parser universal | Integrar sob demanda |
| Docling | Parser de documentos | Integrar sob demanda |
| ArchiveBox | Arquivamento | Integrar seletivamente |
| OpenSearch | Índice documental | Escala conforme benchmark |
| Apache Solr | Índice alternativo | Não instalar inicialmente |
| PostgreSQL | Banco transacional | Núcleo obrigatório |
| MinIO | Storage de objetos | Não escolher repositório arquivado para novo núcleo |
| Qdrant | Motor vetorial | Opcional sob benchmark |
| Paperless-ngx | Gestor documental | Não instalar inicialmente |
| RAGFlow | RAG/plataforma | Piloto competitivo, não obrigatório |
| Onyx | Pesquisa/RAG corporativo | Alternativa de RAGFlow |
| Open WebUI | Interface para LLM | Opcional |
| AnythingLLM | Interface de conhecimento | Opcional alternativo |
| GPT Researcher | Agente de deep research | Piloto para motor de investigação |
| Hyperresearch | Research harness e biblioteca | Piloto de evidência/qualidade |
| LangChain Open Deep Research | Projeto histórico | Somente referência de arquitetura |
| Diffbot | Plataforma comercial | Precedente, não núcleo open source |
| Juniper Marvis | Produto comercial | Concorrência/benchmark |
| Cisco AI | Produto comercial | Concorrência/benchmark |
| Cisco Community | Fonte de conteúdo | Conector de fonte permitido |
| MikroTik Forum | Fonte de conteúdo | Conector de fonte permitido |
| Viva o Linux | Fonte de conteúdo | Conector de fonte permitido |
| Stack Overflow | Fonte de conteúdo | API/extração permitida |
| Network Engineering Stack Exchange | Fonte de conteúdo | API/extração permitida |
| Server Fault | Fonte de conteúdo | API/extração permitida |
| RFC Editor | Fonte documental | Conector prioritário |
| RIPE NCC | Fonte documental | Conector prioritário |
| NANOG | Fonte documental | Conector prioritário |
| FRRouting | Projeto técnico/fonte | Conector prioritário |
| Autonomous Research Agent (experimental) | Precedente experimental | Somente avaliar após identificar URL exata |

## Fonte

Registros derivados do `FONTES_E_PROJETOS.csv` do ZIP original (30/10/2026), com classificação proposta em 02/11/2026.
