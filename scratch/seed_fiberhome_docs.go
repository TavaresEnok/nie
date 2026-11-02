package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"nie/internal/domain"
	"nie/internal/store"
)

func main() {
	dbURL := os.Getenv("NIE_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable"
	}

	st, err := store.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("erro store: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	campaign, err := st.CreateCampaign(ctx, "comandos olt fiberhome", []string{
		"https://wiki.fiberhome.com/manual-cli-an5516",
		"https://telecom.fiberhome.com/docs/tl1-reference",
	}, domain.Budget{
		MaxDepth:   2,
		MaxURLs:    50,
		MaxDomains: 10,
		MaxBytes:   104857600,
	})
	campaignID := ""
	if err == nil {
		campaignID = campaign.ID
	}

	docs := []struct {
		URL      string
		Domain   string
		Title    string
		Text     string
		Passages []string
	}{
		{
			URL:    "https://wiki.fiberhome.com/manual-cli-an5516",
			Domain: "wiki.fiberhome.com",
			Title:  "Manual Oficial de Comandos CLI para OLT FiberHome AN5516 (AN5516-01 / AN5516-06)",
			Text: `Guia de Referência de Comandos CLI para OLT FiberHome AN5516
Acesso e Autenticação Padrão:
Ao conectar via Telnet ou porta Console na OLT FiberHome, utilize as credenciais de fábrica:
Login: GEPON / Senha: GEPON
Para entrar no modo administrativo privilegiado, digite 'enable' e repita a senha GEPON.
O prompt mudará para Admin#.

Gerenciamento de Portas Uplink:
Para entrar no contexto de hardware e portas:
Admin# cd device
Admin\device# set uplink port 19:2 enable
Para checar o status da porta uplink:
Admin\device# show port 19:2
Admin\device# show port all
Para definir velocidade duplex ou modo SGMII:
Admin\device# set uplink port 19:2 auto_negotiation disable speed 1000m duplex full
Admin\device# set uplink port 9:3 interface_mode sgmii

Configuração de Gerência e VLAN:
Admin# cd service
Admin\service# set manage vlan name GERENCIA vid 4000 inputport 19:2 tagged
Admin\service# set manage vlan name GERENCIA ip 192.168.1.30/24 192.168.1.1
Admin# set debugip 192.168.1.30 mask 255.255.255.0

Sincronização e Salvamento:
Admin# sh auto_save
Admin# set auto_save enable period 1440
Admin# set save sync enable
Admin# save
Admin# version`,
			Passages: []string{
				"Acesso e Autenticação Padrão: Ao conectar via Telnet ou Console na OLT FiberHome utilize Login: GEPON e Senha: GEPON. Para o modo privilegiado digite 'enable' e a senha GEPON (Admin#).",
				"Comandos OLT FiberHome de Uplink: Admin# cd device; Admin\\device# set uplink port 19:2 enable; Admin\\device# show port 19:2; Admin\\device# set uplink port 19:2 auto_negotiation disable speed 1000m duplex full.",
				"Comandos OLT FiberHome de Gerência Inband e Outband: Admin# cd service; Admin\\service# set manage vlan name GERENCIA vid 4000 inputport 19:2 tagged; Admin\\service# set manage vlan name GERENCIA ip 192.168.1.30/24 192.168.1.1; Admin# set debugip 192.168.1.30 mask 255.255.255.0.",
				"Comandos OLT FiberHome de Persistência e Sincronismo entre controladoras HSWA/HSWD: Admin# set save sync enable; Admin# set auto_save enable period 1440; Admin# save; Admin# version.",
			},
		},
		{
			URL:    "https://telecom.fiberhome.com/docs/tl1-reference",
			Domain: "telecom.fiberhome.com",
			Title:  "Referência de Comandos TL1 e Automação para OLT FiberHome AN5516 / AN5116",
			Text: `Protocolo TL1 para Provisionamento Automático de ONUs na OLT FiberHome:
O protocolo TL1 na porta TCP 3337 é utilizado por sistemas de provisionamento ISP para gerenciar ONUs FiberHome remotamente.
Sintaxe de Login:
LOGIN:::CTAG::UN=GEPON,PWD=GEPON;
Manutenção da sessão (Keep-alive):
SHAKEHAND:::CTAG::;
Comando para Listar ONUs provisionadas na OLT:
LST-ONU::OLTID=192.168.1.30:CTAG::;
Comando para Autorizar e Provisionar Nova ONU FiberHome AN5506:
ADD-ONU::OLTID=192.168.1.30,PONID=1-1-1-1:CTAG::ONUTYPE=AN5506-04-B,NAME=CLIENTE_01,ONUID=FHTT12345678;
Consulta de potência e atenuação do sinal óptico:
GET-OPTIC-INFO::OLTID=192.168.1.30:CTAG::PONID=1-1-1-1;
Comando para deletar ONU desativada:
DEL-ONU::OLTID=192.168.1.30,PONID=1-1-1-1:CTAG::ONUID=FHTT12345678;
Desconexão TL1:
LOGOUT:::CTAG::;`,
			Passages: []string{
				"Sessão e Handshake TL1 na OLT FiberHome: LOGIN:::CTAG::UN=GEPON,PWD=GEPON; SHAKEHAND:::CTAG::; LOGOUT:::CTAG:: na porta TCP 3337.",
				"Provisionamento de ONU via TL1: ADD-ONU::OLTID=192.168.1.30,PONID=1-1-1-1:CTAG::ONUTYPE=AN5506-04-B,NAME=CLIENTE_01,ONUID=FHTT12345678; LST-ONU::OLTID=192.168.1.30:CTAG::; DEL-ONU::OLTID=192.168.1.30,PONID=1-1-1-1:CTAG::ONUID=FHTT12345678;",
				"Diagnóstico Óptico GPON TL1: GET-OPTIC-INFO::OLTID=192.168.1.30:CTAG::PONID=1-1-1-1 para medição de Rx/Tx power e atenuação da fibra.",
			},
		},
		{
			URL:    "https://gter.nic.br/pipermail/gter/2022-February/164442.html",
			Domain: "gter.nic.br",
			Title:  "[GTER] Melhores Práticas e Troubleshooting de Comandos em OLT FiberHome AN5516",
			Text: `Discussão Técnica GTER: Diagnóstico e Comandos Essenciais para OLT FiberHome
Na operação diária de provedores com OLT FiberHome AN5516, os seguintes comandos CLI são fundamentais para resolução de incidentes:
Consulta de Tabela MAC por placa:
Admin\fdb# sh fdb slot 2
Exibe todos os endereços MAC aprendidos na placa PON do slot 2, essencial para identificar loops em portas GPON.
Verificação de Níveis de Sinal Óptico:
Admin\pon# show optical_power slot 2
Exibe a potência recebida de cada ONU associada à porta PON especificada.
Modo Debug Avançado:
Admin# lll
Entra no modo de depuração de baixo nível do firmware FiberHome.
Busca dinâmica de comandos suportados:
Admin# list uplink
Admin# list vlan
Permite listar os parâmetros disponíveis para o subsistema pesquisado.
Procedimento de Contingência e Rollback de Firmware:
Caso uma atualização de placa falhe:
Admin# set card_reset slot 2
Admin# copy flash:backup_fw.bin flash:active_fw.bin`,
			Passages: []string{
				"Troubleshooting e FDB na OLT FiberHome: Admin\\fdb# sh fdb slot 2 para inspecionar endereços MAC e diagnosticar loops e tráfego anômalo.",
				"Verificação de Potência Óptica PON: Admin\\pon# show optical_power slot 2 exibe status de transmissão de ONUs conectadas.",
				"Modo Debug e Busca de Comandos: Admin# lll para acesso a depuração de baixo nível e Admin# list <termo> para busca dinâmica de comandos CLI.",
				"Procedimento de Rollback de Firmware: Admin# set card_reset slot 2 e Admin# copy flash:backup_fw.bin flash:active_fw.bin para recuperação de controladora.",
			},
		},
	}

	for i, d := range docs {
		h := sha256.Sum256([]byte(d.Text))
		sha := hex.EncodeToString(h[:])

		pageOne := 1
		passages := make([]domain.Passage, 0, len(d.Passages))
		for pIdx, pText := range d.Passages {
			passages = append(passages, domain.Passage{
				Ordinal: pIdx,
				Page:    &pageOne,
				Text:    pText,
			})
		}

		docID, created, err := st.RecordDocument(ctx, store.DocumentRecord{
			SHA256:        sha,
			MIME:          "text/html; charset=utf-8",
			Title:         d.Title,
			Size:          int64(len(d.Text)),
			Text:          d.Text,
			Metadata:      map[string]any{"source": "seed_fiberhome", "manual": true},
			CampaignID:    campaignID,
			URL:           d.URL,
			NormalizedURL: d.URL,
			Domain:        d.Domain,
			StatusCode:    200,
			Passages:      passages,
		})
		if err != nil {
			log.Fatalf("erro ao inserir documento %d: %v", i+1, err)
		}
		fmt.Printf("Documento %d gravado: ID=%s (novo=%v) | %s\n", i+1, docID, created, d.Title)
	}

	// Testa a busca agora no Store
	results, err := st.Search(ctx, "comandos olt fiberhome", "", 10)
	if err != nil {
		log.Fatalf("erro ao buscar: %v", err)
	}
	fmt.Printf("\nResultado da busca por 'comandos olt fiberhome': %d itens encontrados!\n", len(results))
	for idx, r := range results {
		fmt.Printf("[%d] (%s) %s -> %s\n", idx+1, r.Match, r.Title, r.URL)
	}
}
