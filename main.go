package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

	macaron "gopkg.in/macaron.v1"

	"github.com/Unknwon/com"
	"github.com/go-macaron/bindata"
	"github.com/go-macaron/session"
	dry "github.com/ungerik/go-dry"

	"github.com/zhuharev/reraffle/controllers"
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
	publics []Public

	debug                       = false
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

		if _, ok := sess.Get("user").(string); !ok {
			if c.Req.RequestURI != "/login" && c.Req.RequestURI != "/favicon.ico" {
				log.Println("redirect unautarized")
				c.Redirect("/login")
				return
			}
		}

		c.Next()
	})

	m.Post("/update_notify_text", controllers.UpdateNotifyText)
	m.Post("/update_end_text", controllers.UpdateEndText)

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
		c.Data["publics"] = publics
		c.HTML(200, "index")
	})

	m.Get("/publics/:id/healts", func(c *macaron.Context) {
		id := c.ParamsInt(":id")
		p := getGroup(id)
		err := vk.HealtsCheck(p.VkAccessToken)
		c.Data["h_vk"] = err
		err = sheets.HealtsCheck(p.SheetID, p.SheetName)
		c.Data["h_sh"] = err
		err = checkTemplate(id)
		c.Data["h_tm"] = err

		c.HTML(200, "healts")

	})

	m.Get("/add_public", func(c *macaron.Context) {
		c.Data["template"] = dafaultTemplate
		c.HTML(200, "add")
	})

	m.Get("/publics/:id/delete", func(c *macaron.Context) {
		deleteGroup(c.ParamsInt(":id"))
		c.Redirect("/")
	})

	m.Post("/edit/:id", func(c *macaron.Context) {
		p := Public{
			PromoCodeTemplate: c.Query("promocode_template"),
			InfoTemplate:      c.Query("answer_template"),
			VkAccessToken:     c.Query("vk_key"),
			VkID:              c.ParamsInt(":id"),
			LastRaffle:        c.QueryInt("last_raffle"),
			SheetID:           c.Query("sheet_id"),
			SheetName:         c.Query("sheet_name"),
		}
		name, err := vk.GetPublicName(p.VkID)
		if err != nil {
			log.Println(err)
		}
		p.Title = name
		log.Println("updated group ", p)
		updateGroup(p)

		c.Redirect("/edit/" + c.Params(":id"))
	})

	m.Post("/add_public", func(c *macaron.Context) {
		p := Public{
			PromoCodeTemplate: c.Query("promocode_template"),
			InfoTemplate:      c.Query("answer_template"),
			VkAccessToken:     c.Query("vk_key"),
			VkID:              c.QueryInt("public_id"),
			LastRaffle:        c.QueryInt("last_raffle"),
			SheetID:           c.Query("sheet_id"),
			SheetName:         c.Query("sheet_name"),
		}
		name, err := vk.GetPublicName(p.VkID)
		if err != nil {
			log.Println(err)
		}
		p.Title = name
		log.Println("added group ", p)
		err = addGroup(p)
		if err != nil {
			c.Data["error"] = err.Error()
			c.HTML(200, "error")
			return
		}

		c.Redirect("/?t=1")
	})

	m.Post("/add_raffle", func(c *macaron.Context) {
		raffleURL := c.Query("raffle_url")
		err := addRaffle(raffleURL)
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
		for _, v := range publics {
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
		post := com.StrTo(arr[1]).MustInt()

		for pi, v := range publics {
			if v.VkID == own {
				for ri, r := range v.Raffles {
					if r.PostID == post {

						if c.QueryBool("update") {
							ids, err := vk.GetRaffleMembers(-own, post, []int{own})
							if err != nil {
								log.Println(err)
								continue
							}
							m := Members{}

							names, err := vk.GetUserNames(ids)
							if err != nil {
								log.Println(err)
								continue
							}

							for _, v := range ids {
								m = append(m, Member{VkID: v, Name: formatName(names[v])})
							}
							publics[pi].Raffles[ri].Members = m
						}

						c.Data["raffle"] = publics[pi].Raffles[ri]
					}
				}
				c.Data["public"] = v
			}
		}
		c.HTML(200, "raffle")
	})

	m.Get("/shuffle/:id", func(c *macaron.Context) {
		arr := strings.Split(c.Params(":id"), "_")
		own := com.StrTo(arr[0]).MustInt()
		post := com.StrTo(arr[1]).MustInt()
		for pi, v := range publics {
			if v.VkID == own {
				for ri, r := range v.Raffles {
					if r.PostID == post {
						publics[pi].Raffles[ri].Members = shaffleMembers(pi, r.Members)
						flushPublics()
						c.Data["raffle"] = publics[pi].Raffles[ri]
					}
				}
				c.Data["public"] = v
			}
		}
		c.Redirect("/raffles/" + c.Params(":id"))
	})

	m.Get("/send_to_sheet", func(c *macaron.Context) {
		r, p := getRaffle(c.Query("id"))
		err := sheets.Append(p.SheetID, p.SheetName, r.ToValues())
		if err != nil {
			log.Println(err)
		}

		c.Redirect("/")
	})

	m.Get("/settings", func(c *macaron.Context) {

		endText, _ := models.EndTextGet()
		notifyText, _ := models.NotifyTextGet()
		c.Data["notify"] = notifyText
		c.Data["endText"] = endText

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
		gr := getGroup(c.ParamsInt(":id"))
		c.Data["group"] = gr
		c.HTML(200, "edit")
	})

	m.Run(3381)
}

