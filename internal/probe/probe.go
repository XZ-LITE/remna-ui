package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"remna-ui/internal/types"
)

var (
	httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	checkHostAPIBase = "https://check-host.net"
	ipAPIBase        = "http://ip-api.com/json"
)

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

	// 1. Gather ISP and Geolocation in parallel or sequentially
	ispInfo, locationInfo := queryGeoIP(cleanHost)

	// 2. Perform Local TCP Probe (from this machine)
	localRes := runLocalProbe(cleanHost, port)

	// 3. Perform Distributed Remote Probes via Check-Host
	remoteProbes, _ := queryCheckHost(cleanHost, port)

	// Combine probes
	allProbes := make([]types.ProbeNodeResult, 0, len(remoteProbes)+1)
	allProbes = append(allProbes, localRes)
	allProbes = append(allProbes, remoteProbes...)

	// 4. Classify Verdict
	return classifyResults(cleanHost, port, localRes, remoteProbes, ispInfo, locationInfo, allProbes, now)
}

func runLocalProbe(host string, port int) types.ProbeNodeResult {
	start := time.Now()
	target := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", target, 2500*time.Millisecond)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		errStr := "Таймаут соединения"
		if strings.Contains(strings.ToLower(err.Error()), "refused") {
			errStr = "Порт закрыт (RST)"
		} else if strings.Contains(strings.ToLower(err.Error()), "network is unreachable") {
			errStr = "Сеть недоступна"
		}
		return types.ProbeNodeResult{
			NodeID:      "local_pc",
			Location:    "Локальный ПК (Ваш провайдер)",
			CountryCode: "local",
			City:        "Текущий узел",
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
		Location:    "Локальный ПК (Ваш провайдер)",
		CountryCode: "local",
		City:        "Текущий узел",
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

	// Poll results (wait 2 seconds)
	time.Sleep(2200 * time.Millisecond)

	resultURL := fmt.Sprintf("%s/check-result/%s", checkHostAPIBase, initResp.RequestID)
	req2, err := http.NewRequestWithContext(context.Background(), "GET", resultURL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("User-Agent", "RemnaUI-Probe/2.0")

	resp2, err := httpClient.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	var rawResults map[string]json.RawMessage
	if err := json.NewDecoder(resp2.Body).Decode(&rawResults); err != nil {
		return nil, err
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

	// Incorporate local probe if user is in Russia
	// If remote RU probes are missing due to rate limiting, local probe serves as baseline
	status := "clean"
	summary := ""
	isRKNBlocked := false

	if ruTotalCount > 0 {
		if euChecked && euSuccess && ruSuccessCount == 0 && !local.Success {
			// Definitive TSPU/RKN block: EU connects fine, but RU and local fail
			status = "blocked"
			isRKNBlocked = true
			summary = "Высокий риск блокировки ТСПУ / РКН: сервер доступен из Европы, но полностью заблокирован из России (таймаут во всех российских точках). Рекомендуется запросить замену IP у хостинга."
		} else if ruSuccessCount == 0 && !local.Success && (!euChecked || !euSuccess) {
			// Completely offline or firewall closed
			status = "offline"
			summary = fmt.Sprintf("Сервер не отвечает на порту %d ни из РФ, ни из зарубежных точек. Проверьте, включен ли VPS и открыт ли сетевой экран (firewall).", port)
		} else if ruSuccessCount == ruTotalCount && (local.Success || ruSuccessCount >= 2) {
			// All RU probes passed
			status = "clean"
			summary = "IP-адрес свободен и отлично доступен из России. Пакеты проходят без фильтрации ТСПУ. Можно безопасно развертывать сервер."
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
		// Fallback when remote probes couldn't be queried
		if local.Success {
			status = "clean"
			summary = fmt.Sprintf("IP-адрес доступен с вашего ПК на порту %d (задержка %d ms). Локальный провайдер соединяется успешно.", port, local.LatencyMs)
		} else {
			status = "offline"
			summary = fmt.Sprintf("Сервер не ответил на локальное TCP-рукопожатие на порту %d. Проверьте адрес и сетевой экран.", port)
		}
	}

	return types.IPBlockCheckResult{
		Host:         host,
		Port:         port,
		Status:       status,
		Summary:      summary,
		ISPInfo:      ispInfo,
		LocationInfo: locationInfo,
		Probes:       allProbes,
		IsRKNBlocked: isRKNBlocked,
		CheckedAt:    checkedAt,
	}
}
