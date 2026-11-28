package remnawave

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIAudit_CleanURL(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{"example.com", "https://example.com"},
		{"http://192.0.2.1:3000", "http://192.0.2.1:3000"},
		{"http://192.0.2.1:3000/", "http://192.0.2.1:3000"},
		{"https://panel.example.com/api/nodes", "https://panel.example.com"},
		{"  https://panel.example.com:8443/  ", "https://panel.example.com:8443"},
	}

	for _, tc := range testCases {
		got := cleanURL(tc.input)
		if got != tc.want {
			t.Errorf("cleanURL(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestAPIAudit_GenerateX25519Key(t *testing.T) {
	key1, err1 := generateX25519Key()
	if err1 != nil {
		t.Fatalf("unexpected error generating key1: %v", err1)
	}
	key2, err2 := generateX25519Key()
	if err2 != nil {
		t.Fatalf("unexpected error generating key2: %v", err2)
	}

	if len(key1) < 32 {
		t.Errorf("X25519 key length too short: len=%d", len(key1))
	}
	if key1 == key2 {
		t.Errorf("Subsequent keys must be unique random values, got identical keys")
	}
}

func TestAPIAudit_TestPanelAPI_Errors(t *testing.T) {
	// 1. Empty URL and Token
	res := TestPanelAPI("", "")
	if res.Success || !strings.Contains(res.Error, "Укажите URL") {
		t.Errorf("Expected validation error for empty URL, got: %v", res)
	}

	// 2. HTTP 500 Server Error
	server500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`Internal database crash`))
	}))
	defer server500.Close()

	res500 := TestPanelAPI(server500.URL, "token")
	if res500.Success || !strings.Contains(res500.Error, "500") {
		t.Errorf("Expected 500 error, got: %v", res500)
	}

	// 3. Malformed JSON
	serverBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{not valid json`))
	}))
	defer serverBadJSON.Close()

	resBadJSON := TestPanelAPI(serverBadJSON.URL, "token")
	// Should not crash, and should return 0 nodes
	if !resBadJSON.Success || resBadJSON.NodesCount != 0 {
		t.Errorf("Expected graceful handling of non-array JSON, got: %v", resBadJSON)
	}
}

func TestAPIAudit_GetInternalSquads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"response": []any{
				map[string]any{"uuid": "squad-1", "name": "Squad VIP"},
				map[string]any{"uuid": "squad-2", "name": "Squad Free"},
			},
		})
	}))
	defer server.Close()

	squads, err := GetInternalSquads(server.URL, "token")
	if err != nil {
		t.Fatalf("GetInternalSquads failed: %v", err)
	}
	if len(squads) != 2 {
		t.Fatalf("Expected 2 squads, got %d", len(squads))
	}
	if squads[0].Name != "Squad VIP" || squads[1].Name != "Squad Free" {
		t.Errorf("Squad names mismatch: %v", squads)
	}
}
