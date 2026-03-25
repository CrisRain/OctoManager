package grpcclient

import "testing"

func TestDecodeManifestPreservesPluginUIAndActions(t *testing.T) {
	raw := []byte(`{
		"key":"octo_demo",
		"name":"Octo Demo",
		"schema":{"type":"object"},
		"settings":[{"key":"token","label":"Token"}],
		"ui":{"tabs":[{"key":"overview"}]},
		"capabilities":{
			"actions":[
				{"key":"VERIFY","description":"Verify credentials"},
				{"key":"SYNC","name":"Sync Profile","modes":["sync","job"]}
			],
			"webhook": true
		}
	}`)

	manifest, err := decodeManifest(raw, "fallback_key")
	if err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if manifest.Key != "octo_demo" {
		t.Fatalf("unexpected key %q", manifest.Key)
	}
	if manifest.Name != "Octo Demo" {
		t.Fatalf("unexpected name %q", manifest.Name)
	}
	if len(manifest.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(manifest.Actions))
	}
	if manifest.Actions[0].Name != "VERIFY" {
		t.Fatalf("expected missing action name to fall back to key, got %q", manifest.Actions[0].Name)
	}
	if len(manifest.Settings) != 1 {
		t.Fatalf("expected settings to be preserved")
	}
	if len(manifest.UI) == 0 {
		t.Fatalf("expected ui payload to be preserved")
	}
	if len(manifest.Capabilities) != 1 || manifest.Capabilities[0] != "webhook" {
		t.Fatalf("unexpected capabilities %#v", manifest.Capabilities)
	}
}

func TestDecodeManifestFallsBackToRegistryKey(t *testing.T) {
	manifest, err := decodeManifest([]byte(`{"name":"Demo"}`), "octo_demo")
	if err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if manifest.Key != "octo_demo" {
		t.Fatalf("expected fallback key, got %q", manifest.Key)
	}
	if manifest.Name != "Demo" {
		t.Fatalf("unexpected name %q", manifest.Name)
	}
}
