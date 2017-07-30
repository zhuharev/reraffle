package sheets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/Unknwon/com"
	cache "github.com/patrickmn/go-cache"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"

	"github.com/zhuharev/reraffle/models"
	"github.com/zhuharev/reraffle/modules/bindata/public"
)

var (
	srv    *sheets.Service
	ctx    context.Context
	config *oauth2.Config
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		return nil, err
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {

		// tok, err = getTokenFromWeb(config)
		// if err != nil {
		// 	return nil, err
		// }
		// saveToken(cacheFile, tok)
	}

	if tok.Expiry.Before(time.Now()) {
		log.Printf("need to renew new access token")
		tok = RenewToken(config, tok, cacheFile)
	}

	return config.Client(ctx, tok), nil
}

func RenewToken(config *oauth2.Config, tok *oauth2.Token, cacheFile string) *oauth2.Token {

	urlValue := url.Values{"client_id": {config.ClientID}, "client_secret": {config.ClientSecret}, "refresh_token": {tok.RefreshToken}, "grant_type": {"refresh_token"}}

	resp, err := http.PostForm("https://www.googleapis.com/oauth2/v3/token", urlValue)
	if err != nil {
		log.Fatalf("Error when renew token %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", body)
	var refreshToken *oauth2.Token
	json.Unmarshal([]byte(body), &refreshToken)

	log.Println(string(body))

	fmt.Printf("%+v", refreshToken)

	//then := time.Now()
	//then = then.Add(time.Duration(refreshToken.Expiry) * time.Second)

	tok.Expiry = refreshToken.Expiry
	tok.AccessToken = refreshToken.AccessToken
	saveToken(cacheFile, tok)

	return tok

}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, err
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func SaveToken(code string) error {
	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	cacheFile, err := tokenCacheFile()
	if err != nil {
		return err
	}
	err = saveToken(cacheFile, tok)
	return err
}

func AuthURL() string {
	return config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	return filepath.Join(url.QueryEscape("sheets.googleapis.com-go-quickstart.json")), nil
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

func in() {
	ctx = context.Background()

	b, err := public.Asset("client_secret.json")
	if err != nil {
		log.Println("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/sheets.googleapis.com-go-quickstart.json
	config, err = google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Println("Unable to parse client secret file to config: %v", err)
		return
	}
	client, err := getClient(ctx, config)
	if err != nil {
		log.Println("Unable to parse client secret file to config: %v", err)
		return
	}
	srv, err = sheets.New(client)
	if err != nil {
		log.Println("Unable to retrieve Sheets Client %v", err)
	}
}

func Init() {
	in()
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			in()
		}
	}()
}

func Append(spreadsheetID string, sheetName string, values [][]interface{}) error {
	lastCell, err := GetLastCell(spreadsheetID, sheetName)
	if err != nil {
		return err
	}
	writeRange := fmt.Sprintf("%s!A%d:D", sheetName, lastCell+1)

	log.Println("Send values", values)

	_, err = srv.Spreadsheets.Values.Update(spreadsheetID, writeRange, &sheets.ValueRange{Values: values}).
		ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}

func GetLastCell(spreadsheetID string, sheetName string) (int, error) {
	readRange := sheetName + "!C2:C"
	vr, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return 0, err
	}
	last := len(vr.Values)
	for i := len(vr.Values) - 1; i >= 0; i-- {
		if len(vr.Values[i]) < 3 {
			continue
		}
		if str, ok := vr.Values[i][2].(string); ok && str == "" {
			last--
		}
	}
	return last + 1, nil
}

func GetRaffleCount(spreadsheetID string, sheetName string) (int, error) {
	readRange := sheetName + "!A2:A"
	vr, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return 0, err
	}
	cnt := 0
	for i := len(vr.Values) - 1; i >= 0; i-- {
		if len(vr.Values[i]) < 1 {
			continue
		}
		if str, ok := vr.Values[i][0].(string); ok && str != "" {
			cnt++
		}
	}
	return cnt, nil
}

// func GetSpreadSheet(spreadsheetID string) {
// 	sh, err := srv.Spreadsheets.Get(spreadsheetID).IncludeGridData(true).Do()
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}
// }

