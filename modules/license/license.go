package license

import (
	"net/http"

	"github.com/Unknwon/com"
)

type Response struct {
	Good bool `json:"response"`
}

func Check() bool {
	var r Response
	err := com.HttpGetJSON(http.DefaultClient, "https://zhuharev.ru/le.json", &r)
	if err != nil {
		return false
	}
	return r.Good
}
