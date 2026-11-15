package remnawave

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"remna-ui/internal/types"
)

func cleanURL(rawURL string) string {
	u := strings.TrimSpace(rawURL)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	parsed, err := url.Parse(u)
	if err == nil && parsed.Host != "" {
		return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	}
	return strings.TrimRight(u, "/")
}

func getHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 25 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

// TestPanelAPI verifies token and connectivity to the Remnawave Panel
func TestPanelAPI(panelURL, apiToken string) types.PanelInfoResult {
	targetURL := cleanURL(panelURL)
	cleanToken := strings.TrimSpace(apiToken)

	if targetURL == "" || cleanToken == "" {
		return types.PanelInfoResult{
			Success: false,
			Error:   "Укажите URL панели и API токен",
		}
	}

	req, err := http.NewRequest("GET", targetURL+"/api/nodes", nil)
	if err != nil {
		return types.PanelInfoResult{Success: false, Error: fmt.Sprintf("Ошибка запроса: %v", err)}
	}

	req.Header.Set("Authorization", "Bearer "+cleanToken)
	req.Header.Set("Accept", "application/json")

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "Client.Timeout") {
			return types.PanelInfoResult{
				Success: false,
				Error:   "Время ожидания ответа от панели истекло (25 сек). Сервер перегружен или недоступен.",
			}
		}
		return types.PanelInfoResult{Success: false, Error: fmt.Sprintf("Не удалось подключиться к %s: %v", targetURL, err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return types.PanelInfoResult{
			Success: false,
			Error:   "Неверный токен API (401 Unauthorized). Проверьте токен в Settings -> API Tokens",
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return types.PanelInfoResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка API (%d): %s", resp.StatusCode, string(body)),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.PanelInfoResult{Success: false, Error: "Не удалось прочитать ответ панели"}
	}

	// Count nodes in response
	nodesCount := 0
	var rawList []any
	if err := json.Unmarshal(body, &rawList); err == nil {
		nodesCount = len(rawList)
	} else {
		var wrapper map[string]any
		if err := json.Unmarshal(body, &wrapper); err == nil {
			if respList, ok := wrapper["response"].([]any); ok {
				nodesCount = len(respList)
			}
		}
	}

	return types.PanelInfoResult{
		Success:    true,
		Version:    "Remnawave v2/v3",
		NodesCount: nodesCount,
	}
}

func convertToMapList(list []any) []map[string]any {
	res := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			res = append(res, m)
		}
	}
	return res
}

func parseProfilesFromBytes(body []byte) []map[string]any {
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}

	var results []map[string]any

	// Helper to check if a map looks like a profile
	isProfileMap := func(m map[string]any) bool {
		if u, ok := m["uuid"].(string); ok && u != "" {
			return true
		}
		if id, ok := m["id"].(string); ok && id != "" {
			return true
		}
		return false
	}

	// Recursive search for profile list or profile objects
	var findProfiles func(v any)
	findProfiles = func(v any) {
		if len(results) > 0 {
			return
		}
		switch val := v.(type) {
		case []any:
			var candidateList []map[string]any
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					if isProfileMap(m) || m["name"] != nil {
						candidateList = append(candidateList, m)
					}
				}
			}
			if len(candidateList) > 0 {
				results = candidateList
				return
			}
			for _, item := range val {
				findProfiles(item)
			}
		case map[string]any:
			// Priority keys: "configProfiles", "profiles", "response", "items", "data"
			for _, key := range []string{"configProfiles", "profiles", "response", "items", "data"} {
				if child, ok := val[key]; ok {
					if list, ok := child.([]any); ok {
						var candidateList []map[string]any
						for _, item := range list {
							if m, ok := item.(map[string]any); ok {
								candidateList = append(candidateList, m)
							}
						}
						if len(candidateList) > 0 {
							results = candidateList
							return
						}
					}
				}
			}
			// If map itself has uuid/id
			if isProfileMap(val) && (val["name"] != nil || val["config"] != nil || val["inbounds"] != nil) {
				results = []map[string]any{val}
				return
			}
			// Search children
			for _, child := range val {
				findProfiles(child)
				if len(results) > 0 {
					return
				}
			}
		}
	}

	findProfiles(raw)
	return results
}

