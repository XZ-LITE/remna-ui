package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"remna-ui/internal/types"
)

var (
	httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	checkHostAPIBase = "https://check-host.net"
	ipAPIBase        = "http://ip-api.com/json"

	probeCacheMu  sync.RWMutex
	probeCache    = make(map[string]probeCacheEntry)
	probeCacheTTL = 75 * time.Second
)

type probeCacheEntry struct {
	result    types.IPBlockCheckResult
	expiresAt time.Time
}

func getCachedProbe(host string, port int) (types.IPBlockCheckResult, bool) {
	key := fmt.Sprintf("%s:%d", host, port)
	probeCacheMu.RLock()
	defer probeCacheMu.RUnlock()
	entry, found := probeCache[key]
	if found && time.Now().Before(entry.expiresAt) {
		return entry.result, true
	}
	return types.IPBlockCheckResult{}, false
}

func setCachedProbe(host string, port int, res types.IPBlockCheckResult) {
	key := fmt.Sprintf("%s:%d", host, port)
	probeCacheMu.Lock()
	defer probeCacheMu.Unlock()
	probeCache[key] = probeCacheEntry{
		result:    res,
		expiresAt: time.Now().Add(probeCacheTTL),
	}
}

// ResetProbeCache clears the in-memory cache (used in testing or forced refreshes)
func ResetProbeCache() {
	probeCacheMu.Lock()
	defer probeCacheMu.Unlock()
	probeCache = make(map[string]probeCacheEntry)
}

