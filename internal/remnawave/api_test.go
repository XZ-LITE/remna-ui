package remnawave

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"remna-ui/internal/types"
)

func TestTestPanelAPI_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/nodes" {
			t.Errorf("expected path /api/nodes, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"uuid": "node-1", "name": "Node 1"},
			{"uuid": "node-2", "name": "Node 2"},
		})
	}))
	defer server.Close()

	res := TestPanelAPI(server.URL, "test-token-123")
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}
	if res.NodesCount != 2 {
		t.Errorf("expected 2 nodes, got %d", res.NodesCount)
	}
}

func TestTestPanelAPI_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	res := TestPanelAPI(server.URL, "invalid-token")
	if res.Success {
		t.Fatalf("expected failure on 401 unauthorized")
	}
	if res.Error == "" {
		t.Errorf("expected non-empty error message")
	}
}

func TestCreateNodeInPanel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/nodes" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]any{})
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/hosts" {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"success":true}`))
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/config-profiles" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"response": []any{
					map[string]any{
						"uuid": "test-profile-uuid",
						"name": "Standard VLESS Reality",
						"inbounds": []any{
							map[string]any{"uuid": "inbound-uuid-1"},
						},
					},
				},
			})
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/internal-squads" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"response": []any{
					map[string]any{
						"uuid":     "test-squad-uuid",
						"name":     "Default-Squad",
						"inbounds": []any{},
					},
				},
			})
			return
		}

		if (r.Method == http.MethodPatch || r.Method == http.MethodPut) && strings.HasPrefix(r.URL.Path, "/api/internal-squads/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
			return
		}

		if r.Method != http.MethodPost || r.URL.Path != "/api/nodes" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			return
		}

		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["address"] != "194.87.1.1" {
			t.Errorf("expected address 194.87.1.1, got %v", payload["address"])
		}

		cfgProfile, ok := payload["configProfile"].(map[string]any)
		if !ok || cfgProfile["activeConfigProfileUuid"] != "test-profile-uuid" {
			t.Errorf("expected configProfile with test-profile-uuid, got %v", payload["configProfile"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"uuid":      "node-uuid-abc",
				"secretKey": "SECRET_KEY_AUTO_GENERATED_BY_PANEL",
			},
		})
	}))
	defer server.Close()

	req := types.CreateNodeRequest{
		Name:              "DE - Node",
		Country:           "DE",
		Address:           "194.87.1.1",
		Port:              2222,
		TrafficMultiplier: 1.0,
	}

	resp := CreateNodeInPanel(server.URL, "valid-token", req)
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if resp.NodeUUID != "node-uuid-abc" {
		t.Errorf("expected uuid node-uuid-abc, got %s", resp.NodeUUID)
	}
	if resp.SecretKey != "SECRET_KEY_AUTO_GENERATED_BY_PANEL" {
		t.Errorf("expected SECRET_KEY_AUTO_GENERATED_BY_PANEL, got %s", resp.SecretKey)
	}
}

func TestCreateNodeInPanel_AutoCreateConfigProfile(t *testing.T) {
	profilesCreated := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/nodes" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]any{})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/hosts" {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"success":true}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/config-profiles" {
			if !profilesCreated {
				// No profiles yet
				json.NewEncoder(w).Encode(map[string]any{"response": []any{}})
			} else {
				// After creation
				json.NewEncoder(w).Encode(map[string]any{
					"response": []any{
						map[string]any{
							"uuid": "auto-created-profile-uuid",
							"name": "Standard VLESS Reality",
							"inbounds": []any{
								map[string]any{"uuid": "auto-inbound-uuid-1"},
							},
						},
					},
				})
			}
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/config-profiles" {
			profilesCreated = true
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["name"] != "Standard VLESS Reality" {
				t.Errorf("expected profile name 'Standard VLESS Reality', got %v", payload["name"])
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"uuid": "auto-created-profile-uuid"})
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/nodes" {
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			cfgProfile, ok := payload["configProfile"].(map[string]any)
			if !ok || cfgProfile["activeConfigProfileUuid"] != "auto-created-profile-uuid" {
				t.Errorf("expected auto-created profile uuid, got %v", payload["configProfile"])
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"response": map[string]any{
					"uuid":      "new-node-uuid",
					"secretKey": "NEW_SECRET_KEY",
				},
			})
			return
		}
	}))
	defer server.Close()

	req := types.CreateNodeRequest{
		Name:    "DE - Node",
		Country: "DE",
		Address: "194.87.1.1",
		Port:    2222,
	}

	resp := CreateNodeInPanel(server.URL, "valid-token", req)
	if !resp.Success {
		t.Fatalf("expected auto-create success, got error: %s", resp.Error)
	}
	if resp.NodeUUID != "new-node-uuid" {
		t.Errorf("expected new-node-uuid, got %s", resp.NodeUUID)
	}
	if !profilesCreated {
		t.Errorf("expected profile to be auto-created")
	}
}