func extractProfileUUIDAndInbounds(targetURL, token string, client *http.Client, p map[string]any) (string, []string) {
	uuid := ""
	if u, ok := p["uuid"].(string); ok && u != "" {
		uuid = u
	} else if id, ok := p["id"].(string); ok && id != "" {
		uuid = id
	}

	inbounds := make([]string, 0)
	var rawInbounds []any
	if ibs, ok := p["inbounds"].([]any); ok {
		rawInbounds = ibs
	} else if cfg, ok := p["config"].(map[string]any); ok {
		if ibs, ok := cfg["inbounds"].([]any); ok {
			rawInbounds = ibs
		}
	}

	for _, item := range rawInbounds {
		if m, ok := item.(map[string]any); ok {
			if u, ok := m["uuid"].(string); ok && u != "" {
				inbounds = append(inbounds, u)
			} else if id, ok := m["id"].(string); ok && id != "" {
				inbounds = append(inbounds, id)
			}
		}
	}

	// If inbounds list is empty and we have a valid profile UUID, query GET /api/config-profiles/{uuid}/inbounds
	if len(inbounds) == 0 && uuid != "" {
		reqInbounds, err := http.NewRequest("GET", targetURL+fmt.Sprintf("/api/config-profiles/%s/inbounds", uuid), nil)
		if err == nil {
			reqInbounds.Header.Set("Authorization", "Bearer "+token)
			reqInbounds.Header.Set("Accept", "application/json")
			respInbounds, errDo := client.Do(reqInbounds)
			if errDo == nil && respInbounds.StatusCode >= 200 && respInbounds.StatusCode < 300 {
				defer respInbounds.Body.Close()
				ibBody, _ := io.ReadAll(respInbounds.Body)
				ibList := parseProfilesFromBytes(ibBody)
				for _, ibItem := range ibList {
					if u, ok := ibItem["uuid"].(string); ok && u != "" {
						inbounds = append(inbounds, u)
					} else if id, ok := ibItem["id"].(string); ok && id != "" {
						inbounds = append(inbounds, id)
					}
				}
			}
		}
	}

	return uuid, inbounds
}

func fetchActiveConfigProfile(targetURL, token, decoyDomain string, client *http.Client) (string, []string, error) {
	req, err := http.NewRequest("GET", targetURL+"/api/config-profiles", nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	list := parseProfilesFromBytes(body)
	if len(list) == 0 {
		return "", nil, fmt.Errorf("no config profiles found in panel")
	}

	cleanDecoy := strings.TrimSpace(decoyDomain)
	var candidateUUID string
	var candidateInbounds []string

	if cleanDecoy != "" {
		// User requested SelfSteal Decoy: look specifically for a profile configured for SelfSteal
		cleanDecoyLower := strings.ToLower(cleanDecoy)
		for _, p := range list {
			name, _ := p["name"].(string)
			desc, _ := p["description"].(string)
			combined := strings.ToLower(name + " " + desc)
			if strings.Contains(combined, "selfsteal") || strings.Contains(combined, cleanDecoyLower) {
				u, ibs := extractProfileUUIDAndInbounds(targetURL, token, client, p)
				if u != "" && len(ibs) > 0 {
					candidateUUID = u
					candidateInbounds = ibs
					break
				}
			}
		}
		// If no dedicated SelfSteal profile exists yet, trigger auto-creation
		if candidateUUID == "" {
			return "", nil, fmt.Errorf("no selfsteal profile found in panel")
		}
	} else {
		// User requested Classic Reality: prioritize Standard VLESS Reality or any Reality profile
		for _, p := range list {
			name, _ := p["name"].(string)
			desc, _ := p["description"].(string)
			nameLower := strings.ToLower(name + " " + desc)
			if strings.Contains(nameLower, "reality") || strings.Contains(nameLower, "vless") {
				u, ibs := extractProfileUUIDAndInbounds(targetURL, token, client, p)
				if u != "" && len(ibs) > 0 {
					candidateUUID = u
					candidateInbounds = ibs
					break
				}
			}
		}
	}

	// 2. Fallback: inspect any profile with valid inbounds
	if candidateUUID == "" {
		for _, p := range list {
			u, ibs := extractProfileUUIDAndInbounds(targetURL, token, client, p)
			if u != "" && len(ibs) > 0 {
				candidateUUID = u
				candidateInbounds = ibs
				break
			}
		}
	}

	// 3. Fallback: list[0]
	if candidateUUID == "" {
		candidateUUID, candidateInbounds = extractProfileUUIDAndInbounds(targetURL, token, client, list[0])
	}

	if candidateUUID == "" {
		return "", nil, fmt.Errorf("profile found but has no uuid")
	}

	return candidateUUID, candidateInbounds, nil
}

func generateX25519Key() string {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "kK34TqU826q_sJ5jL0V9qW1eR4tY7uI0oP2aSdFgHjK"
	}
	return base64.RawURLEncoding.EncodeToString(priv.Bytes())
}

