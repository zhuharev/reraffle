package models

import (
	"encoding/json"
	"io/ioutil"
)

var (
	publics []Public
)

func GetPublics() []Public {
	return publics
}

func AddGroup(p Public) error {
	publics = append(publics, p)
	return FlushPublics()
}

func GetGroup(vkID int) Public {
	for i, v := range publics {
		if v.VkID == vkID {
			return publics[i]
		}
	}
	return Public{}
}

func FlushPublics() error {
	bts, err := json.MarshalIndent(publics, "  ", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile("publics.json", bts, 0777)
}

func UpdateGroup(p Public) {
	for i, v := range publics {
		if p.VkID == v.VkID {
			publics[i] = p
		}
	}
	FlushPublics()
}

func DeleteGroup(id int) error {
	for i, v := range publics {
		if v.VkID == id {
			publics = append(publics[:i], publics[i+1:]...)
			break
		}
	}
	return FlushPublics()
}

func ReadPublics() error {
	bts, err := ioutil.ReadFile("publics.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(bts, &publics)
}