func TestGetNodes_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"uuid":        "uuid-1",
				"name":        "Germany #1",
				"address":     "1.2.3.4",
				"port":        2222.0,
				"country":     "DE",
				"isConnected": true,
			},
		})
	}))
	defer server.Close()

	nodes, err := GetNodes(server.URL, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Name != "Germany #1" || !nodes[0].IsConnected {
		t.Errorf("unexpected node values: %+v", nodes[0])
	}
}

func TestCreateNodeInPanel_NestedConfigProfilesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/config-profiles" {
			// Typical NestJS Remnawave v2/v3 structure:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"response": map[string]any{
					"configProfiles": []map[string]any{
						{
							"uuid": "nested-profile-uuid",
							"name": "Standard VLESS Reality",
							"inbounds": []map[string]any{
								{"uuid": "nested-inbound-uuid"},
							},
						},
					},
				},
			})
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/nodes" {
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			cfgProfile, ok := payload["configProfile"].(map[string]any)
			if !ok || cfgProfile["activeConfigProfileUuid"] != "nested-profile-uuid" {
				t.Errorf("expected nested-profile-uuid, got %v", payload["configProfile"])
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"response": map[string]any{
					"uuid":      "node-ok-uuid",
					"secretKey": "SECRET_OK",
				},
			})
			return
		}
	}))
	defer server.Close()

	resp := CreateNodeInPanel(server.URL, "token", types.CreateNodeRequest{
		Name:    "Test Node",
		Country: "DE",
		Address: "1.2.3.4",
	})
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if resp.NodeUUID != "node-ok-uuid" {
		t.Errorf("expected node-ok-uuid, got %s", resp.NodeUUID)
	}
}

