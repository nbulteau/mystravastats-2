package dto

type ErrorResponseMessageDto struct {
	Message     string `json:"message"`
	Description string `json:"description"`
	Code        int    `json:"code"`
}
