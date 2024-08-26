package models

type Card struct {
	CardID         int    `json:"card_id"`
	DeckID         int    `json:"deck_id"`
	Correct        int    `json:"correct"`
	Incorrect      int    `json:"incorrect"`
	CardCreated    string `json:"card_created"`
	LastReviewDate string `json:"last_review_date"`
	Stage          string `json:"stage"`
	Lapses         int    `json:"lapses"`
	Ease           int    `json:"ease"`
	ReviewDueDate  string `json:"review_due_date"`
	Question       string `json:"question"`
	Answer         string `json:"answer"`
}