func TestCreateNodeInPanel_Conflict409GracefulResolution(t *testing.T) {
	profilesQueried := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/config-profiles" {
			profilesQueried++
			if profilesQueried == 1 {
				// Initially return empty
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]any{})
				return
			}
			// On second call (after 409 conflict): return profile
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]any{
				{
					"uuid": "recovered-profile-uuid",
					"name": "Standard VLESS Reality",
					"inbounds": []map[string]any{
						{"uuid": "inbound-rec-1"},
					},
				},
			})
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/config-profiles" {
			// Simulate 409 Conflict: Config profile name already exists
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"message":"Config profile name already exists in database. Config profile names must be unique.","errorCode":"A114"}`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/nodes" {
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			cfgProfile, ok := payload["configProfile"].(map[string]any)
			if !ok || cfgProfile["activeConfigProfileUuid"] != "recovered-profile-uuid" {
				t.Errorf("expected recovered-profile-uuid after 409 conflict, got %v", payload["configProfile"])
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"uuid":      "resolved-node-uuid",
				"secretKey": "KEY123",
			})
			return
		}
	}))
	defer server.Close()

	resp := CreateNodeInPanel(server.URL, "token", types.CreateNodeRequest{
		Name:    "Conflict-Recovery-Node",
		Country: "DE",
		Address: "1.2.3.4",
	})
	if !resp.Success {
		t.Fatalf("expected recovery after 409, got error: %s", resp.Error)
	}
	if resp.NodeUUID != "resolved-node-uuid" {
		t.Errorf("expected resolved-node-uuid, got %s", resp.NodeUUID)
	}
}

func TestCreateNodeInPanel_CustomMode(t *testing.T) {
	hostCreated := false
	squadUpdated := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && r.URL.Path == "/api/nodes" {
			json.NewEncoder(w).Encode([]any{})
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/hosts" {
			hostCreated = true
			var hPayload map[string]any
			json.NewDecoder(r.Body).Decode(&hPayload)
			if hPayload["address"] != "node-nl.example.com" {
				t.Errorf("expected host address node-nl.example.com, got %v", hPayload["address"])
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"success":true}`))
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/internal-squads" {
			json.NewEncoder(w).Encode(map[string]any{
				"response": []any{
					map[string]any{
						"uuid":     "my-vip-squad-uuid",
						"name":     "VIP-Squad",
						"inbounds": []any{"existing-inbound"},
					},
				},
			})
			return
		}

		if (r.Method == http.MethodPatch || r.Method == http.MethodPut) && r.URL.Path == "/api/internal-squads/my-vip-squad-uuid" {
			squadUpdated = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/nodes" {
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			cfgProfile, ok := payload["configProfile"].(map[string]any)
			if !ok || cfgProfile["activeConfigProfileUuid"] != "my-custom-profile-uuid" {
				t.Errorf("expected activeConfigProfileUuid my-custom-profile-uuid, got %v", payload["configProfile"])
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"response": map[string]any{
					"uuid":      "custom-node-uuid",
					"secretKey": "CUSTOM_KEY_777",
				},
			})
			return
		}

		t.Errorf("unexpected request in custom mode: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	req := types.CreateNodeRequest{
		Name:              "NL - VIP Node",
		Country:           "NL",
		Address:           "185.12.34.56",
		Port:              2222,
		IntegrationMode:   "custom",
		ConfigProfileUUID: "my-custom-profile-uuid",
		InboundUUID:       "my-custom-inbound-uuid",
		SquadUUID:         "my-vip-squad-uuid",
		CreateHost:        true,
		HostAddress:       "node-nl.example.com",
		HostSNI:           "node-nl.example.com",
	}

	resp := CreateNodeInPanel(server.URL, "token", req)
	if !resp.Success {
		t.Fatalf("expected custom mode success, got error: %s", resp.Error)
	}
	if resp.NodeUUID != "custom-node-uuid" {
		t.Errorf("expected custom-node-uuid, got %s", resp.NodeUUID)
	}
	if resp.SecretKey != "CUSTOM_KEY_777" {
		t.Errorf("expected CUSTOM_KEY_777, got %s", resp.SecretKey)
	}
	if !hostCreated || !resp.HostCreated {
		t.Errorf("expected host to be created in panel")
	}
	if !squadUpdated || !resp.SquadUpdated {
		t.Errorf("expected VIP squad to be updated")
	}
}

func TestCreateNodeInPanel_DeletesAllDuplicates(t *testing.T) {
	deletedUUIDs := make(map[string]bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/nodes" {
			// Return two duplicate nodes with the same address
			json.NewEncoder(w).Encode([]map[string]any{
				{"uuid": "node-dupe-1", "name": "Node 1", "address": "192.168.1.50", "port": 2222},
				{"uuid": "node-dupe-2", "name": "Node 2", "address": "192.168.1.50", "port": 2222},
				{"uuid": "node-other", "name": "Node 3", "address": "10.0.0.1", "port": 2222},
			})
			return
		}
		if r.Method == http.MethodDelete {
			if strings.HasPrefix(r.URL.Path, "/api/nodes/") {
				uuid := strings.TrimPrefix(r.URL.Path, "/api/nodes/")
				deletedUUIDs[uuid] = true
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{"success": true})
				return
			}
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/config-profiles" {
			json.NewEncoder(w).Encode([]map[string]any{
				{
					"uuid": "prof-1",
					"inbounds": []map[string]any{
						{"uuid": "inb-1", "tag": "VLESS-Reality", "protocol": "vless"},
					},
				},
			})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/nodes" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"response": map[string]any{
					"uuid":      "new-node-uuid",
					"secretKey": "KEY_12345",
				},
			})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/hosts" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"success": true})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/internal-squads" {
			json.NewEncoder(w).Encode([]any{})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	req := types.CreateNodeRequest{
		Name:    "Clean Node",
		Country: "DE",
		Address: "192.168.1.50",
		Port:    2222,
	}

	resp := CreateNodeInPanel(server.URL, "token", req)
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if !deletedUUIDs["node-dupe-1"] {
		t.Errorf("expected node-dupe-1 to be deleted")
	}
	if !deletedUUIDs["node-dupe-2"] {
		t.Errorf("expected node-dupe-2 to be deleted")
	}
	if deletedUUIDs["node-other"] {
		t.Errorf("node-other should NOT be deleted")
	}
}
