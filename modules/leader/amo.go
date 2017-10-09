package leader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/Unknwon/com"
	"github.com/zhuharev/reraffle/models"
	"github.com/zhuharev/reraffle/modules/vk"
)

func NotifyAmo(t models.Req, crm *models.Crm) error {

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

	client, err := authAmo(crm.AmoLogin, crm.AmoKey, crm.Subdomain)
	if err != nil {
		return err
	}

	err = createFieldsIfNotExists(client, t, crm)
	if err != nil {
		log.Fatalln(err)
		log.Println(err)
		return err
	}

	lead := Lead{}
	lead.Name = fName + " " + lName
	lead.CustomFields = []Field{
		Field{
			ID: crm.VkFieldID,
			Values: []Value{
				Value{"https://vk.com/id" + fmt.Sprint(t.UserID())},
			},
		},
		Field{
			ID: crm.TextFieldID,
			Values: []Value{
				Value{t.Text()},
			},
		},
		Field{
			ID: crm.LinkFieldID,
			Values: []Value{
				Value{t.PureLink()},
			},
		},
	}

	req := Req{}
	req.Request.Leads.Add = append(req.Request.Leads.Add, lead)

	var r = map[string]interface{}{}

	err = com.HttpPostJSON(client, crm.Subdomain+"/private/api/v2/json/leads/set", req, &r)
	if err != nil {
		return err
	}
	for k, v := range r {
		log.Println(k, v)
	}

	return nil
}

func authAmo(login, hash, prefix string) (*http.Client, error) {
	var (
		cookieJar, _ = cookiejar.New(nil)

		client = &http.Client{
			Jar: cookieJar,
		}
	)

	var params = url.Values{}
	params.Set("USER_LOGIN", login)
	params.Set("USER_HASH", hash)

	resp, err := client.PostForm(prefix+"/private/api/auth.php?type=json", params)
	if err != nil {
		return client, err
	}
	defer resp.Body.Close()
	bts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return client, err
	}
	log.Printf("%s\n", bts)

	return client, nil
}

type Lead struct {
	Name         string  `json:"name"`
	CustomFields []Field `json:"custom_fields"`
}

type Field struct {
	ID int `json:"id"`

	Name     string `json:"name,omitempty"`
	Code     string `json:"code,omitempty"`
	Multiple string `json:"multiple,omitempty"`
	TypeID   string `json:"type_id,omitempty"`
	Type     int    `json:"type,omitempty"`

	ElementType int `json:"element_type,omitempty"`

	Origin string `json:"origin,omitempty"`

	Values []Value `json:"values,omitempty"`
}

type Value struct {
	Value string `json:"value"`
}

