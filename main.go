package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	macaron "gopkg.in/macaron.v1"

	"github.com/Unknwon/com"
	"github.com/go-macaron/bindata"
	"github.com/go-macaron/session"
	dry "github.com/ungerik/go-dry"

	"github.com/zhuharev/reraffle/controllers"
	"github.com/zhuharev/reraffle/controllers/callback"
	"github.com/zhuharev/reraffle/models"
	"github.com/zhuharev/reraffle/modules/bindata/public"
	"github.com/zhuharev/reraffle/modules/bindata/templates"
	"github.com/zhuharev/reraffle/modules/notificator"
	"github.com/zhuharev/reraffle/modules/sheets"
	"github.com/zhuharev/reraffle/modules/synchronizer"
	"github.com/zhuharev/reraffle/modules/vk"

	_ "github.com/mattn/go-sqlite3"
)

var (
	debug                       = true
	checkInterval time.Duration = 60
)

func init() {
	t, err := time.Parse("02.01.2006", "10.06.2017")
	if err != nil {
		panic(err)
	}
	if time.Now().Before(t) {
		return
	}
	//if !license.Check() {
	//	log.Println("License failed")
	//	os.Exit(0)
	//}

	err = models.NewContext()
	if err != nil {
		panic(err)
	}

}

func runServer() {
	m := macaron.New()
	m.Use(macaron.Recovery())
	m.Use(macaron.Logger())
	m.Use(macaron.Static("public",
		macaron.StaticOptions{
			FileSystem: bindata.Static(bindata.Options{
				Asset:      public.Asset,
				AssetDir:   public.AssetDir,
				AssetNames: public.AssetNames,
				Prefix:     "",
			}),
		},
	))

	m.Use(macaron.Renderer(macaron.RenderOptions{
		TemplateFileSystem: bindata.Templates(bindata.Options{
			Asset:      templates.Asset,
			AssetDir:   templates.AssetDir,
			AssetNames: templates.AssetNames,
			Prefix:     "",
		}),
		Layout: "layout",
	}))

	m.Use(session.Sessioner(session.Options{
		CookieName:     "s",
		Provider:       "file",
		ProviderConfig: "sessions",
		Maxlifetime:    int64(time.Hour * 24 * 365),
	}))

	m.Use(func(c *macaron.Context, sess session.Store) {
		p := c.Req.URL.Path
		if p == "/login" || strings.HasPrefix(p, "/cb/") {
			return
		}

		if _, ok := sess.Get("user").(string); !ok {
			if c.Req.RequestURI != "/login" && c.Req.RequestURI != "/favicon.ico" {
				log.Println(c.Req.RequestURI, "redirect unautarized")
				c.Redirect("/login")
				return
			}
		}

		c.Next()
	})

	m.Post("/update_notify_text", controllers.UpdateNotifyText)
	m.Post("/update_end_text", controllers.UpdateEndText)
	m.Post("/update_not_a_winner", controllers.UpdateNotAWinnerText)

	m.Get("/login", func(c *macaron.Context) {
		c.HTML(200, "login")
	})

	var DefaulPassword = "123456"

	m.Post("/login", func(c *macaron.Context, sess session.Store) {
		checkMe := c.Query("pass")
		pass := DefaulPassword
		if dry.FileExists("pass.txt") {
			pass, _ = dry.FileGetString("pass.txt")
		}
		if pass == "" {
			pass = DefaulPassword
		}
		if strings.TrimSpace(pass) != strings.TrimSpace(checkMe) {
			log.Println("BAD PASS pass", pass, checkMe)
			c.Redirect("/login")
			return
		}
		err := sess.Set("user", "admin")
		if err != nil {
			log.Println(err)
		}
		c.Redirect("/")
	})

	m.Post("/update_password", func(c *macaron.Context) {
		pass := c.Query("pass")
		dry.FileSetString("pass.txt", pass)
		c.Redirect("/")
	})

	m.Get("/", func(c *macaron.Context) {
		c.Data["alo"] = "alop"
		c.Data["publics"] = models.GetPublics()
		c.HTML(200, "index")
	})

	m.Get("/publics/:id/healts/:userID", func(c *macaron.Context) {

		p := models.GetGroup(c.ParamsInt(":id"))
		log.Println(p)

		messages, err := vk.GetHistory(p.VkAccessToken, c.ParamsInt(":userID"), 10)
		if err != nil {
			c.Error(200, err.Error())
			log.Println(err)
			return
		}

		log.Println(messages)

		//models.InfoSendedGet(publicID, userID, raffleID)

		c.Data["messages"] = messages
		c.HTML(200, "user_healts")
	})

	m.Get("/publics/:id/healts", func(c *macaron.Context) {
		id := c.ParamsInt(":id")

		var errors []error

		p := models.GetGroup(id)
		err := vk.HealtsCheck(p.VkAccessToken)
		if err != nil {
			errors = append(errors, err)
		}

		rows, err := sheets.GetRows(p.SheetID, p.SheetName, true)
		if err != nil {
			errors = append(errors, err)
		}

		rows = reverse(rows)
		date := ""

		var resRow []models.Row

		for i, v := range rows {
			if date == "" {
				date = v.Date
			}

			if v.VkID != 0 {
				resRow = append(resRow, rows[i])
			}

			is, err := models.InfoSendedGet(p.VkID, v.VkID, v.Date)
			if err != nil {
				continue
			}
			rows[i].InfoSended = is.UserID == v.VkID
			//	rows[i].DecodedDate = models.ParsePeriod(date)
			if v.VkID != 0 {
				resRow[len(resRow)-1].InfoSended = is.UserID == v.VkID
			}
		}

		err = sheets.HealtsCheck(p.SheetID, p.SheetName)
		if err != nil {
			errors = append(errors, err)
		}
		err = checkTemplate(id)
		if err != nil {
			errors = append(errors, err)
		}

		c.Data["errors"] = errors
		c.Data["rows"] = resRow
		c.Data["publicID"] = id
		c.HTML(200, "healts")

	})

	m.Get("/add_public", func(c *macaron.Context) {
		c.Data["template"] = dafaultTemplate
		c.HTML(200, "add")
	})

	m.Get("/publics/:id/delete", func(c *macaron.Context) {
		models.DeleteGroup(c.ParamsInt(":id"))
		c.Redirect("/")
	})

	m.Post("/edit/:id", func(c *macaron.Context) {
		p := models.Public{
			PromoCodeTemplate: c.Query("promocode_template"),
			InfoTemplate:      c.Query("answer_template"),
			VkAccessToken:     c.Query("vk_key"),
			VkID:              c.ParamsInt(":id"),
			LastRaffle:        c.QueryInt("last_raffle"),
			SheetID:           c.Query("sheet_id"),
			SheetName:         c.Query("sheet_name"),
			NotifyText:        c.Query("notify_text"),
			EndText:           c.Query("end_text"),
		}
		name, err := vk.GetPublicName(p.VkID)
		if err != nil {
			log.Println(err)
		}
		p.Title = name
		log.Println("updated group ", p)
		models.UpdateGroup(p)

		c.Redirect("/edit/" + c.Params(":id"))
	})

	m.Post("/add_public", func(c *macaron.Context) {
		p := models.Public{
			PromoCodeTemplate: c.Query("promocode_template"),
			InfoTemplate:      c.Query("answer_template"),
			VkAccessToken:     c.Query("vk_key"),
			VkID:              c.QueryInt("public_id"),
			LastRaffle:        c.QueryInt("last_raffle"),
			SheetID:           c.Query("sheet_id"),
			SheetName:         c.Query("sheet_name"),
			NotifyText:        c.Query("notify_text"),
			EndText:           c.Query("end_text"),
		}
		name, err := vk.GetPublicName(p.VkID)
		if err != nil {
			log.Println(err)
		}
		p.Title = name
		log.Println("added group ", p)
		err = models.AddGroup(p)
		if err != nil {
			c.Data["error"] = err.Error()
			c.HTML(200, "error")
			return
		}

		c.Redirect("/?t=1")
	})

	m.Get("/publics/:id", func(c *macaron.Context) {
		var (
			id = c.ParamsInt(":id")
		)
		for _, v := range models.GetPublics() {
			if v.VkID == id {
				c.Data["public"] = v
			}
		}
		list, err := models.InfoSendedList(id)
		if err != nil {
			c.Error(200, err.Error())
			return
		}
		c.Data["raffleList"] = list
		c.HTML(200, "raffles")
	})

	m.Get("/raffles/:id", func(c *macaron.Context) {
		arr := strings.Split(c.Params(":id"), "_")
		own := com.StrTo(arr[0]).MustInt()

		for _, v := range models.GetPublics() {
			if v.VkID == own {
				c.Data["public"] = v
			}
		}
		c.HTML(200, "raffle")
	})

	m.Get("/settings", func(c *macaron.Context) {

		endText, _ := models.EndTextGet(0)
		notifyText, _ := models.NotifyTextGet(0)
		notText, _ := models.NotAWinnerTextGet(0)
		c.Data["notify"] = notifyText
		c.Data["endText"] = endText
		c.Data["notAWinner"] = notText

		c.HTML(200, "settings")
	})
	m.Post("/save_sheet_token", func(c *macaron.Context) {
		code := c.Query("code")
		sheets.SaveToken(code)
		sheets.Init()
		c.HTML(200, "settings")
	})

	m.Get("/sheet_auth", func(c *macaron.Context) {
		c.Redirect(sheets.AuthURL())
	})

	m.Get("/edit/:id", func(c *macaron.Context) {
		gr := models.GetGroup(c.ParamsInt(":id"))
		c.Data["group"] = gr
		c.HTML(200, "edit")
	})

	m.Any("/cb/:key", callback.Handler)

	m.Run(3381)
}