// CheckIPBlock executes a multi-layered TSPU and RKN connectivity check for a given IP address.
func CheckIPBlock(host string, port int) types.IPBlockCheckResult {
	cleanHost := strings.TrimSpace(host)
	cleanHost = strings.TrimPrefix(cleanHost, "http://")
	cleanHost = strings.TrimPrefix(cleanHost, "https://")
	cleanHost = strings.Trim(cleanHost, "[]/ ")
	if idx := strings.Index(cleanHost, ":"); idx != -1 && !strings.Contains(cleanHost, "]") && strings.Count(cleanHost, ":") == 1 {
		// host:port passed as host string
		pStr := cleanHost[idx+1:]
		cleanHost = cleanHost[:idx]
		if p, err := strconv.Atoi(pStr); err == nil && port <= 0 {
			port = p
		}
	}

	if port <= 0 {
		port = 22 // Default to SSH management port
	}

	// In-Memory TTL-Cache check (protects from Check-Host rate-limiting on repeated clicks)
	if cached, found := getCachedProbe(cleanHost, port); found {
		return cached
	}

	now := time.Now().Format("02.01.2006 15:04:05")

	if cleanHost == "" {
		return types.IPBlockCheckResult{
			Host:      cleanHost,
			Port:      port,
			Status:    "offline",
			Summary:   "Не указан IP-адрес для проверки.",
			CheckedAt: now,
		}
	}

	// 1. Gather Target GeoIP and Client Egress Location in parallel
	var (
		ispInfo       string
		locationInfo  string
		clientCC      string
		clientCountry string
		clientISP     string
	)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ispInfo, locationInfo = queryGeoIP(cleanHost)
	}()
	go func() {
		defer wg.Done()
		clientCC, clientCountry, clientISP = queryClientGeoIP()
	}()
	wg.Wait()

	// Detect if user PC is exiting through a foreign country (VPN / Proxy / Tunnel)
	isVPNDetected := clientCC != "" && clientCC != "RU"

	// 2. Perform Dual-Port Distributed & Local Probes (Primary port + Alt Port 443/22)
	altPort := 443
	if port == 443 {
		altPort = 22
	}

	var (
		localP1  types.ProbeNodeResult
		remoteP1 []types.ProbeNodeResult
		localP2  types.ProbeNodeResult
		remoteP2 []types.ProbeNodeResult
	)

	var probeWg sync.WaitGroup
	probeWg.Add(2)

	go func() {
		defer probeWg.Done()
		localP1 = runLocalProbe(cleanHost, port, isVPNDetected, clientCountry, clientISP)
		remoteP1, _ = queryCheckHost(cleanHost, port)
	}()

	go func() {
		defer probeWg.Done()
		localP2 = runLocalProbe(cleanHost, altPort, isVPNDetected, clientCountry, clientISP)
		remoteP2, _ = queryCheckHost(cleanHost, altPort)
	}()

	probeWg.Wait()

	// Combine probes for primary port (for backward compatibility)
	allP1 := make([]types.ProbeNodeResult, 0, len(remoteP1)+1)
	allP1 = append(allP1, localP1)
	allP1 = append(allP1, remoteP1...)

	allP2 := make([]types.ProbeNodeResult, 0, len(remoteP2)+1)
	allP2 = append(allP2, localP2)
	allP2 = append(allP2, remoteP2...)

	// 3. Classify individual port verdicts
	resP1 := classifyResults(cleanHost, port, localP1, remoteP1, ispInfo, locationInfo, clientCountry, isVPNDetected, allP1, now)
	resP2 := classifyResults(cleanHost, altPort, localP2, remoteP2, ispInfo, locationInfo, clientCountry, isVPNDetected, allP2, now)

	p1Label := fmt.Sprintf("Порт %d (SSH управление)", port)
	if port == 443 {
		p1Label = "Порт 443 (VPN / Reality)"
	}
	p2Label := fmt.Sprintf("Порт %d (VPN / Reality)", altPort)
	if altPort == 22 {
		p2Label = "Порт 22 (SSH управление)"
	}

	portDetails := []types.PortProbeSummary{
		{
			Port:     port,
			Protocol: p1Label,
			Status:   resP1.Status,
			Probes:   allP1,
		},
		{
			Port:     altPort,
			Protocol: p2Label,
			Status:   resP2.Status,
			Probes:   allP2,
		},
	}

	// 4. Synthesize unified multi-port verdict
	overallStatus := resP1.Status
	overallSummary := resP1.Summary
	isRKNBlocked := resP1.IsRKNBlocked || resP2.IsRKNBlocked

	if resP1.Status == "clean" && resP2.Status == "clean" {
		overallStatus = "clean"
		isRKNBlocked = false
		overallSummary = "Все ключевые порты доступны из РФ (SSH 22 и VPN 443 подтверждены, фильтрация ТСПУ отсутствует)."
		if isVPNDetected && clientCountry != "" {
			overallSummary += fmt.Sprintf(" Ваше локальное соединение через VPN (%s) также подтверждает доступность хоста.", clientCountry)
		}
	} else if (resP1.Status == "clean" || resP1.Status == "partial") && resP2.Status == "blocked" {
		overallStatus = "blocked"
		isRKNBlocked = true
		overallSummary = fmt.Sprintf("Высокий риск блокировки ТСПУ: порт SSH (%d) доступен, однако рабочий VPN-порт (%d) полностью заблокирован на магистральных операторах РФ! Клиенты не смогут подключиться к ноде без смены IP.", port, altPort)
	} else if resP1.Status == "blocked" && (resP2.Status == "clean" || resP2.Status == "partial") {
		overallStatus = "partial"
		isRKNBlocked = true
		overallSummary = fmt.Sprintf("Селективная блокировка ТСПУ: рабочий порт VPN (%d) доступен в РФ, но порт управления SSH (%d) сбрасывается ТСПУ.", altPort, port)
	} else if resP1.Status == "blocked" && resP2.Status == "blocked" {
		overallStatus = "blocked"
		isRKNBlocked = true
		overallSummary = fmt.Sprintf("Полная блокировка IP-адреса в РФ (ТСПУ / РКН): ни порт управления (%d), ни порт VPN (%d) не отвечают из России. Рекомендуется запросить замену IP у хостинга.", port, altPort)
	} else if resP1.Status == "offline" && resP2.Status == "clean" {
		overallStatus = "clean"
		isRKNBlocked = false
		overallSummary = fmt.Sprintf("Рабочий порт VPN (%d) чист и доступен в РФ. Порт SSH (%d) закрыт сетевым экраном хостинга.", altPort, port)
	} else if resP1.Status == "offline" && resP2.Status == "offline" {
		overallStatus = "offline"
		isRKNBlocked = false
		overallSummary = fmt.Sprintf("Сервер не отвечает ни на порту %d, ни на порту %d (ни из РФ, ни из Европы). Проверьте включен ли VPS и настроен ли файрвол.", port, altPort)
	}

	finalRes := types.IPBlockCheckResult{
		Host:          cleanHost,
		Port:          port,
		Status:        overallStatus,
		Summary:       overallSummary,
		ISPInfo:       ispInfo,
		LocationInfo:  locationInfo,
		ClientCountry: clientCountry,
		IsVPNDetected: isVPNDetected,
		Probes:        allP1,
		PortDetails:   portDetails,
		IsRKNBlocked:  isRKNBlocked,
		CheckedAt:     now,
	}

	setCachedProbe(cleanHost, port, finalRes)
	return finalRes
}

