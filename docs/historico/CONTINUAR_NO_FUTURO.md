# PROMPT PARA RETOMAR O PROJETO EM OUTRA CONVERSA

> **Documento histórico (30/10/2026).** Escrito antes de existir código. Hoje o
> projeto já tem implementação; para retomar, comece por [`../README.md`](../README.md),
> [`../DECISOES.md`](../DECISOES.md) e o `README.md` da raiz. `ARQUIVO_MESTRE.md`
> corresponde a [`ARQUIVO_MESTRE_V1.md`](ARQUIVO_MESTRE_V1.md).

Quero continuar a especificação e implementação do **Network Intelligence Engine**, também referido como **Git Eye + Web Eye + Autonomous Discovery Engine**.

Leia o documento `ARQUIVO_MESTRE.md` integralmente e, se necessário, os quatro arquivos em `originais/`. É um arquivo de planejamento e pesquisa: **não confunda recursos propostos com funcionalidades já implementadas**.

## Objetivo

Desenvolver uma plataforma auto-hospedável de pesquisa técnica profunda para redes/ISPs que descubra autonomamente fontes por links, sitemaps, menções, citações, fóruns, Git, PDFs e arquivos históricos; pesquise conteúdo público permitido; mantenha acervo local; classifique equipamento, modelo, release/firmware e comandos CLI; confronte evidências independentes e gere dossiês rastreáveis.

## Preferências e decisões registradas

- Não desenvolver buscadores do zero; avaliar uso de Zoekt, SearXNG, PostgreSQL, OpenSearch, Scrapy/Crawl4AI, Tika/Docling e armazenamento de objetos.
- GPT Researcher/Hyperresearch/RAGFlow/Onyx e Nutch/StormCrawler são candidatos a avaliar, **não dependências obrigatórias**.
- Forte ênfase em **expansão recursiva autônoma, guiada por relevância e limites**, sem exigir listas manuais completas de sites.
- Priorizar Huawei NE8000/VRP, MikroTik RouterOS, OLTs, Datacom DmOS, Cisco, ZTE, VSOL, FiberHome, SonicWall, Hillstone, BGP, MPLS, PBR, PPPoE, BNG/BRAS e RADIUS.
- Três resultados iguais **não bastam** se as fontes copiarem umas às outras; exigir proveniência e compatibilidade por firmware.
- Preservar as fontes originais quando autorizado; registrar links, hashes, páginas, versões, datas e incertezas.
- Sem burlar acesso, sem coletar dados privados, sem executar scripts/CLI encontrados.
- Não prometer “primeiro do mundo”; Diffbot, GPT Researcher, Hyperresearch, YaCy, StormCrawler, Mwmbl e Onyx são precedentes.
- Começar por um **MVP verificável** e benchmarking com ~20 perguntas técnicas, antes de escalar para TB.

## Sua primeira tarefa

Revisar a especificação do MVP e apresentar as **decisões mínimas de arquitetura**, os módulos, interfaces, esquema de dados, plano de implementação incremental e critérios de teste, distinguindo claramente fatos verificados, propostas e hipóteses.