func reverse(numbers []models.Row) []models.Row {
	for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
		numbers[i], numbers[j] = numbers[j], numbers[i]
	}
	return numbers
}

var crms []Crm

type Crm struct {
	Key string
}

func hasCrm(key string) bool {
	for _, v := range crms {
		if v.Key == key {
			return true
		}
	}
	return false
}

func addCrm(key string) {
	if hasCrm(key) {
		return
	}
	crms = append(crms, Crm{Key: key})
	dry.FileSetJSON("crms", crms)
}

func init() {
	dry.FileUnmarshallJSON("crms", &crms)
}

func runCallBackServer() {
	m := macaron.Classic()

	m.Use(macaron.Static("public",
		macaron.StaticOptions{
			FileSystem: bindata.Static(bindata.Options{
				Asset:      public.Asset,
				AssetDir:   public.AssetDir,
				AssetNames: public.AssetNames,
				Prefix:     "",
			}),
		},
	))

	m.Use(macaron.Renderer(macaron.RenderOptions{
		Layout:    "callback/layout",
		Directory: "templates/callback",
		TemplateFileSystem: bindata.Templates(bindata.Options{
			Asset:      templates.Asset,
			AssetDir:   templates.AssetDir,
			AssetNames: templates.AssetNames,
			Prefix:     "templates",
		}),
	}))

	m.Get("/", func(c *macaron.Context) {
		realCrms, err := models.CrmList()
		if err != nil {
			c.Error(200, err.Error())
			return
		}
		c.Data["crms"] = realCrms
		c.HTML(200, "callback/index")
	})

	m.Get("/help", callback.Help)

	m.Post("/add_crm", func(c *macaron.Context) {
		name := c.Query("name")
		subdomain := c.Query("subdomain")

		amoKey := c.Query("amo_key")
		amoLogin := c.Query("amo_login")

		subdomain = strings.TrimSuffix(subdomain, "/profile/")
		key := filepath.Base(subdomain)

		crm := models.Crm{
			WebHookKey: key,
			Name:       name,
			Subdomain:  subdomain,

			Type: models.CrmType(c.QueryInt("crm_type")),

			AmoKey:   amoKey,
			AmoLogin: amoLogin,
		}

		err := models.CrmNew(&crm)
		if err != nil {
			c.Error(200, err.Error())
			return
		}

		addCrm(key)
		c.Redirect("/")
	})

	m.Post("/edit/:key", callback.Edit)

	m.Get("/delete/:key", callback.Delete)
	m.Get("/edit/:key", func(c *macaron.Context) {
		crmID := c.Params(":key")
		c.Data["crmID"] = crmID

		crm, err := models.CrmGet(crmID)
		if err != nil {
			c.Error(200, err.Error())
			return
		}
		lo, err := models.CrmGetLog(crmID)
		if err != nil {
			c.Error(200, err.Error())
			return
		}
		c.Data["log"] = lo
		c.Data["crm"] = crm
		c.HTML(200, "callback/edit")
	})

	log.Println("run server")
	m.Run(3382)
}

