package dto

type BatchFailure struct {
	ID      uint64 `json:"id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BatchResult struct {
	Total    int            `json:"total"`
	Success  int            `json:"success"`
	Failed   int            `json:"failed"`
	Failures []BatchFailure `json:"failures"`
	Queued   bool           `json:"queued,omitempty"`
	TaskID   string         `json:"task_id,omitempty"`
	JobID    uint64         `json:"job_id,omitempty"`
}
