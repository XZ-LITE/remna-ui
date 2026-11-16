package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"remna-ui/internal/dns"
	"remna-ui/internal/types"
)

func hasComponent(components []string, target string) bool {
	for _, c := range components {
		if c == target {
			return true
		}
	}
	return false
}

func WriteOutputFiles(plan types.DeploymentPlan, targetDir string) types.GenerateResult {
	if targetDir == "" {
		targetDir = "./out"
	}

	cleanDir, err := filepath.Abs(targetDir)
	if err != nil {
		cleanDir = targetDir
	}

	if err := os.MkdirAll(cleanDir, 0755); err != nil {
		return types.GenerateResult{
			Success: false,
			Error:   fmt.Sprintf("Не удалось создать папку: %v", err),
		}
	}

	var createdFiles []string

	writeFile := func(subPath, content string) error {
		fullPath := filepath.Join(cleanDir, subPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
		createdFiles = append(createdFiles, subPath)
		return nil
	}

	// 1. Manifest
	manifestData := map[string]interface{}{
		"components":          plan.Components,
		"domain":              dns.NormalizeDomain(plan.Domain),
		"panel_domain":        dns.NormalizeDomain(plan.PanelDomain),
		"subscription_domain": dns.NormalizeDomain(plan.SubscriptionDomain),
		"proxy":               plan.Proxy,
		"subscription_mode":   plan.SubscriptionMode,
		"node_port":           plan.NodePort,
	}
	manifestJson, _ := json.MarshalIndent(manifestData, "", "  ")
	_ = writeFile("manifest.json", string(manifestJson)+"\n")

	// 2. README & SETUP_GUIDE
	readmeContent := fmt.Sprintf(`# Развёртывание Remnawave

Конфигурация сгенерирована через Remnawave Manager.
- Домен панели: %s
- Прокси: %s
- Компоненты: %s
`, plan.PanelDomain, plan.Proxy, stringsJoin(plan.Components, ", "))
	_ = writeFile("README.md", readmeContent)

	guideContent := `# Руководство по развёртыванию

1. Скопируйте эту папку на ваш Linux VPS сервер (например, в /opt/remnawave).
2. Настройте DNS A-записи вашего домена на IP сервера.
3. Запустите скрипт автоматической установки:
   chmod +x deploy.sh && ./deploy.sh
4. Панель будет доступна по адресу https://%s
`
	_ = writeFile("SETUP_GUIDE.md", fmt.Sprintf(guideContent, dns.NormalizeDomain(plan.PanelDomain)))

	// 3. Panel
	hasPanel := hasComponent(plan.Components, "panel")
	hasNode := hasComponent(plan.Components, "node")
	hasProxy := hasComponent(plan.Components, "proxy")
	hasSub := hasComponent(plan.Components, "subscription")

	if hasPanel {
		var envContent string
		var composeContent string
		if plan.PerformanceOptimization {
			envContent = RenderPanelEnv(plan.PanelDomain, plan.SubscriptionDomain, hasSub)
			composeContent = RenderPanelCompose()
		} else {
			envContent = RenderPanelEnvStandard(plan.PanelDomain, plan.SubscriptionDomain, hasSub)
			composeContent = RenderPanelComposeStandard()
		}
		_ = writeFile(filepath.Join("panel", ".env"), envContent)
		_ = writeFile(filepath.Join("panel", "docker-compose.yml"), composeContent)
	}

	// 4. Node
	if hasNode {
		nodePort := plan.NodePort
		if nodePort == 0 {
			nodePort = 2222
		}
		_ = writeFile(filepath.Join("node", "docker-compose.yml"), RenderNodeCompose(nodePort, plan.NodeSecret))
		_ = writeFile(filepath.Join("node", "xray-config.sample.json"), RenderXraySample())

		nodeGuide := fmt.Sprintf(`# Подключение ноды к панели

1. Откройте веб-панель Remnawave: Nodes -> Management -> Создать ноду.
2. Укажите Node Port %d.
3. Скопируйте выданный панели Compose и вставьте его SECRET_KEY в node/docker-compose.yml.
4. Выполните: cd node && docker compose up -d
`, nodePort)
		_ = writeFile(filepath.Join("node", "CONNECT_NODE.md"), nodeGuide)
	}

	// 5. Proxy
	if hasProxy {
		panelDomain := plan.PanelDomain
		if panelDomain == "" {
			panelDomain = plan.Domain
		}
		bundledSub := hasSub && plan.SubscriptionMode == "bundled"

		if plan.Proxy == "nginx" {
			conf := RenderNginxConf(panelDomain, plan.SubscriptionDomain, bundledSub, plan.SecretGate, plan.UseCamouflage)
			_ = writeFile(filepath.Join("nginx", "nginx.conf"), conf)
			_ = writeFile(filepath.Join("nginx", "docker-compose.yml"), RenderNginxCompose())
			if plan.UseCamouflage {
				_ = writeFile(filepath.Join("nginx", "site", "index.html"), RenderCamouflageSite())
			}
		} else {
			caddyfile := RenderCaddyfile(panelDomain, plan.SubscriptionDomain, bundledSub, plan.SecretGate, plan.UseCamouflage)
			_ = writeFile(filepath.Join("caddy", "Caddyfile"), caddyfile)
			_ = writeFile(filepath.Join("caddy", "docker-compose.yml"), RenderCaddyCompose())
			if plan.UseCamouflage {
				_ = writeFile(filepath.Join("caddy", "site", "index.html"), RenderCamouflageSite())
			}
		}
	}

	// 6. Subscription
	if hasSub {
		_ = writeFile(filepath.Join("subscription", ".env"), RenderSubscriptionEnv(plan.PanelDomain, "", plan.SubscriptionMode))
		_ = writeFile(filepath.Join("subscription", "docker-compose.yml"), RenderSubscriptionCompose(plan.SubscriptionMode))
	}

	// 7. Scripts
	deployShPath := filepath.Join(cleanDir, "deploy.sh")
	_ = os.WriteFile(deployShPath, []byte(RenderDeploySh()), 0755)
	createdFiles = append(createdFiles, "deploy.sh")

	_ = writeFile("deploy.ps1", RenderDeployPs1())

	return types.GenerateResult{
		Success: true,
		Files:   createdFiles,
		OutDir:  cleanDir,
	}
}

func stringsJoin(items []string, sep string) string {
	res := ""
	for i, item := range items {
		if i > 0 {
			res += sep
		}
		res += item
	}
	return res
}