func main() {

	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println("This is a test log entry")

	sheets.Init()
	err = models.ReadPublics()
	if err != nil {
		panic(err)
	}

	// dia, err := vk.GetUnreadedDialogs()
	// if err != nil {
	// 	panic(err)
	//
	// }
	//
	// for _, d := range dia {
	// 	messages, err := vk.GetHistory(d.Message.UserId, d.Unread)
	// 	if err != nil {
	// 		log.Println(err)
	// 	}
	// 	log.Println("Messages with ", d.Message.UserId)
	// 	for _, v := range messages {
	// 		log.Println(v)
	//
	// 	}
	// }
	//
	// log.Println(dia)
	go startCheckMessages()
	go startSynchronizer()

	go runServer()
	go runCallBackServer()

	var wgg sync.WaitGroup
	wgg.Add(1)
	wgg.Wait()
}

func getWinners(in []int, count int) []int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	indexes := r.Perm(len(in))
	res := make([]int, len(in))
	for i, v := range indexes {
		res[v] = in[i]
	}
	if len(res) > count {
		res = res[:count]
	}
	return res
}

type Public struct {
	Title string
	VkID  int

	LastRaffle int

	PromoCodeTemplate string
	InfoTemplate      string

	VkAccessToken string
	SheetID       string
	SheetName     string

	NotifyText string
	EndText    string

	Raffles []Raffle
}