func createDefaultConfigProfile(targetURL, token string, decoyDomain, portMode string, torrentBlocker, blockSpamPorts bool, client *http.Client) (string, []string, error) {
	privateKey := generateX25519Key()

	dest := "www.microsoft.com:443"
	serverNames := []string{"www.microsoft.com"}
	profileName := "Standard VLESS Reality"
	inboundTag := "vless-reality-in"
	xver := 0

	cleanDecoy := strings.TrimSpace(decoyDomain)
	if cleanDecoy != "" {
		dest = "127.0.0.1:8444"
		serverNames = []string{cleanDecoy}
		// Remnawave validation: name <= 30 chars and regex /^[A-Za-z0-9_\s-]+$/ (no parentheses or dots)
		profileName = "SelfSteal Reality"
		// Inbound tags must be unique in Remnawave database (A113)
		inboundTag = "selfsteal-reality-in"
	}

	if portMode == "stealth443" {
		dest = "/dev/shm/nginx.sock"
		xver = 1
		profileName = "Stealth Reality 443"
		inboundTag = "stealth-reality-in"
	}

	var rules []any
	rules = append(rules, map[string]any{
		"ip":          []string{"geoip:private"},
		"outboundTag": "block",
	})
	if torrentBlocker {
		rules = append(rules, map[string]any{
			"protocol":    []string{"bittorrent"},
			"outboundTag": "block",
		})
	}
	if blockSpamPorts {
		rules = append(rules, map[string]any{
			"port":        "25,465,587",
			"outboundTag": "block",
		})
	}

	configObj := map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
		},
		"routing": map[string]any{
			"domainStrategy": "IPIfNonMatch",
			"rules":          rules,
		},
		"inbounds": []any{
			map[string]any{
				"tag":      inboundTag,
				"listen":   "0.0.0.0",
				"port":     443,
				"protocol": "vless",
				"settings": map[string]any{
					"clients":    []any{},
					"decryption": "none",
				},
				"streamSettings": map[string]any{
					"network":  "tcp",
					"security": "reality",
					"realitySettings": map[string]any{
						"show":        false,
						"dest":        dest,
						"xver":        xver,
						"serverNames": serverNames,
						"privateKey":  privateKey,
						"shortIds":    []string{""},
					},
				},
				"sniffing": map[string]any{
					"enabled":      true,
					"destOverride": []string{"http", "tls", "quic"},
				},
			},
		},
		"outbounds": []any{
			map[string]any{
				"protocol": "freedom",
				"tag":      "direct",
			},
			map[string]any{
				"protocol": "blackhole",
				"tag":      "block",
			},
		},
	}

	payload := map[string]any{
		"name":   profileName,
		"config": configObj,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("ошибка сериализации: %v", err)
	}

	req, err := http.NewRequest("POST", targetURL+"/api/config-profiles", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("сбой запроса создания профиля: %v", err)
	}
	defer resp.Body.Close()

	postBodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// If 409 Conflict or profile name already exists, the panel already has this profile!
		if resp.StatusCode == http.StatusConflict || strings.Contains(string(postBodyBytes), "already exists") {
			// 1. Try to re-fetch existing profile from GET /api/config-profiles
			profileUUID, inbounds, errFetch := fetchActiveConfigProfile(targetURL, token, decoyDomain, client)
			if errFetch == nil && profileUUID != "" {
				return profileUUID, inbounds, nil
			}

			// 2. If existing profile was not discoverable under that name, retry creating with a unique timestamped name
			prefix := "Reality"
			uniqueTag := fmt.Sprintf("in-%s", time.Now().Format("150405"))
			if cleanDecoy != "" {
				prefix = "SelfSteal"
				uniqueTag = fmt.Sprintf("self-%s", time.Now().Format("150405"))
			}
			uniqueName := fmt.Sprintf("%s-%s", prefix, time.Now().Format("150405"))
			payload["name"] = uniqueName
			if inboundsArr, ok := configObj["inbounds"].([]any); ok && len(inboundsArr) > 0 {
				if ibMap, okMap := inboundsArr[0].(map[string]any); okMap {
					ibMap["tag"] = uniqueTag
				}
			}
			newBytes, _ := json.Marshal(payload)
			reqRetry, errRetry := http.NewRequest("POST", targetURL+"/api/config-profiles", bytes.NewBuffer(newBytes))
			if errRetry == nil {
				reqRetry.Header.Set("Authorization", "Bearer "+token)
				reqRetry.Header.Set("Content-Type", "application/json")
				reqRetry.Header.Set("Accept", "application/json")
				respRetry, errDoRetry := client.Do(reqRetry)
				if errDoRetry == nil {
					defer respRetry.Body.Close()
					retryBody, _ := io.ReadAll(respRetry.Body)
					retryProfiles := parseProfilesFromBytes(retryBody)
					if len(retryProfiles) > 0 {
						uuid, inbounds := extractProfileUUIDAndInbounds(targetURL, token, client, retryProfiles[0])
						if uuid != "" {
							return uuid, inbounds, nil
						}
					}
					// Re-fetch active profile after retry creation
					pUUID, pInbounds, errFetchRetry := fetchActiveConfigProfile(targetURL, token, decoyDomain, client)
					if errFetchRetry == nil && pUUID != "" {
						return pUUID, pInbounds, nil
					}
				}
			}
		}
		return "", nil, fmt.Errorf("панель вернула ошибку создания профиля (%d): %s", resp.StatusCode, string(postBodyBytes))
	}

	// 1. Try to extract directly from POST response body
	postProfiles := parseProfilesFromBytes(postBodyBytes)
	if len(postProfiles) > 0 {
		uuid, inbounds := extractProfileUUIDAndInbounds(targetURL, token, client, postProfiles[0])
		if uuid != "" {
			return uuid, inbounds, nil
		}
	}

	// 2. Re-fetch via GET /api/config-profiles
	profileUUID, inbounds, errFetch := fetchActiveConfigProfile(targetURL, token, decoyDomain, client)
	if errFetch != nil || profileUUID == "" {
		return "", nil, fmt.Errorf("профиль создан, но не удалось получить его идентификаторы. Ответ панели: %s", string(postBodyBytes))
	}

	return profileUUID, inbounds, nil
}

