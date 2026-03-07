package dto

type PaginationQuery struct {
    Limit  int `form:"limit" binding:"omitempty,min=1,max=1000"`
    Offset int `form:"offset" binding:"omitempty,min=0"`
}

type PagedResponse[T any] struct {
    Items  []T   `json:"items"`
    Total  int64 `json:"total"`
    Limit  int   `json:"limit"`
    Offset int   `json:"offset"`
}
