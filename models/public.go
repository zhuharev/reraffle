package models

import "time"

type Public struct {
	Title string
	VkID  int

	LastRaffle int

	PromoCodeTemplate string
	InfoTemplate      string

	VkAccessToken string
	SheetID       string
	SheetName     string

	NotifyText string
	EndText    string

	Raffles []Raffle
}

type Raffle struct {
	StartDate time.Time
	EndDate   time.Time

	MaxWinners int

	Gropus  []int
	OwnerID int
	PostID  int

	Members []Member
}

type Member struct {
	VkID int
	Name string

	Place      int
	PromoCode  string
	InfoSended bool
}
