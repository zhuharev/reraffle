package synchronizer

import (
	"github.com/zhuharev/reraffle/models"
	"github.com/zhuharev/reraffle/modules/vk"
)

const (
	messagesLimit = 10000
)

// NewContext starts synchronizer background job
func NewContext() error {
	return nil
}

// Job algorythm:
// 1. select from db unreaded infos
// 2. fetch from vk dialogs
// 3. set answered
// 4. set readed
func Job(token string, publicID int) (err error) {

	infos, err := models.InfoSendedList(publicID, messagesLimit)
	if err != nil {
		return
	}

	//log.Println("return, no infos for ", publicID)

	// simple optimization: don't fetch vk's api, if no infos
	if len(infos) == 0 {
		return
	}

	dialogs, err := vk.GetDialogs(token, false)
	if err != nil {
		return
	}

	for _, lastMessage := range dialogs {
		for _, info := range infos {
			if info.UserID != lastMessage.Message.UserId ||
				info.Answered {
				continue
			}
			if info.MessageID != lastMessage.Message.Id {
				err = models.InfoSendedSetAnswered(publicID, info.MessageID)
				if err != nil {
					return
				}
			} else {
				if lastMessage.Message.ReadState == 1 && !info.Readed {
					err = models.InfoSendedSetReaded(publicID, info.MessageID)
					if err != nil {
						return
					}
				}
			}
		}
	}

	return
}