// CreateNodeInPanel creates a node via POST /api/nodes and retrieves secretKey
func CreateNodeInPanel(panelURL, apiToken string, req types.CreateNodeRequest) types.CreateNodeResponse {
	targetURL := cleanURL(panelURL)
	cleanToken := strings.TrimSpace(apiToken)

	if targetURL == "" || cleanToken == "" {
		return types.CreateNodeResponse{
			Success: false,
			Error:   "URL панели и API токен обязательны",
		}
	}

	client := getHTTPClient()

	if req.Port <= 0 {
		req.Port = 2222
	}
	if req.TrafficMultiplier <= 0 {
		req.TrafficMultiplier = 1.0
	}
	if req.Country == "" {
		req.Country = "DE"
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("Node-%s-%d", req.Country, req.Port)
	}

	// Self-healing: check if a stale/disconnected node with this IP address already exists in panel
	existingNodes, errGet := GetNodes(targetURL, cleanToken)
	if errGet == nil {
		for _, n := range existingNodes {
			if strings.TrimSpace(n.Address) == strings.TrimSpace(req.Address) {
				// Clean up orphaned/disconnected node entry so a fresh node with a valid SECRET_KEY can be registered
				delReq, errDel := http.NewRequest("DELETE", fmt.Sprintf("%s/api/nodes/%s", targetURL, n.UUID), nil)
				if errDel == nil {
					delReq.Header.Set("Authorization", "Bearer "+cleanToken)
					delResp, errDoDel := client.Do(delReq)
					if errDoDel == nil {
						delResp.Body.Close()
					}
				}
				break
			}
		}
	}

	var profileUUID string
	var inbounds []string
	var errProfile error

	if req.IntegrationMode == "custom" && req.ConfigProfileUUID != "" {
		profileUUID = req.ConfigProfileUUID
		if req.InboundUUID != "" {
			inbounds = []string{req.InboundUUID}
		} else {
			_, ibs := extractProfileUUIDAndInbounds(targetURL, cleanToken, client, map[string]any{"uuid": profileUUID})
			if len(ibs) > 0 {
				inbounds = ibs
			}
		}
	} else {
		// Turnkey / Default Mode: Search or auto-create Reality profile
		profileUUID, inbounds, errProfile = fetchActiveConfigProfile(targetURL, cleanToken, req.DecoyDomain, client)
		if errProfile != nil || profileUUID == "" {
			// Zero-Touch auto-creation: if panel has no config profile, create standard VLESS Reality profile!
			var errCreate error
			profileUUID, inbounds, errCreate = createDefaultConfigProfile(targetURL, cleanToken, req.DecoyDomain, req.PortMode, req.TorrentBlocker, req.BlockSpamPorts, client)
			if errCreate != nil {
				return types.CreateNodeResponse{
					Success: false,
					Error:   fmt.Sprintf("В панели нет Config Profile и авто-создание не удалось: %v", errCreate),
				}
			}
		}
	}

	payload := map[string]any{
		"name":              req.Name,
		"country":           req.Country,
		"countryCode":       req.Country,
		"address":           req.Address,
		"port":              req.Port,
		"trafficMultiplier": req.TrafficMultiplier,
	}

	if profileUUID != "" {
		payload["configProfile"] = map[string]any{
			"activeConfigProfileUuid": profileUUID,
			"activeInbounds":          inbounds,
		}
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return types.CreateNodeResponse{Success: false, Error: "Ошибка сериализации данных ноды"}
	}

	httpReq, err := http.NewRequest("POST", targetURL+"/api/nodes", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return types.CreateNodeResponse{Success: false, Error: fmt.Sprintf("Ошибка формирования запроса: %v", err)}
	}

	httpReq.Header.Set("Authorization", "Bearer "+cleanToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "Client.Timeout") {
			return types.CreateNodeResponse{
				Success: false,
				Error:   "Время ожидания ответа от панели истекло (25 сек). Сервер перегружен или недоступен.",
			}
		}
		return types.CreateNodeResponse{Success: false, Error: fmt.Sprintf("Не удалось отправить запрос к панели: %v", err)}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return types.CreateNodeResponse{
			Success: false,
			Error:   fmt.Sprintf("Панель вернула ошибку (%d): %s", resp.StatusCode, string(bodyBytes)),
		}
	}

	// Parse node response for secretKey and uuid
	var rawMap map[string]any
	if err := json.Unmarshal(bodyBytes, &rawMap); err != nil {
		return types.CreateNodeResponse{
			Success: false,
			Error:   "Не удалось разобрать JSON ответ панели",
		}
	}

	// Check if wrapped in "response"
	data := rawMap
	if nested, ok := rawMap["response"].(map[string]any); ok {
		data = nested
	}

	nodeUUID := ""
	if u, ok := data["uuid"].(string); ok {
		nodeUUID = u
	} else if id, ok := data["id"].(string); ok {
		nodeUUID = id
	}

	secretKey := ""
	if sk, ok := data["secretKey"].(string); ok {
		secretKey = sk
	} else if sk, ok := data["secret_key"].(string); ok {
		secretKey = sk
	} else if sk, ok := data["nodeSecret"].(string); ok {
		secretKey = sk
	}

	hostCreated := false
	squadUpdated := false

	// Host creation logic
	shouldCreateHost := false
	hostAddr := req.Address
	hostSNI := ""
	remark := fmt.Sprintf("%s - Reality", req.Name)

	if req.IntegrationMode == "custom" {
		if req.CreateHost {
			shouldCreateHost = true
			if req.HostAddress != "" {
				hostAddr = req.HostAddress
			}
			hostSNI = req.HostSNI
		}
	} else {
		// Turnkey: always create host for client subscriptions
		shouldCreateHost = true
		if strings.TrimSpace(req.DecoyDomain) != "" {
			hostAddr = strings.TrimSpace(req.DecoyDomain)
			hostSNI = hostAddr
			remark = fmt.Sprintf("%s - SelfSteal", req.Name)
		}
	}

	if shouldCreateHost && len(inbounds) > 0 && hostAddr != "" && profileUUID != "" {
		primaryInbound := inbounds[0]
		errH := CreateHostInPanel(targetURL, cleanToken, profileUUID, primaryInbound, hostAddr, remark, hostSNI, 443)
		if errH == nil {
			hostCreated = true
		}
	}

	// Squad update logic
	if len(inbounds) > 0 {
		primaryInbound := inbounds[0]
		if req.IntegrationMode == "custom" {
			if req.SquadUUID != "" {
				errSq := AddInboundToSquad(targetURL, cleanToken, req.SquadUUID, primaryInbound)
				if errSq == nil {
					squadUpdated = true
				}
			}
		} else {
			// Turnkey: automatically add to default squad
			squads, _ := GetInternalSquads(targetURL, cleanToken)
			if len(squads) > 0 {
				targetSquadUUID := squads[0].UUID
				for _, sq := range squads {
					if strings.Contains(strings.ToLower(sq.Name), "default") {
						targetSquadUUID = sq.UUID
						break
					}
				}
				errSq := AddInboundToSquad(targetURL, cleanToken, targetSquadUUID, primaryInbound)
				if errSq == nil {
					squadUpdated = true
				}
			}
		}
	}

	return types.CreateNodeResponse{
		Success:      true,
		NodeUUID:     nodeUUID,
		SecretKey:    secretKey,
		HostCreated:  hostCreated,
		SquadUpdated: squadUpdated,
	}
}