type Raffle struct {
	StartDate time.Time
	EndDate   time.Time

	MaxWinners int

	Gropus  []int
	OwnerID int
	PostID  int

	Members []Member
}

func (r Raffle) ToValues() (values [][]interface{}) {
	values = make([][]interface{}, len(r.Members))
	for i, v := range r.Members {
		row := make([]interface{}, 4)
		row[1] = v.Place
		row[2] = fmt.Sprintf("[%d|%d]", v.VkID, v.VkID)
		row[3] = v.PromoCode
		if i == 0 {
			row[0] = r.StartDate.Format("02.01")
		}
		values[i] = row
	}
	return
}

type Member struct {
	VkID int
	Name string

	Place      int
	PromoCode  string
	InfoSended bool
}

type Members []Member

func (m Members) Len() int           { return len(m) }
func (m Members) Less(i, j int) bool { return m[i].Place < m[j].Place }
func (m Members) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

func sendInfo(vkID int) {

}

var messageFmt = `Ваш промокод: %s
%s`

func checkMessages() {
	for _, v := range models.GetPublics() {
		if v.VkAccessToken == "" || v.SheetID == "" {
			continue
		}
		dias, err := vk.GetUnreadedDialogs(v.VkAccessToken)
		if err != nil {
			log.Println("Error getting unreaded dialogs", err)
			continue
		}
		for _, dia := range dias {
			has, _, _, _, promocode, prize, date, _, err := sheets.SearchUserInLastRaffle(v.SheetID, v.SheetName, dia.Message.UserId)
			if err != nil {
				log.Printf("Error getting info from sheets for public %d: %s\n", v.VkID, err)
				continue
			}
			//log.Println(dia.Message.UserId, has, infoSended, promocode, prize)
			//if infoSended {
			//	log.Println("Info already sended", dia.Message.UserId)
			//		continue
			//}

			if dia.Message.UserId == 64684980 {
				log.Println(has, prize)
			}

			messages, err := vk.GetHistory(v.VkAccessToken, dia.Message.UserId, dia.Unread)
			if err != nil {
				log.Println(err)
				continue
			}
			gift := false
			for _, msg := range messages {
				if strings.Contains(strings.ToLower(msg.Body), "приз") {
					gift = true
				}
			}

			// что это?
			_, err = models.InfoSendedGet(v.VkID, dia.Message.UserId, date)
			if err != nil {
				if err != models.ErrNotFound {
					log.Println(err)
					continue
				}
			}

			if has && gift && err == models.ErrNotFound {
				msg, err := render(v.InfoTemplate, promocode, prize, formatDate(date))
				if err != nil {
					log.Println(err)
					continue
				}
				var msgID int
				msgID, err = vk.SendMessage(v.VkAccessToken, dia.Message.UserId, msg)
				if err != nil {
					log.Println(err)
					continue
				}
				err = models.InfoSendedNew(v.VkID, dia.Message.UserId, msgID, date)
				if err != nil {
					log.Println(err)
					continue
				}
				err = sheets.SetUserInfoSended(v.SheetID, v.SheetName, dia.Message.UserId)
				if err != nil {
					log.Println(err)
					continue
				}
			} else if gift {
				// Пользователь написал "приз", но не является победителем

				textTpl, _ := models.NotAWinnerTextGet(v.VkID)
				if textTpl == "" {
					continue
				}

				_, err = vk.SendMessage(v.VkAccessToken, dia.Message.UserId, textTpl)
				if err != nil {
					log.Println(err)
					continue
				}
				//log.Println("User not has in sheet, gift = ", gift, "has = ", has)
			}

		}
	}
}