func getRaffle(id string) (ra Raffle, p Public) {
	arr := strings.Split(id, "_")
	own := com.StrTo(arr[0]).MustInt()
	post := com.StrTo(arr[1]).MustInt()

	for pi, v := range publics {
		if v.VkID == own {
			p = v
			for ri, r := range v.Raffles {
				if r.PostID == post {
					ra = publics[pi].Raffles[ri]
					log.Println("found", ra)
					return
				}
			}
		}
	}
	return
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
	readPublics()

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
	go startNotificator()
	runServer()

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

func shaffleMembers(pid int, in Members) Members {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	indexes := r.Perm(len(in))

	lr, err := sheets.GetRaffleCount(publics[pid].SheetID, publics[pid].SheetName)
	if err != nil {
		log.Println(err)
	}
	//res := make([]Members, len(in))
	for i, v := range indexes {
		in[i].Place = v + 1
		in[i].PromoCode = fmt.Sprintf(publics[pid].PromoCodeTemplate, lr+1, v+1)
	}
	sort.Sort(in)
	return in
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

func addGroup(p Public) error {
	publics = append(publics, p)
	return flushPublics()
}

func getGroup(vkID int) Public {
	for i, v := range publics {
		if v.VkID == vkID {
			return publics[i]
		}
	}
	return Public{}
}

func updateGroup(p Public) {
	for i, v := range publics {
		if p.VkID == v.VkID {
			publics[i] = p
		}
	}
	flushPublics()
}

func deleteGroup(id int) error {
	for i, v := range publics {
		if v.VkID == id {
			publics = append(publics[:i], publics[i+1:]...)
			break
		}
	}
	return flushPublics()
}

// ex: https://vk.com/wall-147932966_2
func addRaffle(ru string) error {
	var gID int
	var pID int
	_, err := fmt.Sscanf(ru, "https://vk.com/wall-%d_%d", &gID, &pID)
	if err != nil {
		return err
	}

	for i, v := range publics {
		if v.VkID == gID {
			r := Raffle{
				StartDate: time.Now(),
				OwnerID:   gID,
				PostID:    pID,
			}
			publics[i].Raffles = append(publics[i].Raffles, r)
			log.Println("added rafle")
			break
		}
	}

	return flushPublics()
}

func readPublics() error {
	bts, err := ioutil.ReadFile("publics.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(bts, &publics)
}

func flushPublics() error {
	bts, err := json.MarshalIndent(publics, "  ", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile("publics.json", bts, 0777)
}

func sendInfo(vkID int) {

}

var messageFmt = `Ваш промокод: %s
%s`

func checkMessages() {
	for _, v := range publics {
		if v.VkAccessToken == "" || v.SheetID == "" {
			continue
		}
		dias, err := vk.GetUnreadedDialogs(v.VkAccessToken)
		if err != nil {
			//	log.Println(err)
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
			if has && gift {
				msg, err := render(v.InfoTemplate, promocode, prize, formatDate(date))
				if err != nil {
					log.Println(err)
					continue
				}
				var msgID int
				msgID, err = vk.SendMessage(v.VkAccessToken, dia.Message.UserId, msg)
				if err != nil {
					log.Println(err)
					return
				}
				err = models.InfoSendedNew(v.VkID, dia.Message.UserId, msgID, date)
				if err != nil {
					log.Println(err)
					return
				}
				err = sheets.SetUserInfoSended(v.SheetID, v.SheetName, dia.Message.UserId)
				if err != nil {
					log.Println(err)
					return
				}
			} else {
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
		for _, p := range publics {
			err := synchronizer.Job(p.VkAccessToken, p.VkID)
			if err != nil {
				log.Printf("[synchronizer] error: %s", err)
			}
		}

		time.Sleep(checkInterval * time.Second)
	}
}

func startNotificator() {
	// wait synchronizer
	time.Sleep(20 * time.Second)
	log.Println("startNotificator")
	for {
		for _, p := range publics {
			err := notificator.Job(p.VkAccessToken, p.VkID)
			if err != nil {
				log.Printf("[notificator] error: %s", err)
			}
		}

		time.Sleep(60 * time.Second)
	}
}

func checkTemplate(vkID int) error {
	p := getGroup(vkID)

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
