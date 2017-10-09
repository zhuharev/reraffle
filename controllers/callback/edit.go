package callback

import (
	"strings"

	"github.com/zhuharev/reraffle/models"
	macaron "gopkg.in/macaron.v1"
)

// Edit update confiration string of crm
func Edit(c *macaron.Context) {
	key := c.Params(":key")

	crm := new(models.Crm)
	crm.WebHookKey = key
	crm.ConfirmationString = c.Query("confirmation")
	crm.Subdomain = c.Query("subdomain")
	crm.Subdomain = strings.TrimSuffix(crm.Subdomain, "/profile/")

	crm.AmoKey = c.Query("amo_key")
	crm.AmoLogin = c.Query("amo_login")

	crm.SheetID = c.Query("sheet_id")
	crm.SheetName = c.Query("sheet_name")

	if err := models.CrmUpdate(crm); err != nil {
		c.Error(200, err.Error())
		return
	}

	c.Redirect("/edit/" + key)
}

// Delete update confiration string of crm
func Delete(c *macaron.Context) {
	key := c.Params(":key")

	if err := models.CrmDelete(key); err != nil {
		c.Error(200, err.Error())
		return
	}

	c.Redirect("/")
}