// CreateHostInPanel creates a public Host entry in Remnawave for client subscriptions
func CreateHostInPanel(panelURL, apiToken, profileUUID, inboundUUID, hostAddress, remark, sni string, port int) error {
	targetURL := cleanURL(panelURL)
	cleanToken := strings.TrimSpace(apiToken)
	if targetURL == "" || cleanToken == "" || hostAddress == "" {
		return fmt.Errorf("не указан адрес панели или хоста")
	}
	if port <= 0 {
		port = 443
	}
	if remark == "" {
		remark = "Reality Host"
	}
	if len(remark) > 30 {
		remark = remark[:30]
	}

	client := getHTTPClient()

	// Remnawave API Host DTO: expects nested inbound { configProfileUuid, configProfileInboundUuid }
	hostPayload := map[string]any{
		"inbound": map[string]any{
			"configProfileUuid":        profileUUID,
			"configProfileInboundUuid": inboundUUID,
		},
		"remark":  remark,
		"address": hostAddress,
		"port":    port,
	}
	if sni != "" {
		hostPayload["sni"] = sni
	}

	hBytes, _ := json.Marshal(hostPayload)
	hReq, err := http.NewRequest("POST", targetURL+"/api/hosts", bytes.NewBuffer(hBytes))
	if err != nil {
		return err
	}
	hReq.Header.Set("Authorization", "Bearer "+cleanToken)
	hReq.Header.Set("Content-Type", "application/json")
	hReq.Header.Set("Accept", "application/json")

	hResp, errH := client.Do(hReq)
	if errH != nil {
		return errH
	}
	defer hResp.Body.Close()

	if hResp.StatusCode >= 400 {
		// Fallback for older Remnawave panel builds with flat inboundUuid
		fbPayload := map[string]any{
			"inboundUuid": inboundUUID,
			"remark":      remark,
			"address":     hostAddress,
			"port":        port,
		}
		if sni != "" {
			fbPayload["sni"] = sni
		}
		fbBytes, _ := json.Marshal(fbPayload)
		fbReq, _ := http.NewRequest("POST", targetURL+"/api/hosts", bytes.NewBuffer(fbBytes))
		if fbReq != nil {
			fbReq.Header.Set("Authorization", "Bearer "+cleanToken)
			fbReq.Header.Set("Content-Type", "application/json")
			fbReq.Header.Set("Accept", "application/json")
			fbResp, errFb := client.Do(fbReq)
			if errFb == nil {
				defer fbResp.Body.Close()
				if fbResp.StatusCode < 400 {
					return nil
				}
			}
		}
		return fmt.Errorf("status %d", hResp.StatusCode)
	}

	return nil
}

