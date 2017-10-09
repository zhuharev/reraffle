package leader

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/zhuharev/reraffle/models"
	"github.com/zhuharev/reraffle/modules/vk"
)

func Notify1C(t models.Req, crm *models.Crm) error {
	var (
		userID = t.UserID()
	)

	m, err := vk.GetUserNames([]int{userID})
	if err != nil {
		return err
	}

	if _, h := m[userID]; !h {
		return fmt.Errorf("Name not found for user %d", userID)
	}

	if len(m[userID]) != 2 {
		return fmt.Errorf("err name %d ", userID)
	}

	fName := m[userID][1]
	lName := m[userID][0]

	q := url.Values{}
	q.Set("fields[TITLE]", fName+" "+lName)
	q.Set("fields[NAME]", fName)
	q.Set("fields[LAST_NAME]", lName)

	q.Set("fields[STATUS_ID]", "NEW")
	q.Set("fields[OPENED]", "Y")
	q.Set("fields[ASSIGNED_BY_ID]", "1")
	q.Set("fields[IM][0][VALUE]", "id"+fmt.Sprint(userID))
	q.Set("fields[IM][0][VALUE_TYPE]", "VK")
	q.Set("fields[SOURCE_DESCRIPTION]", t.SourceDescription())

	q.Set("fields[COMMENTS]", t.Link())

	uri := fmt.Sprintf("%s/crm.lead.add.json", crm.Subdomain)
	log.Println("POST", uri)
	log.Println(q)
	resp, err := http.PostForm(uri, q)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}
