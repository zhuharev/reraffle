package notificator

import (
	"bytes"
	"log"
	"text/template"
	"time"

	"github.com/zhuharev/reraffle/models"
	"github.com/zhuharev/reraffle/modules/vk"
)

var (
	notifyAfter = 2 * time.Minute
)

// TODO: remove DRY

// Job run job
func Job(token string, publicID int) (err error) {
	infos, err := models.InfoSendedGetReaded(publicID)
	if err != nil {
		return
	}

	for _, info := range infos {
		switch info.MessageType {
		case models.Info:
			if time.Since(info.SendedAt) > notifyAfter {
				log.Println("Send notification")
				var (
					tplBody string
					tpl     *template.Template
					names   map[int][]string
					buf     = bytes.NewBuffer(nil)
					name    string
					msgID   int
				)
				tplBody, err = models.NotifyTextGet()
				if err != nil {
					return
				}

				tpl, err = template.New("ololo").Parse(tplBody)
				if err != nil {
					return
				}

				names, err = vk.GetUserNames([]int{info.UserID})
				if err != nil {
					return
				}

				if nms, has := names[info.UserID]; has && len(nms) == 2 {
					name = nms[1]
				}

				err = tpl.Execute(buf, map[string]interface{}{
					"name": name,
				})
				if err != nil {
					return
				}

				msgID, err = vk.SendMessage(token, info.UserID, buf.String())
				if err != nil {
					return
				}

				err = models.InfoSendedUpdate(publicID, info.UserID, msgID, info.RaffleID,
					models.Notification)
				if err != nil {
					return
				}

			}
		case models.Notification:
			if !info.EndDate.IsZero() && time.Until(info.EndDate.Add(24*time.Hour)) > 7*24*time.Hour {
				continue
			}
			log.Println("send end notification")
			var (
				tplBody string
				tpl     *template.Template
				names   map[int][]string
				buf     = bytes.NewBuffer(nil)
				name    string
				msgID   int
			)
			tplBody, err = models.EndTextGet()
			if err != nil {
				return
			}

			tpl, err = template.New("ololo").Parse(tplBody)
			if err != nil {
				return
			}

			names, err = vk.GetUserNames([]int{info.UserID})
			if err != nil {
				return
			}

			if nms, has := names[info.UserID]; has && len(nms) == 2 {
				name = nms[1]
			}

			err = tpl.Execute(buf, map[string]interface{}{
				"name": name,
			})
			if err != nil {
				return
			}

			msgID, err = vk.SendMessage(token, info.UserID, buf.String())
			if err != nil {
				return
			}

			err = models.InfoSendedUpdate(publicID, info.UserID, msgID, info.RaffleID,
				models.LastChance)
			if err != nil {
				return
			}

		}
	}

	return
}

func Send(token string, typ models.MessageType, userID int) error {
	return nil
}