// AddInboundToSquad adds an inbound to an internal squad without dropping existing inbounds
func AddInboundToSquad(panelURL, apiToken, squadUUID, inboundUUID string) error {
	targetURL := cleanURL(panelURL)
	cleanToken := strings.TrimSpace(apiToken)
	if targetURL == "" || cleanToken == "" || squadUUID == "" || inboundUUID == "" {
		return fmt.Errorf("недостаточно параметров для привязки к скваду")
	}

	client := getHTTPClient()

	// 1. Fetch current inbounds of the squad
	squads, err := GetInternalSquads(panelURL, apiToken)
	var currentInbounds []string
	if err == nil {
		for _, s := range squads {
			if s.UUID == squadUUID {
				currentInbounds = s.InboundUUIDs
				break
			}
		}
	}

	// 2. Append new inbound if not already present
	alreadyPresent := false
	for _, id := range currentInbounds {
		if id == inboundUUID {
			alreadyPresent = true
			break
		}
	}
	if !alreadyPresent {
		currentInbounds = append(currentInbounds, inboundUUID)
	}

	// 3. Send PATCH /api/internal-squads/{squadUUID}
	patchPayload := map[string]any{
		"inbounds": currentInbounds,
	}
	pBytes, _ := json.Marshal(patchPayload)
	pReq, err := http.NewRequest("PATCH", fmt.Sprintf("%s/api/internal-squads/%s", targetURL, squadUUID), bytes.NewBuffer(pBytes))
	if err != nil {
		return err
	}
	pReq.Header.Set("Authorization", "Bearer "+cleanToken)
	pReq.Header.Set("Content-Type", "application/json")
	pReq.Header.Set("Accept", "application/json")

	pResp, errP := client.Do(pReq)
	if errP != nil {
		return errP
	}
	defer pResp.Body.Close()

	if pResp.StatusCode >= 400 {
		// Fallback: Some Remnawave API versions use inboundUuids
		fbPayload := map[string]any{
			"inboundUuids": currentInbounds,
		}
		fbBytes, _ := json.Marshal(fbPayload)
		fbReq, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/api/internal-squads/%s", targetURL, squadUUID), bytes.NewBuffer(fbBytes))
		if fbReq != nil {
			fbReq.Header.Set("Authorization", "Bearer "+cleanToken)
			fbReq.Header.Set("Content-Type", "application/json")
			fbReq.Header.Set("Accept", "application/json")
			fbResp, errFb := client.Do(fbReq)
			if errFb == nil {
				defer fbResp.Body.Close()
				if fbResp.StatusCode < 400 {
					return nil
				}
			}
		}
		return fmt.Errorf("status %d", pResp.StatusCode)
	}

	return nil
}

