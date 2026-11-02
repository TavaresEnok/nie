# Pacote de correção de escopo — Network Intelligence Engine V2.1

Este pacote não substitui o ZIP original nem afirma implementação concluída. Ele faz a ponte entre a pesquisa histórica e a arquitetura leve V2.

- `ADENDO_MESTRE_NIE_V2_1.md`: nova arquitetura, seleção de crawlers, desenho de ramificações, interfaces, infraestrutura, critérios de aceite e fontes oficiais.
- `MATRIZ_59_FONTES_E_PROJETOS_V2_1.csv`: inventário detalhado, em ponto-e-vírgula, pronto para importar em planilha.
- `CATALOGO_COMPLETO_59_REGISTROS.md`: visão Markdown pesquisável da classificação.

**Principal correção:** manter Go no control plane e aproveitar o rastreamento maduro do Scrapy, do Crawl4AI e futuramente do StormCrawler em vez de reimplementar tudo com Colly.
