package vk

import (
	"fmt"
	"log"
	"net/url"

	"github.com/zhuharev/vk"
	"github.com/zhuharev/vkutil"
)

var (
	// AT - access token
	// 6058492
	AT = "d034ea9d5675ea901e9d277e79911c7fd38b01bed8ad46bf578c34b13064ff4dcc4b65b119a1db0c87775" // "dd53be31dd53be31dd53be31a7dd0fcfcdddd53dd53be3184620e3723eb71a45a999867"

	MSGAT = "794e9fb8ed553908d8c4b50b855716168ad4403d56f6d544f39bc933cc8183e87e037749eabfbe081dd64"

	api *vkutil.Api

	nameCache = map[int][]string{}

	debug = false
)

// GetRaffleMembers return users
func GetRaffleMembers(ownerID int, postID int, publics []int) ([]int, error) {

	log.Println(vk.GetAuthURL(vk.DefaultRedirectURI, "token", "6058492", "wall"))

	api := vkutil.New()
	api.VkApi.AccessToken = AT

	list, err := api.LikesGetListAll(vkutil.OBJECT_POST, ownerID, postID, url.Values{"filter": {"copies"}})
	if err != nil {
		return nil, err
	}

	memberGroups := make([][]int, len(publics))
	for i, gID := range publics {
		memberGroups[i], err = api.GroupsGetAllMembers(gID)
		if err != nil {
			return nil, err
		}
	}

	var members []int
	var ok bool
	for _, id := range list {
		ok = false
		for i := range memberGroups {
			if inArr(id, memberGroups[i]) {
				ok = true
			} else {
				ok = false
			}
		}
		if ok {
			members = append(members, id)
		}
	}

	return members, nil
}

func HealtsCheck(token string) error {
	_, err := GetUnreadedDialogs(token)
	if err != nil {
		return err
	}
	return nil
}

func GetUnreadedDialogs(token string) ([]vkutil.Dialog, error) {
	return GetDialogs(token, true)
}

func GetDialogs(token string, onlyUnread bool) ([]vkutil.Dialog, error) {
	api := vkutil.New()
	api.VkApi.AccessToken = token
	api.SetDebug(debug)
	api.VkApi.SetDebug(debug)
	values := url.Values{"count": {"200"}}
	if onlyUnread {
		values["unread"] = []string{"1"}
	}
	return api.MessagesGetDialogs(values)
}

func GetHistory(token string, userID int, count int) ([]vkutil.Message, error) {
	api := vkutil.New()
	api.VkApi.AccessToken = token
	api.SetDebug(debug)
	api.VkApi.SetDebug(debug)
	return api.MessagesGetHistory(userID, url.Values{"count": {fmt.Sprint(count)}})
}

// SendMessage sends an message to vk.com via api
func SendMessage(token string, userID int, message string) (int, error) {
	api := vkutil.New()
	api.VkApi.AccessToken = token
	api.SetDebug(debug)
	api.VkApi.SetDebug(debug)
	return api.MessagesSend(userID, message)
}

func GetPublicName(id int) (string, error) {
	api := vkutil.New()
	api.VkApi.AccessToken = AT
	api.VkApi.Lang = "ru"
	api.SetDebug(debug)
	api.VkApi.SetDebug(debug)
	groups, err := api.GroupsGetByID(id)
	if err != nil {
		return "", err
	}
	if len(groups) != 1 {
		return "", fmt.Errorf("Неизвестная ошибка")
	}

	return groups[0].Name, nil
}

func GetUserNames(ids []int) (map[int][]string, error) {
	api := vkutil.New()
	api.VkApi.AccessToken = AT
	api.VkApi.Lang = "ru"
	api.SetDebug(debug)
	api.VkApi.SetDebug(debug)

	var need []int
	var res = map[int][]string{}
	for _, v := range ids {
		if name, has := nameCache[v]; has {
			res[v] = name
		} else {
			need = append(need, v)
		}
	}

	users, err := api.UsersGet(need)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, u := range users {
		name := []string{u.LastName, u.FirstName}
		res[u.Id] = name
		nameCache[u.Id] = name
	}

	return res, nil
}

func inArr(need int, arr []int) bool {
	for _, v := range arr {
		if v == need {
			return true
		}
	}
	return false
}