// GetConfigProfiles retrieves all config profiles and their inbounds from the panel
func GetConfigProfiles(panelURL, apiToken string) ([]types.PanelProfileItem, error) {
	targetURL := cleanURL(panelURL)
	cleanToken := strings.TrimSpace(apiToken)

	req, err := http.NewRequest("GET", targetURL+"/api/config-profiles", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cleanToken)
	req.Header.Set("Accept", "application/json")

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	list := parseProfilesFromBytes(body)
	var items []types.PanelProfileItem

	for _, p := range list {
		profUUID, _ := p["uuid"].(string)
		if profUUID == "" {
			profUUID, _ = p["id"].(string)
		}
		profName, _ := p["name"].(string)
		if profName == "" {
			profName = "Unnamed Profile"
		}

		// Look for inbounds
		var rawInbounds []any
		if ibs, ok := p["inbounds"].([]any); ok {
			rawInbounds = ibs
		} else if cfg, ok := p["config"].(map[string]any); ok {
			if ibs, ok := cfg["inbounds"].([]any); ok {
				rawInbounds = ibs
			}
		}

		if len(rawInbounds) > 0 {
			for _, ib := range rawInbounds {
				if ibMap, ok := ib.(map[string]any); ok {
					ibUUID, _ := ibMap["uuid"].(string)
					if ibUUID == "" {
						ibUUID, _ = ibMap["id"].(string)
					}
					ibTag, _ := ibMap["tag"].(string)
					proto, _ := ibMap["protocol"].(string)
					portNum := 0
					if prt, ok := ibMap["port"].(float64); ok {
						portNum = int(prt)
					}

					items = append(items, types.PanelProfileItem{
						UUID:        profUUID,
						Name:        profName,
						InboundUUID: ibUUID,
						InboundTag:  ibTag,
						Port:        portNum,
						Protocol:    proto,
					})
				}
			}
		} else {
			items = append(items, types.PanelProfileItem{
				UUID: profUUID,
				Name: profName,
			})
		}
	}

	return items, nil
}