type AccountResponse struct {
	Response struct {
		Account struct {
			ID                   string `json:"id"`
			Name                 string `json:"name"`
			Subdomain            string `json:"subdomain"`
			Currency             string `json:"currency"`
			PaidFrom             bool   `json:"paid_from"`
			PaidTill             bool   `json:"paid_till"`
			Timezone             string `json:"timezone"`
			Language             string `json:"language"`
			NotificationsBaseURL string `json:"notifications_base_url"`
			NotificationsWsURL   string `json:"notifications_ws_url"`
			AmojoBaseURL         string `json:"amojo_base_url"`
			AmojoRights          struct {
				CanDirect      bool `json:"can_direct"`
				CanGroupCreate bool `json:"can_group_create"`
			} `json:"amojo_rights"`
			CurrentUser      int    `json:"current_user"`
			Version          int    `json:"version"`
			DatePattern      string `json:"date_pattern"`
			ShortDatePattern struct {
				Date     string `json:"date"`
				Time     string `json:"time"`
				DateTime string `json:"date_time"`
			} `json:"short_date_pattern"`
			DateFormat           string      `json:"date_format"`
			TimeFormat           string      `json:"time_format"`
			Country              string      `json:"country"`
			UnsortedOn           string      `json:"unsorted_on"`
			MobileFeatureVersion int         `json:"mobile_feature_version"`
			CustomersEnabled     interface{} `json:"customers_enabled"`
			Limits               struct {
				UsersCount       bool `json:"users_count"`
				ContactsCount    bool `json:"contacts_count"`
				ActiveDealsCount bool `json:"active_deals_count"`
			} `json:"limits"`
			Users []struct {
				ID                  string      `json:"id"`
				MailAdmin           string      `json:"mail_admin"`
				Name                string      `json:"name"`
				LastName            interface{} `json:"last_name"`
				Login               string      `json:"login"`
				PhotoURL            interface{} `json:"photo_url"`
				PhoneNumber         interface{} `json:"phone_number"`
				Language            string      `json:"language"`
				Active              bool        `json:"active"`
				IsAdmin             string      `json:"is_admin"`
				UnsortedAccess      string      `json:"unsorted_access"`
				CatalogsAccess      string      `json:"catalogs_access"`
				GroupID             int         `json:"group_id"`
				RightsLeadAdd       string      `json:"rights_lead_add"`
				RightsLeadView      string      `json:"rights_lead_view"`
				RightsLeadEdit      string      `json:"rights_lead_edit"`
				RightsLeadDelete    string      `json:"rights_lead_delete"`
				RightsLeadExport    string      `json:"rights_lead_export"`
				RightsContactAdd    string      `json:"rights_contact_add"`
				RightsContactView   string      `json:"rights_contact_view"`
				RightsContactEdit   string      `json:"rights_contact_edit"`
				RightsContactDelete string      `json:"rights_contact_delete"`
				RightsContactExport string      `json:"rights_contact_export"`
				RightsCompanyAdd    string      `json:"rights_company_add"`
				RightsCompanyView   string      `json:"rights_company_view"`
				RightsCompanyEdit   string      `json:"rights_company_edit"`
				RightsCompanyDelete string      `json:"rights_company_delete"`
				RightsCompanyExport string      `json:"rights_company_export"`
				FreeUser            bool        `json:"free_user"`
			} `json:"users"`
			Groups        []interface{} `json:"groups"`
			LeadsStatuses []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				PipelineID int    `json:"pipeline_id"`
				Sort       string `json:"sort"`
				Color      string `json:"color"`
				Editable   string `json:"editable"`
			} `json:"leads_statuses"`
			CustomFields struct {
				Contacts []struct {
					ID       string            `json:"id"`
					Name     string            `json:"name"`
					Code     string            `json:"code"`
					Multiple string            `json:"multiple"`
					TypeID   string            `json:"type_id"`
					Disabled string            `json:"disabled"`
					Sort     int               `json:"sort"`
					Enums    map[string]string `json:"enums,omitempty"`
				} `json:"contacts"`
				Leads []struct {
					ID       string            `json:"id"`
					Name     string            `json:"name"`
					Code     string            `json:"code"`
					Multiple string            `json:"multiple"`
					TypeID   string            `json:"type_id"`
					Disabled string            `json:"disabled"`
					Sort     int               `json:"sort"`
					Enums    map[string]string `json:"enums,omitempty"`
				} `json:"leads"`
				Companies []struct {
					ID       string            `json:"id"`
					Name     string            `json:"name"`
					Code     string            `json:"code"`
					Multiple string            `json:"multiple"`
					TypeID   string            `json:"type_id"`
					Disabled string            `json:"disabled"`
					Sort     int               `json:"sort"`
					Enums    map[string]string `json:"enums,omitempty"`
				} `json:"companies"`
				Customers []interface{} `json:"customers"`
			} `json:"custom_fields"`
			NoteTypes []struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				Code     string `json:"code"`
				Editable string `json:"editable"`
			} `json:"note_types"`
			TaskTypes []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Code string `json:"code"`
			} `json:"task_types"`
			Pipelines struct {
				Num720733 struct {
					ID       int    `json:"id"`
					Value    int    `json:"value"`
					Label    string `json:"label"`
					Name     string `json:"name"`
					Sort     int    `json:"sort"`
					IsMain   bool   `json:"is_main"`
					Statuses struct {
						Num142 struct {
							ID         int    `json:"id"`
							Name       string `json:"name"`
							Color      string `json:"color"`
							Sort       int    `json:"sort"`
							Editable   string `json:"editable"`
							PipelineID int    `json:"pipeline_id"`
						} `json:"142"`
						Num143 struct {
							ID         int    `json:"id"`
							Name       string `json:"name"`
							Color      string `json:"color"`
							Sort       int    `json:"sort"`
							Editable   string `json:"editable"`
							PipelineID int    `json:"pipeline_id"`
						} `json:"143"`
						Num15918913 struct {
							ID         int    `json:"id"`
							Name       string `json:"name"`
							PipelineID int    `json:"pipeline_id"`
							Sort       int    `json:"sort"`
							Color      string `json:"color"`
							Editable   string `json:"editable"`
						} `json:"15918913"`
						Num15918916 struct {
							ID         int    `json:"id"`
							Name       string `json:"name"`
							PipelineID int    `json:"pipeline_id"`
							Sort       int    `json:"sort"`
							Color      string `json:"color"`
							Editable   string `json:"editable"`
						} `json:"15918916"`
						Num15918919 struct {
							ID         int    `json:"id"`
							Name       string `json:"name"`
							PipelineID int    `json:"pipeline_id"`
							Sort       int    `json:"sort"`
							Color      string `json:"color"`
							Editable   string `json:"editable"`
						} `json:"15918919"`
						Num15918922 struct {
							ID         int    `json:"id"`
							Name       string `json:"name"`
							PipelineID int    `json:"pipeline_id"`
							Sort       int    `json:"sort"`
							Color      string `json:"color"`
							Editable   string `json:"editable"`
						} `json:"15918922"`
					} `json:"statuses"`
					Leads int `json:"leads"`
				} `json:"720733"`
			} `json:"pipelines"`
			Timezoneoffset string `json:"timezoneoffset"`
		} `json:"account"`
		ServerTime int `json:"server_time"`
	} `json:"response"`
}

