package classifier

import (
	"testing"
)

func TestClassifyHuawei(t *testing.T) {
	text := `
		Huawei NetEngine 8000 configuration for BGP
		system-view
		bgp 65000
		peer 10.0.0.1 as-number 65001
		display bgp peer
		commit
	`
	res := Classify(text, "support.huawei.com")
	if !res.IsOfficialDoc {
		t.Errorf("deveria ser considerado doc oficial")
	}
	if len(res.Vendors) == 0 || res.Vendors[0].Slug != "huawei" {
		t.Fatalf("falha ao identificar Huawei")
	}
	if res.Vendors[0].OS != "VRP v8" {
		t.Errorf("esperava VRP v8, obteve %s", res.Vendors[0].OS)
	}
	if len(res.CLICommands) == 0 {
		t.Errorf("esperava comandos CLI extraídos")
	}
}

func TestClassifyMikroTik(t *testing.T) {
	text := `
		MikroTik RouterOS v7 BGP configuration:
		/routing bgp connection add name=uplink remote.address=192.168.1.1 as=65000
		/routing filter rule add chain=bgp-in rule="if dst in 10.0.0.0/8 then accept"
	`
	res := Classify(text, "forum.mikrotik.com")
	if len(res.Vendors) == 0 || res.Vendors[0].Slug != "mikrotik" {
		t.Fatalf("falha ao identificar MikroTik")
	}
	if res.Vendors[0].OS != "RouterOS v7" {
		t.Errorf("esperava RouterOS v7, obteve %s", res.Vendors[0].OS)
	}
	if len(res.CLICommands) < 2 {
		t.Errorf("esperava comandos MikroTik extraídos, obteve %d", len(res.CLICommands))
	}
}

func TestClassifyCisco(t *testing.T) {
	text := `
		Cisco ASR9000 IOS-XR l2vpn setup
		router bgp 65000
		show bgp summary
		crypto ikev2 profile IPSEC-PROF
	`
	res := Classify(text, "cisco.com")
	if len(res.Vendors) == 0 || res.Vendors[0].Slug != "cisco" {
		t.Fatalf("falha ao identificar Cisco")
	}
	if res.Vendors[0].OS != "IOS-XR" {
		t.Errorf("esperava IOS-XR, obteve %s", res.Vendors[0].OS)
	}
}

func TestOfficialDomainSpoofing(t *testing.T) {
	adversarialNegative := []string{
		"huawei.com.attacker.com",
		"http://huawei.com.exemplo-atacante.com/malicious",
		"evil-cisco.com",
		"notcisco.com",
		"cisco.com.attacker.net",
		"zte.com.cn.fake.org",
		"user:pass@attacker.com#huawei.com",
		"http://attacker.com/?redirect=huawei.com",
		"fiberhome.com.br", // se oficial for fiberhome.com
		"fakefiberhome.com",
		"mikrotik.com.br.evil.com",
	}

	for _, domain := range adversarialNegative {
		if IsOfficialDomain(domain) {
			t.Errorf("Vulnerabilidade P0-02 detectada! Domínio adversário foi classificado como oficial: %s", domain)
		}
	}

	legitimatePositive := []string{
		"huawei.com",
		"support.huawei.com",
		"https://support.huawei.com/enterprise/en/doc/EDOC11000",
		"cisco.com",
		"support.cisco.com",
		"http://help.mikrotik.com/docs/display/ROS/BGP",
		"mikrotik.com",
		"forum.mikrotik.com",
		"datacom.com.br",
		"support.datacom.com.br",
		"fiberhome.com",
		"cisco.com:8443",
	}

	for _, domain := range legitimatePositive {
		if !IsOfficialDomain(domain) {
			t.Errorf("Domínio legítimo não foi reconhecido como oficial: %s", domain)
		}
	}
}

