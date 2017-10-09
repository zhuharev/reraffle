package callback

import macaron "gopkg.in/macaron.v1"

// Help is static page
func Help(c *macaron.Context) {

	c.HTML(200, "callback/help")
}
