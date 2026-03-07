package dto

import "time"

type ListEmailMessagesQuery struct {
	Mailbox string `form:"mailbox" binding:"omitempty"`
	Limit   int    `form:"limit" binding:"omitempty,min=1,max=200"`
	Offset  int    `form:"offset" binding:"omitempty,min=0"`
}

type GetEmailMessageQuery struct {
	Mailbox string `form:"mailbox" binding:"omitempty"`
}

type ListEmailMailboxesQuery struct {
	Reference string `form:"reference" binding:"omitempty"`
	Pattern   string `form:"pattern" binding:"omitempty"`
}

type EmailMessageSummary struct {
	ID      string    `json:"id"`
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	To      string    `json:"to,omitempty"`
	Date    time.Time `json:"date"`
	Size    int64     `json:"size"`
	Flags   []string  `json:"flags,omitempty"`
}

type EmailMessageDetail struct {
	ID       string            `json:"id"`
	Subject  string            `json:"subject"`
	From     string            `json:"from"`
	To       string            `json:"to,omitempty"`
	Cc       string            `json:"cc,omitempty"`
	Date     time.Time         `json:"date"`
	Size     int64             `json:"size"`
	Flags    []string          `json:"flags,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	TextBody string            `json:"text_body,omitempty"`
	HTMLBody string            `json:"html_body,omitempty"`
}

type ListEmailMessagesResponse struct {
	Mailbox string                `json:"mailbox"`
	Limit   int                   `json:"limit"`
	Offset  int                   `json:"offset"`
	Total   int                   `json:"total"`
	Items   []EmailMessageSummary `json:"items"`
}

type EmailMailbox struct {
	Name      string   `json:"name"`
	Delimiter string   `json:"delimiter,omitempty"`
	Flags     []string `json:"flags,omitempty"`
}

type ListEmailMailboxesResponse struct {
	Reference string         `json:"reference"`
	Pattern   string         `json:"pattern"`
	Items     []EmailMailbox `json:"items"`
}

type LatestEmailMessageResponse struct {
	Mailbox string              `json:"mailbox"`
	Found   bool                `json:"found"`
	Item    *EmailMessageDetail `json:"item,omitempty"`
}
