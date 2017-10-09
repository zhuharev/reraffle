package models

import (
	"github.com/go-xorm/xorm"
	dry "github.com/ungerik/go-dry"
)

const (
	endFile        = "endtext"
	notifyFile     = "texttext"
	notAWinnerFile = "not_a_winner"
)

var (
	db *xorm.Engine
)

// NewContext open database
func NewContext() (err error) {
	db, err = xorm.NewEngine("sqlite3", "db.sqlite")
	if err != nil {
		return
	}

	db.ShowSQL(true)

	err = db.Sync2(new(InfoSended),
		new(Crm),
		new(WebHookLog))

	return
}

// EndTextUpdate update end text notification
func EndTextUpdate(text string) error {
	return dry.FileSetString(endFile, text)
}

// EndTextGet return and text
func EndTextGet(publicID int) (string, error) {
	p := GetGroup(publicID)
	if p.EndText != "" {
		return p.EndText, nil
	}
	return dry.FileGetString(endFile)
}

// NotifyTextUpdate update end text notification
func NotifyTextUpdate(text string) error {
	return dry.FileSetString(notifyFile, text)
}

// NotifyTextGet return and text
func NotifyTextGet(publicID int) (string, error) {
	p := GetGroup(publicID)
	if p.NotifyText != "" {
		return p.NotifyText, nil
	}
	return dry.FileGetString(notifyFile)
}

// NotifyTextUpdate update end text notification
func NotAWinnerTextUpdate(text string) error {
	return dry.FileSetString(notAWinnerFile, text)
}

// NotifyTextGet return and text
func NotAWinnerTextGet(publicID int) (string, error) {
	// p := GetGroup(publicID)
	// if p.NotifyText != "" {
	// 	return p.NotifyText, nil
	// }
	return dry.FileGetString(notAWinnerFile)
}
