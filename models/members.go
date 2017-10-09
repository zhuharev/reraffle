package models

import "time"

// Row Sheet row
type Row struct {
	Date      string
	Place     int
	Name      string
	VkID      int
	Promocode string
	Prize     string
	Duration  string

	InfoSended     bool
	InfoSendedTime time.Time
	InfoReaded     bool
	InfoAnsered    bool
	DecodedDate    time.Time
}