func runLocalProbe(host string, port int, isVPNDetected bool, clientCountry, clientISP string) types.ProbeNodeResult {
	start := time.Now()
	target := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", target, 2500*time.Millisecond)
	latency := int(time.Since(start).Milliseconds())

	locName := "Локальный ПК (Ваш провайдер)"
	cityName := "Текущий узел"
	if isVPNDetected && clientCountry != "" {
		locName = fmt.Sprintf("Локальный ПК (VPN: %s)", clientCountry)
		if clientISP != "" {
			cityName = fmt.Sprintf("Шлюз: %s", clientISP)
		}
	}

	if err != nil {
		errStr := "Таймаут соединения"
		if strings.Contains(strings.ToLower(err.Error()), "refused") {
			errStr = "Порт закрыт (RST)"
		} else if strings.Contains(strings.ToLower(err.Error()), "network is unreachable") {
			errStr = "Сеть недоступна"
		}
		return types.ProbeNodeResult{
			NodeID:      "local_pc",
			Location:    locName,
			CountryCode: "local",
			City:        cityName,
			Success:     false,
			LatencyMs:   0,
			Error:       errStr,
		}
	}
	_ = conn.Close()

	if latency <= 0 {
		latency = 1
	}

	return types.ProbeNodeResult{
		NodeID:      "local_pc",
		Location:    locName,
		CountryCode: "local",
		City:        cityName,
		Success:     true,
		LatencyMs:   latency,
	}
}

type ipAPIResponse struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	Org         string `json:"org"`
	AS          string `json:"as"`
}

func queryGeoIP(host string) (string, string) {
	url := fmt.Sprintf("%s/%s?fields=status,country,countryCode,city,isp,org,as", ipAPIBase, host)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "Неизвестный хостинг", "Неизвестная локация"
	}
	req.Header.Set("User-Agent", "RemnaUI-Probe/2.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "Неизвестный хостинг", "Неизвестная локация"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Неизвестный хостинг", "Неизвестная локация"
	}

	var data ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Status != "success" {
		return "Неизвестный хостинг", "Неизвестная локация"
	}

	isp := data.ISP
	if data.AS != "" {
		asParts := strings.SplitN(data.AS, " ", 2)
		if len(asParts) > 0 {
			isp = fmt.Sprintf("%s (%s)", data.ISP, asParts[0])
		}
	}
	location := fmt.Sprintf("%s, %s", data.City, data.Country)
	return isp, location
}

func queryClientGeoIP() (string, string, string) {
	url := fmt.Sprintf("%s/?fields=status,country,countryCode,city,isp,org,as", ipAPIBase)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", "", ""
	}
	req.Header.Set("User-Agent", "RemnaUI-Probe/2.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", ""
	}

	var data ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Status != "success" {
		return "", "", ""
	}

	isp := data.ISP
	if data.AS != "" {
		asParts := strings.SplitN(data.AS, " ", 2)
		if len(asParts) > 0 {
			isp = fmt.Sprintf("%s (%s)", data.ISP, asParts[0])
		}
	}
	return data.CountryCode, data.Country, isp
}

type checkHostInitResponse struct {
	Ok        int    `json:"ok"`
	RequestID string `json:"request_id"`
	Error     string `json:"error,omitempty"`
}

type checkHostNodeInfo struct {
	Location []string `json:"location"`
	ASN      string   `json:"asn"`
}