func SetUserInfoSended(spreadsheetID string, sheetName string, vkID int) error {
	has, index, _, _, _, _, _, shetID, err := SearchUserInLastRaffle(spreadsheetID, sheetName, vkID)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("Пользователь не участвует в розыгрыше")
	}
	_, err = srv.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{Requests: []*sheets.Request{
		&sheets.Request{
			UpdateCells: &sheets.UpdateCellsRequest{
				Start: &sheets.GridCoordinate{RowIndex: index, ColumnIndex: 2, SheetId: shetID},
				Rows: []*sheets.RowData{
					&sheets.RowData{Values: []*sheets.CellData{{
						UserEnteredFormat: &sheets.CellFormat{
							BackgroundColor: &sheets.Color{
								Green: 0.7,
							},
						},
					}}},
				},
				Fields: "userEnteredFormat(backgroundColor)"},
			//UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
			//	Properties: &sheets.SheetProperties{Title: sheetName},
			//},
		},
	}}).Context(ctx).Do()
	if err != nil {
		return err
	}
	return nil
}

func HealtsCheck(sID, sName string) (err error) {
	sh, err := srv.Spreadsheets.Get(sID).IncludeGridData(true).Do()
	if err != nil {
		return
	}

	if len(sh.Sheets) == 0 {
		err = fmt.Errorf("Ошибка, нет листов в гугл-документе")
		return
	}

	idxFound := false
	idx := 0
	//var sheetID int64
	for i, v := range sh.Sheets {
		if v.Properties.Title == sName {
			idxFound = true
			idx = i
			//sheetID = v.Properties.SheetId
			break
		}
	}
	if !idxFound {
		err = fmt.Errorf("Лист не найден")
		return
	}

	if len(sh.Sheets[idx].Data) == 0 {
		err = fmt.Errorf("Ошибка, нет данных в гугл-документе")
		return
	}

	for i := len(sh.Sheets[idx].Data[0].RowData) - 1; i >= 0; i-- {
		row := sh.Sheets[idx].Data[0].RowData[i]
		if len(row.Values) < 3 {
			continue
		}
		if row.Values[2].FormattedValue == "" || row.Values[2].FormattedValue[0] != '[' {
			continue
		}
		var curID int
		var curName string
		var infoSended bool
		var (
			promocode string
			prize     string
			date      string
		)
		fmt.Sscanf(row.Values[2].FormattedValue, "[id%d|%s]", &curID, &curName)
		place := com.StrTo(row.Values[1].FormattedValue).MustInt()
		if len(row.Values) > 2 {
			log.Println(row.Values[2].UserEnteredFormat)
			infoSended = (row.Values[2].UserEnteredFormat != nil &&
				row.Values[2].UserEnteredFormat.BackgroundColor != nil &&
				row.Values[2].UserEnteredFormat.BackgroundColor.Green > 0.5)
		}

		index := int64(i)
		if len(row.Values) > 3 {
			promocode = row.Values[3].FormattedValue
		}
		if len(row.Values) > 4 {
			prize = row.Values[4].FormattedValue
		}
		if len(row.Values) > 5 {
			date = row.Values[5].FormattedValue
		}
		if !infoSended {
			//	log.Println("Already sended", len(sh.Sheets[idx].Data[0].RowData)-1-i, curName, curID, prize)
			break
		}

		log.Println(place, infoSended, prize, promocode, date, index)
	}

	return nil
}

var (
	cach *cache.Cache
)

func init() {
	cach = cache.New(5*time.Minute, 1*time.Minute)
}

// GetRows return users line by line
func GetRows(spreadsheetID string, sheetName string, force ...bool) (rows []models.Row, err error) {
	if iface, has := cach.Get(spreadsheetID); has && len(force) == 0 {
		return iface.([]models.Row), nil
	}

	rows, err = getRows(spreadsheetID, sheetName)
	if err != nil {
		return nil, err
	}

	cach.Set(spreadsheetID, rows, 5*time.Minute)

	return
}

