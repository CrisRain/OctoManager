package service

import "encoding/json"

func normalizeJSON(value json.RawMessage, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return value
}

// patchGraphConfigToken updates OAuth token fields on graph config JSON.
func patchGraphConfigToken(
	raw json.RawMessage,
	accessToken string,
	expiresAt string,
	refreshToken string,
	tokenURL string,
	scope []string,
) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	if m == nil {
		m = make(map[string]json.RawMessage)
	}
	if b, err := json.Marshal(accessToken); err == nil {
		m["access_token"] = b
	}
	if expiresAt != "" {
		if b, err := json.Marshal(expiresAt); err == nil {
			m["token_expires_at"] = b
		}
	}
	if refreshToken != "" {
		if b, err := json.Marshal(refreshToken); err == nil {
			m["refresh_token"] = b
		}
	}
	if tokenURL != "" {
		if b, err := json.Marshal(tokenURL); err == nil {
			m["token_url"] = b
		}
	}
	if len(scope) > 0 {
		if b, err := json.Marshal(scope); err == nil {
			m["scope"] = b
		}
	}
	updated, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return updated
}

// mergeJSONObjects merges base and overlay JSON objects.
// Keys in overlay take precedence; keys present only in base are kept.
// Returns base unchanged if overlay is empty or not a valid object.
func mergeJSONObjects(base, overlay json.RawMessage) json.RawMessage {
	if len(overlay) == 0 {
		return base
	}
	var baseMap map[string]json.RawMessage
	if err := json.Unmarshal(base, &baseMap); err != nil {
		return base
	}
	var overlayMap map[string]json.RawMessage
	if err := json.Unmarshal(overlay, &overlayMap); err != nil {
		return base
	}
	if baseMap == nil {
		baseMap = make(map[string]json.RawMessage)
	}
	// overlay wins on conflicts, but existing keys in base are preserved
	for k, v := range overlayMap {
		if _, exists := baseMap[k]; !exists {
			baseMap[k] = v
		}
	}
	merged, err := json.Marshal(baseMap)
	if err != nil {
		return base
	}
	return merged
}