func queryCheckHost(host string, port int) ([]types.ProbeNodeResult, error) {
	// Request TCP check for RU probes + reference DE node
	nodes := []string{"ru1.node.check-host.net", "ru2.node.check-host.net", "ru3.node.check-host.net", "de1.node.check-host.net"}
	var nodeParams []string
	for _, n := range nodes {
		nodeParams = append(nodeParams, "node="+n)
	}

	initURL := fmt.Sprintf("%s/check-tcp?host=%s:%d&%s", checkHostAPIBase, host, port, strings.Join(nodeParams, "&"))
	req, err := http.NewRequestWithContext(context.Background(), "GET", initURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "RemnaUI-Probe/2.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check-host init failed: %d", resp.StatusCode)
	}

	var initResp checkHostInitResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil || initResp.RequestID == "" {
		return nil, fmt.Errorf("check-host invalid init: %v", err)
	}

	// Adaptive polling loop: up to 4 iterations (initial 1200ms, then 800ms)
	// As soon as all requested nodes have finished their checks, exit immediately!
	var rawResults map[string]json.RawMessage
	resultURL := fmt.Sprintf("%s/check-result/%s", checkHostAPIBase, initResp.RequestID)

	for attempt := 0; attempt < 4; attempt++ {
		if attempt == 0 {
			time.Sleep(1200 * time.Millisecond)
		} else {
			time.Sleep(800 * time.Millisecond)
		}

		req2, err := http.NewRequestWithContext(context.Background(), "GET", resultURL, nil)
		if err != nil {
			continue
		}
		req2.Header.Set("Accept", "application/json")
		req2.Header.Set("User-Agent", "RemnaUI-Probe/2.0")

		resp2, err := httpClient.Do(req2)
		if err != nil {
			continue
		}

		var current map[string]json.RawMessage
		decErr := json.NewDecoder(resp2.Body).Decode(&current)
		resp2.Body.Close()
		if decErr != nil {
			continue
		}

		rawResults = current

		// Check if all requested nodes have finished
		allDone := true
		for _, nodeID := range nodes {
			raw, found := rawResults[nodeID]
			if !found || string(raw) == "null" || len(raw) == 0 {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
	}

	if rawResults == nil {
		return nil, fmt.Errorf("check-host result polling timed out")
	}

	var results []types.ProbeNodeResult

	nodeMeta := map[string]struct {
		Loc  string
		City string
		Code string
	}{
		"ru1.node.check-host.net": {Loc: "Москва (AS14576)", City: "Москва", Code: "ru"},
		"ru2.node.check-host.net": {Loc: "Москва (AS210644)", City: "Москва", Code: "ru"},
		"ru3.node.check-host.net": {Loc: "Санкт-Петербург (AS207713)", City: "Санкт-Петербург", Code: "ru"},
		"de1.node.check-host.net": {Loc: "Германия (Эталон AS24940)", City: "Франкфурт", Code: "de"},
	}

	for _, nodeID := range nodes {
		meta, exists := nodeMeta[nodeID]
		if !exists {
			meta = struct {
				Loc  string
				City string
				Code string
			}{Loc: nodeID, City: "Узел", Code: "net"}
		}

		raw, found := rawResults[nodeID]
		if !found || string(raw) == "null" || len(raw) == 0 {
			continue
		}

		// Check-Host returns array of checks, e.g. [{"address": "...", "time": 0.025}] or [{"error": "Connection timed out"}]
		var checks []map[string]interface{}
		if err := json.Unmarshal(raw, &checks); err != nil || len(checks) == 0 {
			continue
		}

		first := checks[0]
		if errVal, hasErr := first["error"]; hasErr && errVal != nil {
			errStr := fmt.Sprintf("%v", errVal)
			if strings.Contains(strings.ToLower(errStr), "timed out") {
				errStr = "Таймаут (ТСПУ / Блокировка)"
			} else if strings.Contains(strings.ToLower(errStr), "refused") {
				errStr = "Сброс соединения (RST)"
			}
			results = append(results, types.ProbeNodeResult{
				NodeID:      nodeID,
				Location:    meta.Loc,
				CountryCode: meta.Code,
				City:        meta.City,
				Success:     false,
				LatencyMs:   0,
				Error:       errStr,
			})
		} else if timeSec, hasTime := first["time"]; hasTime {
			var latMs int
			if tFloat, ok := timeSec.(float64); ok {
				latMs = int(tFloat * 1000)
			}
			if latMs <= 0 {
				latMs = 1
			}
			results = append(results, types.ProbeNodeResult{
				NodeID:      nodeID,
				Location:    meta.Loc,
				CountryCode: meta.Code,
				City:        meta.City,
				Success:     true,
				LatencyMs:   latMs,
			})
		}
	}

	return results, nil
}

func classifyResults(
	host string,
	port int,
	local types.ProbeNodeResult,
	remoteProbes []types.ProbeNodeResult,
	ispInfo string,
	locationInfo string,
	clientCountry string,
	isVPNDetected bool,
	allProbes []types.ProbeNodeResult,
	checkedAt string,
) types.IPBlockCheckResult {
	ruSuccessCount := 0
	ruTotalCount := 0
	euSuccess := false
	euChecked := false

	// Count RU probes
	for _, p := range remoteProbes {
		if p.CountryCode == "ru" {
			ruTotalCount++
			if p.Success {
				ruSuccessCount++
			}
		} else if p.CountryCode == "de" {
			euChecked = true
			euSuccess = p.Success
		}
	}

	status := "clean"
	summary := ""
	isRKNBlocked := false

	if ruTotalCount > 0 {
		if euChecked && euSuccess && ruSuccessCount == 0 {
			// Definitive TSPU/RKN block: EU connects fine, but ALL Russian probes failed!
			// Independent probes in Moscow (AS14576, AS210644) and SPb (AS207713) confirm the block.
			status = "blocked"
			isRKNBlocked = true
			if local.Success {
				vpnNote := "активного VPN / прокси на вашем компьютере"
				if isVPNDetected && clientCountry != "" {
					vpnNote = fmt.Sprintf("активного VPN-туннеля (%s)", clientCountry)
				}
				summary = fmt.Sprintf("Обнаружена блокировка ТСПУ / РКН в России: независимые контрольные узлы в Москве и Санкт-Петербурге не могут подключиться к серверу (таймаут). С вашего ПК соединение проходит только из-за %s. Пользователи из РФ без VPN подключиться не смогут! Рекомендуется сменить IP.", vpnNote)
			} else {
				summary = "Высокий риск блокировки ТСПУ / РКН: сервер доступен из Европы, но полностью заблокирован из России (таймаут во всех российских точках). Рекомендуется запросить замену IP у хостинга."
			}
		} else if ruSuccessCount == 0 && (!euChecked || !euSuccess) && !local.Success {
			// Completely offline or firewall closed
			status = "offline"
			summary = fmt.Sprintf("Сервер не отвечает на порту %d ни из РФ, ни из зарубежных точек. Проверьте, включен ли VPS и открыт ли сетевой экран (firewall).", port)
		} else if ruSuccessCount == 0 && (!euChecked || !euSuccess) && local.Success {
			status = "partial"
			summary = fmt.Sprintf("Сервер откликается только локально, но недоступен с публичных контрольных точек (порт %d). Проверьте настройки фаервола VPS.", port)
		} else if ruSuccessCount == ruTotalCount {
			// All RU probes passed!
			status = "clean"
			if isVPNDetected && clientCountry != "" {
				summary = fmt.Sprintf("IP-адрес свободен и отлично доступен из России. Пакеты успешно проходят через контрольные узлы Москвы и СПб без фильтрации ТСПУ (ваше локальное соединение через VPN: %s также подтверждает открытый порт %d).", clientCountry, port)
			} else {
				summary = "IP-адрес свободен и отлично доступен из России. Пакеты проходят без фильтрации ТСПУ. Можно безопасно развертывать сервер."
			}
		} else if ruSuccessCount > 0 && ruSuccessCount < ruTotalCount {
			// Partial block (some RU ISPs pass, others drop)
			status = "partial"
			isRKNBlocked = true
			summary = fmt.Sprintf("Частичная доступность из РФ (%d из %d российских зондов ответили). Возможна точечная фильтрация у отдельных магистральных операторов.", ruSuccessCount, ruTotalCount)
		} else if ruSuccessCount > 0 {
			status = "clean"
			summary = "IP-адрес доступен из России. Соединение подтверждено распределенными зондами."
		}
	} else {
		// Fallback when remote probes couldn't be queried (e.g. rate limit or offline check-host)
		if local.Success {
			if isVPNDetected && clientCountry != "" {
				status = "partial"
				summary = fmt.Sprintf("Внимание: внешние контрольные узлы были временно недоступны. С вашего ПК порт %d отвечает, но на компьютере активен VPN (%s). Данная проверка не может гарантировать отсутствие блокировки у пользователей в РФ без VPN! Рекомендуется отключить VPN и повторить опрос.", port, clientCountry)
			} else {
				status = "clean"
				summary = fmt.Sprintf("IP-адрес доступен с вашего ПК на порту %d (задержка %d ms). Локальный провайдер соединяется успешно.", port, local.LatencyMs)
			}
		} else {
			status = "offline"
			summary = fmt.Sprintf("Сервер не ответил на локальное TCP-рукопожатие на порту %d. Проверьте адрес и сетевой экран.", port)
		}
	}

	return types.IPBlockCheckResult{
		Host:          host,
		Port:          port,
		Status:        status,
		Summary:       summary,
		ISPInfo:       ispInfo,
		LocationInfo:  locationInfo,
		ClientCountry: clientCountry,
		IsVPNDetected: isVPNDetected,
		Probes:        allProbes,
		IsRKNBlocked:  isRKNBlocked,
		CheckedAt:     checkedAt,
	}
}
