package models

import (
	"encoding/json"
	"fmt"
)

// Crm name cool
type Crm struct {
	WebHookKey string `xorm:"pk"`
	Name       string
	Type       CrmType

	AmoLogin string
	AmoKey   string

	// 1c
	Subdomain string

	VkFieldID   int `xorm:"vk_field_id"`
	TextFieldID int `xorm:"text_field_id"`
	LinkFieldID int `xorm:"link_field_id"`

	SheetID   string `xorm:"sheet_id"`
	SheetName string `xorm:"sheet_name"`

	ConfirmationString string
}

type CrmType int

const (
	Crm1C CrmType = iota + 1
	CrmAmo
)

// WebHookLog log
type WebHookLog struct {
	ID  int64  `xorm:"pk autoincr 'id'"`
	Key string `xorm:"index"`

	Data []byte

	Req Req `xorm:"-"`
}

func (wl WebHookLog) String() string {
	return string(wl.Data)
}

type EventType string

const (
	Confirmation  EventType = "confirmation"
	Message       EventType = "message_new"
	Comment       EventType = "wall_reply_new"
	CommentPhoto  EventType = "photo_comment_new"
	CommentVideo  EventType = "video_comment_new"
	CommentBoard  EventType = "board_post_new"
	CommentMarket EventType = "market_comment_new"

	WallPost EventType = "wall_post_new"
)

func (et EventType) String() string {
	switch et {
	case Message:
		return "сообщение"
	case Comment:
		return "Комментарий к записи на стене"
	default:
		return "Комментарий"
	}
}

type Req struct {
	Type    EventType `json:"type"`
	GroupID int       `json:"group_id"`
	Object  Object    `json:"object"`
}

func (r Req) PureLink() string {
	switch r.Type {
	case Message:
		return ""
	case Comment:
		return fmt.Sprintf(`https://vk.com/public%[1]d?w=wall-%[1]d_%[2]d_r%[3]d`,
			r.GroupID, r.Object.PostID, r.Object.ID)
	case CommentPhoto:
		return fmt.Sprintf(`https://vk.com/photo%[1]d_%[2]d`, r.Object.PhotoOwnerID, r.Object.PhotoID)
	case CommentVideo:
		return fmt.Sprintf(`https://vk.com/video%[1]d_%[2]d`, r.Object.VideoOwnerID, r.Object.VideoID)
	case CommentBoard:
		return fmt.Sprintf(`https://vk.com/topic-%[1]d_%[2]d?post=%[3]d`, r.Object.TopicOwnerID, r.Object.TopicID, r.Object.ID)
	case CommentMarket:
		return fmt.Sprintf(`https://vk.com/public%[1]d?w=product-%[1]d_%[2]d`,
			r.GroupID, r.Object.ItemID)
	}
	return ""
}

func (r Req) Link() string {
	return fmt.Sprintf(`<a href="%s">ссылка</a>`, r.PureLink())
}

func (r Req) Text() string {
	switch r.Type {
	case Message:
		return r.Object.Body
	case Comment:
		return r.Object.Text
	}
	return ""
}

func (r Req) UserID() int {
	switch r.Type {
	case Message:
		return r.Object.UserID
	case Comment:
		return r.Object.FromID
	case CommentPhoto:
		return r.Object.FromID
	case CommentMarket:
		return r.Object.FromID
	case CommentVideo:
		return r.Object.FromID
	}
	return 0
}

func (r Req) UserLink() string {
	return fmt.Sprintf("https://vk.com/id%d", r.UserID())
}

func (r Req) SourceDescription() string {
	switch r.Type {
	case Message:
		return "Сообщение в группу"
	case Comment:
		return "Комментарий к записи"
	case CommentPhoto:
		return "Комментарий к фото"
	case CommentVideo:
		return "Комментарий к видео"
	case CommentBoard:
		return "Комментарий в обсуждении"
	case CommentMarket:
		return "Комментарий к товару"
	}
	return ""
}

// Object unmarshal
type Object struct {
	ID      int    `json:"id"`
	FromID  int    `json:"from_id"`
	UserID  int    `json:"user_id"`
	PostID  int    `json:"post_id"`
	OwnerID int    `json:"owner_id"`
	Body    string `json:"body"`

	Text string `json:"text"`

	PhotoID      int `json:"photo_id"`
	PhotoOwnerID int `json:"photo_owner_id"`

	VideoID      int `json:"video_id"`
	VideoOwnerID int `json:"video_owner_id"`

	TopicID      int `json:"topic_id"`
	TopicOwnerID int `json:"topic_owner_id"`

	ItemID        int `json:"item_id"`
	MarketOwnerID int `json:"market_owner_id"`
}

// CrmNew create new crm in database
func CrmNew(crm *Crm) error {
	_, err := db.InsertOne(crm)
	return err
}

// CrmGet return crm by webHookKey
func CrmGet(key string) (*Crm, error) {
	crm := new(Crm)
	has, err := db.Where("web_hook_key = ?", key).Get(crm)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrNotFound
	}
	return crm, nil
}

// CrmUpdate update confirmation_string of crm
func CrmUpdate(crm *Crm) error {
	_, err := db.Where("web_hook_key = ?", crm.WebHookKey).Cols("confirmation_string",
		"subdomain", "amo_login", "amo_key", "sheet_id", "sheet_name").Update(crm)
	return err
}

// CrmDelete update confirmation_string of crm
func CrmDelete(key string) error {
	_, err := db.Where("web_hook_key = ?", key).Delete(new(Crm))
	return err
}

// CrmList return all crms
func CrmList() (res []Crm, err error) {
	err = db.Find(&res)
	return
}

func CrmSetVkFieldID(id string, fieldID int) error {
	crm := new(Crm)
	crm.VkFieldID = fieldID
	_, err := db.Where("web_hook_key = ?", id).Update(crm)
	if err != nil {
		return err
	}
	return nil
}

func CrmSetTextFieldID(id string, fieldID int) error {
	crm := new(Crm)
	crm.TextFieldID = fieldID
	_, err := db.Where("web_hook_key = ?", id).Update(crm)
	if err != nil {
		return err
	}
	return nil
}

func CrmSetLinkFieldID(id string, fieldID int) error {
	crm := new(Crm)
	crm.LinkFieldID = fieldID
	_, err := db.Where("web_hook_key = ?", id).Update(crm)
	if err != nil {
		return err
	}
	return nil
}

// CrmAddLog append log data
func CrmAddLog(key string, data []byte) error {
	wl := new(WebHookLog)
	wl.Data = data
	wl.Key = key
	_, err := db.InsertOne(wl)
	return err
}

// CrmGetLog return last 20 log records
func CrmGetLog(key string) (wl []WebHookLog, err error) {
	err = db.Where("key = ?", key).OrderBy("id desc").Limit(20).Find(&wl)
	for i, v := range wl {
		json.Unmarshal(v.Data, &wl[i].Req)
	}
	return
}