func formatDate(strDate string) string {
	arr := strings.Split(strDate, "-")
	if len(arr) != 2 {
		return strDate
	}

	startTime, err := time.Parse("02.01", arr[0])
	if err != nil {
		return strDate
	}
	endTime, err := time.Parse("02.01", arr[1])
	if err != nil {
		return strDate
	}

	return fmt.Sprintf("с %d %s по %d %s", startTime.Day(),
		strMounth(startTime.Month()), endTime.Day(), strMounth(endTime.Month()))

}

func strMounth(num time.Month) string {
	if num < 1 || num > 12 {
		return ""
	}
	a := []string{
		"января",
		"февраля",
		"марта",
		"апреля",
		"мая",
		"июня",
		"июля",
		"августа",
		"сентября",
		"октября",
		"ноября",
		"декабря",
	}
	return a[int(num)-1]
}
func render(tpl string, promo, prize, date string) (string, error) {
	buf := bytes.NewBuffer(nil)
	tmpl, err := template.New("alo").Parse(tpl)
	if err != nil {
		return "", err
	}
	tmpl.Execute(buf, map[string]interface{}{
		"prize": prize,
		"promo": promo,
		"date":  date,
	})
	return string(buf.Bytes()), nil
}

func startCheckMessages() {
	for {
		checkMessages()
		time.Sleep(checkInterval * time.Second)
	}
}

func startSynchronizer() {
	for {
		for _, p := range models.GetPublics() {
			err := synchronizer.Job(p.VkAccessToken, p.VkID)
			if err != nil {
				log.Printf("[synchronizer] error: %s", err)
			}
			err = notificator.Job(p.VkAccessToken, p.VkID)
			if err != nil {
				log.Printf("[notificator] error: %s", err)
			}
		}

		time.Sleep(checkInterval * time.Second)
	}
}

func checkTemplate(vkID int) error {
	p := models.GetGroup(vkID)

	_, err := template.New("alo").Parse(p.InfoTemplate)
	return err
}

func formatName(in []string) string {
	if len(in) != 2 {
		return ""
	}
	return strings.Join(in, " ")
}

var (
	dafaultTemplate = `Здравствуйте! Поздравляем!👍
{{if .prize }}Ваш приз: {{ .prize }} {{ end }}
{{if .promo }}Ваш уникальный промо-код: "{{ .promo }}" {{ end }}
Сертификатом Вы можете воспользоваться {{ .date }} включительно.
Свой сертификат Вы можете передать другому человеку, только обязательно сообщите ему свой промо-код
Вам посчитать стоимость установки потолка с учетом скидки? Эта цена со скидкой закрепляется за Вами, даже если Вы не планируете сейчас ставить потолок

Напишите пожалуйста ваш номер телефона, и время когда вам было бы удобно принять звонок. Наш менеджер свяжется с вами чтобы рассказать как воспользоваться Сертификатом.`
)