// getRows return users line by line
func getRows(spreadsheetID string, sheetName string) (rows []models.Row, err error) {
	sh, err := srv.Spreadsheets.Get(spreadsheetID).IncludeGridData(true).Do()
	if err != nil {
		return
	}
	if len(sh.Sheets) == 0 {
		err = fmt.Errorf("Ошибка, нет листов в гугл-документе")
		return
	}

	idx := 0
	for i, v := range sh.Sheets {
		if v.Properties.Title == sheetName {
			idx = i
			break
		}
	}
	if len(sh.Sheets[idx].Data) == 0 {
		err = fmt.Errorf("Ошибка, нет данных в гугл-документе")
		return
	}

	var (
		currentDate = ""
	)

	for i := 0; i < len(sh.Sheets[idx].Data[0].RowData); i++ {
		r := sh.Sheets[idx].Data[0].RowData[i]
		row := models.Row{}
		for i, v := range r.Values {
			switch i {
			case 0:
				if v.FormattedValue != "" {
					currentDate = v.FormattedValue
				}
				row.Date = currentDate
			case 1:
				row.Place = com.StrTo(v.FormattedValue).MustInt()
			case 2:
				var curID int
				var curName string
				fmt.Sscanf(v.FormattedValue, "[id%d|%s]", &curID, &curName)
				row.Name = curName
				row.VkID = curID

				infoSended := (v.UserEnteredFormat != nil &&
					v.UserEnteredFormat.BackgroundColor != nil &&
					v.UserEnteredFormat.BackgroundColor.Green > 0.5 &&
					v.UserEnteredFormat.BackgroundColor.Green < 0.9)

				row.InfoSended = infoSended
			case 3:
				row.Promocode = v.FormattedValue
			case 4:
				row.Prize = v.FormattedValue
			case 5:
				row.Date = v.FormattedValue
			}
		}

	}

	return
}

func SearchUserInLastRaffle(spreadsheetID string,
	sheetName string,
	vkID int) (has bool,
	index int64, place int, infoSended bool, promocode string, prize string, date string, sheetID int64, err error) {

	//readRange := "Sheet1!A2:D"
	sh, err := srv.Spreadsheets.Get(spreadsheetID).IncludeGridData(true).Do()
	if err != nil {
		return
	}
	br := false
	if len(sh.Sheets) == 0 {
		err = fmt.Errorf("Ошибка, нет листов в гугл-документе")
		return
	}
	idx := 0
	for i, v := range sh.Sheets {
		if v.Properties.Title == sheetName {
			idx = i
			sheetID = v.Properties.SheetId
			break
		}
	}
	if len(sh.Sheets[idx].Data) == 0 {
		err = fmt.Errorf("Ошибка, нет данных в гугл-документе")
		return
	}
	for i := len(sh.Sheets[idx].Data[0].RowData) - 1; i >= 0; i-- {
		row := sh.Sheets[idx].Data[0].RowData[i]
		if len(row.Values) < 3 {
			continue
		}
		if row.Values[0].FormattedValue != "" && date == "" {
			//date = row.Values[0].FormattedValue
			br = true
		}
		if row.Values[2].FormattedValue == "" || row.Values[2].FormattedValue[0] != '[' {
			continue
		}
		var curID int
		var curName string
		fmt.Sscanf(row.Values[2].FormattedValue, "[id%d|%s]", &curID, &curName)
		if vkID == curID {
			has = true
			place = com.StrTo(row.Values[1].FormattedValue).MustInt()
			if len(row.Values) > 2 {
				//log.Println(row.Values[2].UserEnteredFormat)
				infoSended = (row.Values[2].UserEnteredFormat != nil &&
					row.Values[2].UserEnteredFormat.BackgroundColor != nil &&
					row.Values[2].UserEnteredFormat.BackgroundColor.Green > 0.5 &&
					row.Values[2].UserEnteredFormat.BackgroundColor.Green < 0.9)
			}

			index = int64(i)
			if len(row.Values) > 3 {
				promocode = row.Values[3].FormattedValue
			}
			if len(row.Values) > 4 {
				prize = row.Values[4].FormattedValue
			}
			if len(row.Values) > 5 {
				date = row.Values[5].FormattedValue
			}
			if !infoSended {
				//	log.Println("Already sended", len(sh.Sheets[idx].Data[0].RowData)-1-i, curName, curID, prize)
				break
			}
		}

		if br {
			break
		}
	}
	return
}
