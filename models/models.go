package models

import (
	"github.com/go-xorm/xorm"
	dry "github.com/ungerik/go-dry"
)

const (
	endFile    = "endtext"
	notifyFile = "texttext"
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

	err = db.Sync2(new(InfoSended))

	return
}

// EndTextUpdate update end text notification
func EndTextUpdate(text string) error {
	return dry.FileSetString(endFile, text)
}

// EndTextGet return and text
func EndTextGet() (string, error) {
	return dry.FileGetString(endFile)
}

// NotifyTextUpdate update end text notification
func NotifyTextUpdate(text string) error {
	return dry.FileSetString(notifyFile, text)
}

// NotifyTextGet return and text
func NotifyTextGet() (string, error) {
	return dry.FileGetString(notifyFile)
}
