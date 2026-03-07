package graphclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultGraphBaseURL = "https://graph.microsoft.com/v1.0"

type Config struct {
	AccessToken string
	BaseURL     string
	Timeout     time.Duration
}

type MailFolder struct {
	ID          string
	DisplayName string
}

type MessageSummary struct {
	ID      string
	Subject string
	From    string
	To      string
	Date    time.Time
	Size    int64
	Flags   []string
}

type MessageDetail struct {
	ID       string
	Subject  string
	From     string
	To       string
	Cc       string
	Date     time.Time
	Size     int64
	Flags    []string
	Headers  map[string]string
	TextBody string
	HTMLBody string
}

type graphEnvelope[T any] struct {
	Value      []T    `json:"value"`
	Count      int    `json:"@odata.count"`
	NextLink   string `json:"@odata.nextLink"`
	ContextURL string `json:"@odata.context"`
}

type graphErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type mailFolderEntity struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type emailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type recipient struct {
	EmailAddress emailAddress `json:"emailAddress"`
}

type internetMessageHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type messageBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type messageEntity struct {
	ID                     string                  `json:"id"`
	Subject                string                  `json:"subject"`
	From                   recipient               `json:"from"`
	ToRecipients           []recipient             `json:"toRecipients"`
	CcRecipients           []recipient             `json:"ccRecipients"`
	ReceivedDateTime       string                  `json:"receivedDateTime"`
	Size                   int64                   `json:"size"`
	IsRead                 bool                    `json:"isRead"`
	InternetMessageHeaders []internetMessageHeader `json:"internetMessageHeaders"`
	Body                   messageBody             `json:"body"`
}

func ResolveMailFolder(ctx context.Context, cfg Config, mailbox string) (string, string, error) {
	target := strings.TrimSpace(mailbox)
	if target == "" || strings.EqualFold(target, "inbox") {
		return "inbox", "INBOX", nil
	}

	var byID mailFolderEntity
	query := url.Values{}
	query.Set("$select", "id,displayName")
	if err := requestJSONWithSelectFallback(ctx, cfg, http.MethodGet, "/me/mailFolders/"+url.PathEscape(target), query, nil, &byID); err == nil {
		if strings.TrimSpace(byID.ID) != "" {
			name := strings.TrimSpace(byID.DisplayName)
			if name == "" {
				name = target
			}
			return strings.TrimSpace(byID.ID), name, nil
		}
	}

	items, err := ListMailFolders(ctx, cfg, 200)
	if err != nil {
		return "", "", err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.DisplayName), target) || strings.EqualFold(strings.TrimSpace(item.ID), target) {
			return strings.TrimSpace(item.ID), strings.TrimSpace(item.DisplayName), nil
		}
	}
	return "", "", fmt.Errorf("mail folder %q not found", target)
}

func ListMailFolders(ctx context.Context, cfg Config, top int) ([]MailFolder, error) {
	if top <= 0 {
		top = 200
	}
	query := url.Values{}
	query.Set("$top", fmt.Sprintf("%d", top))
	query.Set("$select", "id,displayName")

	var payload graphEnvelope[mailFolderEntity]
	if err := requestJSONWithSelectFallback(ctx, cfg, http.MethodGet, "/me/mailFolders", query, nil, &payload); err != nil {
		return nil, err
	}

	items := make([]MailFolder, 0, len(payload.Value))
	for _, raw := range payload.Value {
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(raw.DisplayName)
		if name == "" {
			name = id
		}
		items = append(items, MailFolder{ID: id, DisplayName: name})
	}
	return items, nil
}

