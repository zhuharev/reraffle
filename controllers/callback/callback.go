package callback

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"

	dry "github.com/ungerik/go-dry"
	"github.com/zhuharev/reraffle/models"
	"github.com/zhuharev/reraffle/modules/leader"
	"github.com/zhuharev/reraffle/modules/sheets"

	macaron "gopkg.in/macaron.v1"
)

var sended = map[int][]int{}

func init() {
	bts, err := dry.FileGetBytes("cb_crm_sended")
	if err != nil {
		return
	}
	err = json.Unmarshal(bts, &sended)
	if err != nil {
		panic(err)
	}
}

var (
	linkFmt = `<a href="https://vk.com/%s">ссылка</a>`
)

func Handler(m *macaron.Context) string {
	var (
		key = m.Params(":key")
	)

	bts, err := ioutil.ReadAll(m.Req.Request.Body)
	if err != nil {
		log.Println("error reading request body", err)
		//m.Error(200, err.Error())
		return "ok"
	}

	err = models.CrmAddLog(key, bts)
	if err != nil {
		log.Println("crm add error", err)
		//	m.Error(200, err.Error())
		return "ok"
	}

	var (
		t   models.Req
		uID int
	)
	err = json.Unmarshal(bts, &t)
	if err != nil {
		if err != io.EOF {
			log.Println("EOF, return", err)
			return "ok"
		}
	}

	log.Println(t)

	crm, err := models.CrmGet(key)
	if err != nil {
		log.Println("Err crm get, return ", err)
		return "ok"
	}

	uID = t.UserID()

	switch t.Type {
	case models.Confirmation:
		return crm.ConfirmationString
	}

	//if h, _ := has(t.GroupID, uID); h {
	//return "ok"
	//}

	log.Println("start notify")
	err = notify(t, crm)
	if err != nil {
		log.Println("Err notify", err)
		return "ok"
	}
	log.Println("end notify")
	err = set(t.GroupID, uID)
	if err != nil {
		log.Println("Err set", err)
	}

	log.Println("ALL DONE, RETUrN")
	return "ok"
}

func notify(t models.Req, crm *models.Crm) error {
	switch crm.Type {
	case models.Crm1C:
		err := sheets.Append(crm.SheetID, crm.SheetID, [][]interface{}{
			[]interface{}{
				t.Type, t.PureLink(), t.Text(), t.UserID(),
			},
		})
		if err != nil {
			log.Println(err)
		}
		return leader.Notify1C(t, crm)
	case models.CrmAmo:
		err := sheets.Append(crm.SheetID, crm.SheetName, [][]interface{}{
			[]interface{}{
				t.Type.String(), t.PureLink(), t.Text(), t.UserLink(),
			},
		})
		if err != nil {
			log.Println(err)
		}
		return leader.NotifyAmo(t, crm)
	}
	return nil
}

func has(groupID, userID int) (bool, error) {
	if sended == nil {
		return false, nil
	}
	for _, v := range sended[groupID] {
		if userID == v {
			return true, nil
		}
	}
	return false, nil
}

func set(groupID, userID int) error {
	arr := sended[groupID]
	arr = append(arr, userID)
	sended[groupID] = arr

	return dry.FileSetJSON("cb_crm_sended", sended)
}
