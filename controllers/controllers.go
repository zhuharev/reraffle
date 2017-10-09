package controllers

import (
	"github.com/zhuharev/reraffle/models"
	macaron "gopkg.in/macaron.v1"
)

// UpdateEndText update end notify text
func UpdateEndText(c *macaron.Context) {

	text := c.Query("text")

	err := models.EndTextUpdate(text)
	if err != nil {
		c.Error(200, err.Error())
		return
	}

	c.Redirect("/settings")
}

// UpdateNotifyText update end notify text
func UpdateNotifyText(c *macaron.Context) {

	text := c.Query("text")

	err := models.NotifyTextUpdate(text)
	if err != nil {
		c.Error(200, err.Error())
		return
	}

	c.Redirect("/settings")
}

func UpdateNotAWinnerText(c *macaron.Context) {
	text := c.Query("text")

	err := models.NotAWinnerTextUpdate(text)
	if err != nil {
		c.Error(200, err.Error())
		return
	}

	c.Redirect("/settings")
}