func ListMessages(
	ctx context.Context,
	cfg Config,
	folderID string,
	limit int,
	offset int,
) ([]MessageSummary, int, error) {
	if strings.TrimSpace(folderID) == "" {
		folderID = "inbox"
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	query := url.Values{}
	query.Set("$top", fmt.Sprintf("%d", limit))
	query.Set("$skip", fmt.Sprintf("%d", offset))
	query.Set("$count", "true")
	query.Set("$orderby", "receivedDateTime desc")
	query.Set("$select", "id,subject,from,toRecipients,receivedDateTime,size,isRead")

	headers := map[string]string{
		"ConsistencyLevel": "eventual",
	}

	var payload graphEnvelope[messageEntity]
	path := "/me/mailFolders/" + url.PathEscape(folderID) + "/messages"
	if err := requestJSONWithSelectFallback(ctx, cfg, http.MethodGet, path, query, headers, &payload); err != nil {
		return nil, 0, err
	}

	items := make([]MessageSummary, 0, len(payload.Value))
	for _, raw := range payload.Value {
		items = append(items, MessageSummary{
			ID:      strings.TrimSpace(raw.ID),
			Subject: strings.TrimSpace(raw.Subject),
			From:    formatRecipient(raw.From),
			To:      joinRecipients(raw.ToRecipients),
			Date:    parseGraphDate(raw.ReceivedDateTime),
			Size:    raw.Size,
			Flags:   buildFlags(raw.IsRead),
		})
	}

	total := payload.Count
	if total < offset+len(items) {
		total = offset + len(items)
	}
	return items, total, nil
}

func GetMessage(
	ctx context.Context,
	cfg Config,
	messageID string,
) (MessageDetail, error) {
	trimmedID := strings.TrimSpace(messageID)
	if trimmedID == "" {
		return MessageDetail{}, errors.New("message id is required")
	}

	query := url.Values{}
	query.Set("$select", "id,subject,from,toRecipients,ccRecipients,receivedDateTime,size,isRead,internetMessageHeaders,body")

	var payload messageEntity
	path := "/me/messages/" + url.PathEscape(trimmedID)
	if err := requestJSONWithSelectFallback(ctx, cfg, http.MethodGet, path, query, nil, &payload); err != nil {
		return MessageDetail{}, err
	}

	headers := make(map[string]string)
	for _, item := range payload.InternetMessageHeaders {
		key := strings.TrimSpace(item.Name)
		if key == "" {
			continue
		}
		headers[key] = item.Value
	}

	textBody := ""
	htmlBody := ""
	switch strings.ToLower(strings.TrimSpace(payload.Body.ContentType)) {
	case "text":
		textBody = payload.Body.Content
	default:
		htmlBody = payload.Body.Content
	}

	return MessageDetail{
		ID:       strings.TrimSpace(payload.ID),
		Subject:  strings.TrimSpace(payload.Subject),
		From:     formatRecipient(payload.From),
		To:       joinRecipients(payload.ToRecipients),
		Cc:       joinRecipients(payload.CcRecipients),
		Date:     parseGraphDate(payload.ReceivedDateTime),
		Size:     payload.Size,
		Flags:    buildFlags(payload.IsRead),
		Headers:  headers,
		TextBody: textBody,
		HTMLBody: htmlBody,
	}, nil
}

func requestJSON(
	ctx context.Context,
	cfg Config,
	method string,
	path string,
	query url.Values,
	headers map[string]string,
	output any,
) error {
	accessToken := strings.TrimSpace(cfg.AccessToken)
	if accessToken == "" {
		return errors.New("access token is required")
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultGraphBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	pathValue := path
	if !strings.HasPrefix(pathValue, "/") {
		pathValue = "/" + pathValue
	}

	u, err := url.Parse(baseURL + pathValue)
	if err != nil {
		return fmt.Errorf("invalid graph base url: %w", err)
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload graphErrorResponse
		if json.Unmarshal(bodyRaw, &payload) == nil {
			code := strings.TrimSpace(payload.Error.Code)
			message := strings.TrimSpace(payload.Error.Message)
			if code != "" || message != "" {
				if code == "" {
					code = fmt.Sprintf("http_%d", resp.StatusCode)
				}
				if message == "" {
					message = "graph request failed"
				}
				return fmt.Errorf("%s: %s", code, message)
			}
		}
		return fmt.Errorf("graph request failed with status %d", resp.StatusCode)
	}

	if output == nil || len(bodyRaw) == 0 {
		return nil
	}
	if err := json.Unmarshal(bodyRaw, output); err != nil {
		return fmt.Errorf("failed to parse graph response: %w", err)
	}
	return nil
}

func requestJSONWithSelectFallback(
	ctx context.Context,
	cfg Config,
	method string,
	path string,
	query url.Values,
	headers map[string]string,
	output any,
) error {
	err := requestJSON(ctx, cfg, method, path, query, headers, output)
	if err == nil {
		return nil
	}
	if !shouldRetryWithoutSelect(err, query) {
		return err
	}

	retryQuery := cloneValues(query)
	retryQuery.Del("$select")
	return requestJSON(ctx, cfg, method, path, retryQuery, headers, output)
}

func shouldRetryWithoutSelect(err error, query url.Values) bool {
	if err == nil || len(query) == 0 {
		return false
	}
	if strings.TrimSpace(query.Get("$select")) == "" {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "could not find a property named")
}

func cloneValues(source url.Values) url.Values {
	if len(source) == 0 {
		return url.Values{}
	}
	cloned := make(url.Values, len(source))
	for key, values := range source {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func parseGraphDate(raw string) time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func formatRecipient(raw recipient) string {
	address := strings.TrimSpace(raw.EmailAddress.Address)
	name := strings.TrimSpace(raw.EmailAddress.Name)
	switch {
	case name != "" && address != "":
		return name + " <" + address + ">"
	case address != "":
		return address
	default:
		return name
	}
}

func joinRecipients(items []recipient) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		value := formatRecipient(item)
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, ", ")
}

func buildFlags(isRead bool) []string {
	if isRead {
		return []string{"\\Seen"}
	}
	return []string{}
}
