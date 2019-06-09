package server

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"sync"
	"time"

	"github.com/cbeuw/Cloak/internal/server/usermanager"
)

type rawConfig struct {
	ProxyBook     map[string]string
	RedirAddr     string
	PrivateKey    string
	AdminUID      string
	DatabasePath  string
	BackupDirPath string
}

// State type stores the global state of the program
type State struct {
	ProxyBook map[string]string

	BindHost string
	BindPort string

	Now         func() time.Time
	AdminUID    []byte
	staticPv    crypto.PrivateKey
	Userpanel   *usermanager.Userpanel
	usedRandomM sync.RWMutex
	usedRandom  map[[32]byte]int

	RedirAddr string
}

func InitState(bindHost, bindPort string, nowFunc func() time.Time) (*State, error) {
	ret := &State{
		BindHost: bindHost,
		BindPort: bindPort,
		Now:      nowFunc,
	}
	ret.usedRandom = make(map[[32]byte]int)
	return ret, nil
}

// ParseConfig parses the config (either a path to json or in-line ssv config) into a State variable
func (sta *State) ParseConfig(conf string) (err error) {
	var content []byte
	var preParse rawConfig

	content, errPath := ioutil.ReadFile(conf)
	if errPath != nil {
		errJson := json.Unmarshal(content, &preParse)
		if errJson != nil {
			return errors.New("Failed to read/unmarshal configuration, path is invalid or " + errJson.Error())
		}
	} else {
		errJson := json.Unmarshal(content, &preParse)
		if errJson != nil {
			return errors.New("Failed to read configuration file: " + errJson.Error())
		}
	}

	up, err := usermanager.MakeUserpanel(preParse.DatabasePath, preParse.BackupDirPath)
	if err != nil {
		return errors.New("Attempting to open database: " + err.Error())
	}
	sta.Userpanel = up

	sta.RedirAddr = preParse.RedirAddr
	sta.ProxyBook = preParse.ProxyBook

	pvBytes, err := base64.StdEncoding.DecodeString(preParse.PrivateKey)
	if err != nil {
		return errors.New("Failed to decode private key: " + err.Error())
	}
	var pv [32]byte
	copy(pv[:], pvBytes)
	sta.staticPv = &pv

	adminUID, err := base64.StdEncoding.DecodeString(preParse.AdminUID)
	if err != nil {
		return errors.New("Failed to decode AdminUID: " + err.Error())
	}
	sta.AdminUID = adminUID
	return nil
}

// UsedRandomCleaner clears the cache of used random fields every 12 hours
func (sta *State) UsedRandomCleaner() {
	for {
		time.Sleep(12 * time.Hour)
		now := int(sta.Now().Unix())
		sta.usedRandomM.Lock()
		for key, t := range sta.usedRandom {
			if now-t > 12*3600 {
				delete(sta.usedRandom, key)
			}
		}
		sta.usedRandomM.Unlock()
	}
}