// GetInternalSquads retrieves all internal squads from the panel
func GetInternalSquads(panelURL, apiToken string) ([]types.PanelSquadItem, error) {
	targetURL := cleanURL(panelURL)
	cleanToken := strings.TrimSpace(apiToken)

	req, err := http.NewRequest("GET", targetURL+"/api/internal-squads", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cleanToken)
	req.Header.Set("Accept", "application/json")

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawList []map[string]any
	if err := json.Unmarshal(body, &rawList); err != nil {
		var wrapper map[string]any
		if err := json.Unmarshal(body, &wrapper); err == nil {
			if l, ok := wrapper["response"].([]any); ok {
				for _, item := range l {
					if m, ok := item.(map[string]any); ok {
						rawList = append(rawList, m)
					}
				}
			}
		}
	}

	var squads []types.PanelSquadItem
	for _, m := range rawList {
		sq := types.PanelSquadItem{}
		if u, ok := m["uuid"].(string); ok {
			sq.UUID = u
		}
		if n, ok := m["name"].(string); ok {
			sq.Name = n
		}

		var ibUUIDs []string
		if ibs, ok := m["inbounds"].([]any); ok {
			for _, ib := range ibs {
				if s, ok := ib.(string); ok {
					ibUUIDs = append(ibUUIDs, s)
				} else if obj, ok := ib.(map[string]any); ok {
					if u, ok := obj["uuid"].(string); ok {
						ibUUIDs = append(ibUUIDs, u)
					}
				}
			}
		}
		sq.InboundUUIDs = ibUUIDs
		sq.InboundsCount = len(ibUUIDs)
		squads = append(squads, sq)
	}

	return squads, nil
}

// GetNodes fetches the list of nodes from Remnawave Panel
func GetNodes(panelURL, apiToken string) ([]types.PanelNodeItem, error) {
	targetURL := cleanURL(panelURL)
	cleanToken := strings.TrimSpace(apiToken)

	req, err := http.NewRequest("GET", targetURL+"/api/nodes", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cleanToken)
	req.Header.Set("Accept", "application/json")

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawList []map[string]any
	if err := json.Unmarshal(body, &rawList); err != nil {
		var wrapper map[string]any
		if err := json.Unmarshal(body, &wrapper); err == nil {
			if l, ok := wrapper["response"].([]any); ok {
				for _, item := range l {
					if m, ok := item.(map[string]any); ok {
						rawList = append(rawList, m)
					}
				}
			}
		}
	}

	var nodes []types.PanelNodeItem
	for _, m := range rawList {
		node := types.PanelNodeItem{}
		if u, ok := m["uuid"].(string); ok {
			node.UUID = u
		}
		if n, ok := m["name"].(string); ok {
			node.Name = n
		}
		if a, ok := m["address"].(string); ok {
			node.Address = a
		}
		if p, ok := m["port"].(float64); ok {
			node.Port = int(p)
		}
		if c, ok := m["country"].(string); ok {
			node.Country = c
		}
		if conn, ok := m["isConnected"].(bool); ok {
			node.IsConnected = conn
		} else if conn, ok := m["is_connected"].(bool); ok {
			node.IsConnected = conn
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
