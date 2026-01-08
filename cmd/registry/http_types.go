package main

type RegisterRequest struct {
	Number string `json:"number"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
}

type RegisterResponse struct {
	OK bool `json:"ok"`
}

type LookupResponse struct {
	Number string `json:"number"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
}
