package models

import (
	"fmt"
	"strings"
	"time"
)

var (
	// ErrNotFound named error
	ErrNotFound = fmt.Errorf("not found")
)

// InfoSended like a log where sended messages
type InfoSended struct {
	PublicID  int    `xorm:"public_id"`
	UserID    int    `xorm:"user_id"`
	MessageID int    `xorm:"message_id"`
	RaffleID  string `xorm:"raffle_id"` // now date
	EndDate   time.Time
	SendedAt  time.Time

	MessageType MessageType

	Readed   bool
	ReadedAt time.Time

	Answered bool
}

// MessageType represend type of outgoing message
type MessageType int

const (
	// Info show last message is onfo
	Info = iota + 1
	// Notification show last message is 5 hours notification
	Notification
	// LastChance show last mesage is week before end notification
	LastChance
)

// InfoSendedHas return true if is in db
func InfoSendedHas(publicID, userID int, raffleID string) (bool, error) {
	is := new(InfoSended)
	return db.Where("public_id = ? and user_id = ? and raffle_id = ?", publicID, userID, raffleID).Get(is)
}

// InfoSendedGet return true if is in db
func InfoSendedGet(publicID, userID int, raffleID string) (*InfoSended, error) {
	is := new(InfoSended)
	has, err := db.Where("public_id = ? and user_id = ? and raffle_id = ?", publicID, userID, raffleID).Get(is)
	if !has {
		return nil, ErrNotFound
	}
	return is, err
}

func parsePeriod(strPeriod string) (start time.Time, end time.Time, err error) {
	arr := strings.Split(strPeriod, "-")
	if len(arr) != 2 {
		err = fmt.Errorf("Not a period")
		return
	}

	start, err = time.Parse("02.01", arr[0])
	if err != nil {
		return
	}
	end, err = time.Parse("02.01", arr[1])
	if err != nil {
		return
	}

	return
}

// InfoSendedNew create an info about sended message
func InfoSendedNew(publicID, userID, messageID int, raffleID string) error {
	is := &InfoSended{
		PublicID:    publicID,
		UserID:      userID,
		MessageID:   messageID,
		RaffleID:    raffleID,
		SendedAt:    time.Now(),
		MessageType: Info,
	}

	_, is.EndDate, _ = parsePeriod(raffleID)

	_, err := db.Insert(is)
	return err
}

// InfoSendedUpdate update and info sended
func InfoSendedUpdate(publicID, userID, messageID int, raffleID string, messageType MessageType) error {
	is := InfoSended{
		MessageID:   messageID,
		SendedAt:    time.Now(),
		MessageType: messageType,
	}
	_, err := db.Cols("readed", "answered", "readed_at", "message_id", "sended_at", "message_type").
		Where("public_id = ? and user_id = ? and raffle_id = ?", publicID, userID, raffleID).Update(&is)
	if err != nil {
		return err
	}

	return nil
}

// InfoSendedSetReaded update read status
func InfoSendedSetReaded(publicID, messageID int) error {
	is := InfoSended{
		Readed:   true,
		ReadedAt: time.Now(),
	}
	_, err := db.Where("message_id = ? and public_id = ?", messageID, publicID).Cols("readed", "readed_at").
		Update(&is)

	return err
}

// InfoSendedSetAnswered update read status
func InfoSendedSetAnswered(publicID, messageID int) error {
	is := InfoSended{
		Answered: true,
	}
	_, err := db.Where("message_id = ? and public_id = ?", messageID, publicID).Cols("answered").
		Update(&is)

	return err
}

// InfoSendedGetReaded return readed and not answered dialogs
func InfoSendedGetReaded(publickID int) (iss []InfoSended, err error) {
	err = db.Where("readed = ? and answered = ? and public_id = ?", true, false, publickID).Find(&iss)
	return
}

// InfoSendedGetUnreaded returns array of info,
// where infoType = Info or Notification
func InfoSendedGetUnreaded() (iss []InfoSended, err error) {
	err = db.Where("readed = ? and (message_type = ? or message_type = ? )", false, Info, Notification).Find(&iss)
	return
}

// InfoSendedList returns list infos
func InfoSendedList(publicID int, limitOffset ...int) (iss []InfoSended, err error) {
	var (
		offset = 0
		limit  = 20
	)

	if len(limitOffset) > 0 {
		limit = limitOffset[0]
	}
	if len(limitOffset) > 1 {
		offset = limitOffset[1]
	}

	err = db.Where("public_id = ?", publicID).OrderBy("date(sended_at) desc").Limit(limit, offset).Find(&iss)
	return
}