type Req struct {
	Request struct {
		Leads struct {
			Add []Lead `json:"add"`
		} `json:"leads"`
	} `json:"request"`
}

const (
	VkFieldName   = "Vk"
	TextFieldName = "Текст комментария"
	LinkFieldName = "Ссылка на пост"
)

func createFieldsIfNotExists(client *http.Client, t models.Req, crm *models.Crm) (err error) {

	if crm.VkFieldID != 0 && crm.TextFieldID != 0 && crm.LinkFieldID != 0 {
		return nil
	}

	var r AccountResponse
	err = com.HttpGetJSON(client, crm.Subdomain+"/private/api/v2/json/accounts/current", &r)
	if err != nil {
		return
	}

	var (
		hasVkField          = false
		hasCommentTextField = false
		hasLinkField        = false
	)

	for _, v := range r.Response.Account.CustomFields.Leads {
		id := com.StrTo(v.ID).MustInt()
		if v.Name == VkFieldName {
			hasVkField = true
			models.CrmSetVkFieldID(crm.WebHookKey, id)
		}
		if v.Name == TextFieldName {
			hasCommentTextField = true
			models.CrmSetTextFieldID(crm.WebHookKey, id)
		}
		if v.Name == LinkFieldName {
			hasLinkField = true
			log.Println("link field existts, id = ", id)
			err := models.CrmSetLinkFieldID(crm.WebHookKey, id)
			if err != nil {
				log.Println(err)
			}
		}
	}

	if !hasVkField {
		vkID, err := createField(client, crm, VkFieldName, "url")
		if err != nil {
			log.Printf("Error creating vk field: %s", err)
			return err
		}
		err = models.CrmSetVkFieldID(crm.WebHookKey, vkID)
		if err != nil {
			log.Printf("Error set vk field: %s", err)
			return err
		}
		crm.VkFieldID = vkID
	}

	if !hasCommentTextField {
		textID, err := createField(client, crm, TextFieldName, "text")
		if err != nil {
			log.Printf("Error creating text field: %s", err)
			return err
		}
		err = models.CrmSetTextFieldID(crm.WebHookKey, textID)
		if err != nil {
			log.Printf("Error set text field: %s", err)
			return err
		}
		crm.TextFieldID = textID
	}

	if !hasLinkField {
		log.Println("Создаю поле со сслыкой")
		textID, err := createField(client, crm, LinkFieldName, "url")
		if err != nil {
			log.Printf("Error creating text field: %s", err)
			return err
		}
		err = models.CrmSetLinkFieldID(crm.WebHookKey, textID)
		if err != nil {
			log.Printf("Error set text field: %s", err)
			return err
		}
		crm.LinkFieldID = textID
	}

	return nil
}

func createField(client *http.Client, crm *models.Crm, name string, typ string) (int, error) {

	var (
		endpoint = "/private/api/v2/json/fields/set"
	)
	var f Field
	switch typ {
	case "url":
		f = Field{
			Name: name,
			Type: 7,
			//	Code:        "URL",
			ElementType: 2,

			Origin: "vk_custom_field",
		}
		if name == LinkFieldName {
			f.Origin = "vk_link_field"
		}
	case "text":
		f = Field{
			Name: name,
			Type: 1,
			//		Code:        "TEXT",
			ElementType: 2,

			Origin: "text_custom_field",
		}
	}

	var req = FieldsRequest{}
	req.Request.Fields.Add = []Field{f}

	//var r = map[string]interface{}{}

	bts, err := json.Marshal(req)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	log.Println("REQ")
	log.Println(string(bts))

	//var r = map[string]interface{}{}
	uri := crm.Subdomain + endpoint
	log.Println(uri)
	//log.Println(client.Jar.Cookies())
	rsp, err := com.HttpPost(client, uri, http.Header{}, bts)
	if err != nil {
		log.Println(err)
	}
	defer rsp.Close()

	data, err := ioutil.ReadAll(rsp)
	if err != nil {
		log.Println(err)
	}

	log.Println("RESP")
	log.Println(string(data))

	var r SetFieldsResponse

	err = json.Unmarshal(data, &r)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	if len(r.Response.Fields.Add) == 0 {
		return 0, fmt.Errorf("error create field")
	}
	return r.Response.Fields.Add[0].ID, nil
}

type FieldsRequest struct {
	Request struct {
		Fields struct {
			Add []Field `json:"add"`
		} `json:"fields"`
	} `json:"request"`
}

type SetFieldsResponse struct {
	Response struct {
		Fields struct {
			Add []Field `json:"add"`
		} `json:"fields"`
	} `json:"response"`
}
