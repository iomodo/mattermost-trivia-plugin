package main

import (
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	botUserID string
	userData  map[string]*UserData
	dataLock  *sync.RWMutex
}

// UserData describes data of a current user
type UserData struct {
	Question *Question
	Timer    *time.Timer
}

func (p *Plugin) setupBot() error {
	botID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    "suggestions",
		DisplayName: "Suggestions",
		Description: "Created by the Suggestions plugin.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure suggestions bot")
	}
	p.botUserID = botID
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "couldn't read profile image")
	}

	appErr := p.API.SetProfileImage(botID, profileImage)
	if appErr != nil {
		return errors.Wrap(appErr, "couldn't set profile image")
	}
	return nil
}

// OnActivate will be run on plugin activation.
func (p *Plugin) OnActivate() error {
	p.API.RegisterCommand(getCommand())

	err := p.setupBot()
	if err != nil {
		return err
	}

	var lock sync.RWMutex
	p.dataLock = &lock
	p.userData = make(map[string]*UserData)
	userPoints := make(map[string]int)
	p.Helpers.KVSetJSON(trigger, userPoints)

	return nil
}
