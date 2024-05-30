package mediawiki

type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

type Error struct {
	Code string `json:"code"`
}

type TokenResponse struct {
	BatchComplete string `json:"batchcomplete"`
	Query         struct {
		Tokens struct {
			LoginToken string `json:"logintoken"`
		} `json:"tokens"`
	} `json:"query"`
}
