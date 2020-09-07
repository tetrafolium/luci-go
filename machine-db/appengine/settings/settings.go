// Copyright 2017 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package settings

import (
	"context"
	"html/template"

	"github.com/tetrafolium/luci-go/server/portal"
	"github.com/tetrafolium/luci-go/server/settings"
)

// settingsKey is the name used to read/write these settings.
const settingsKey = "database"

// DatabaseSettings are app-level settings configuring how to connect to the SQL database.
type DatabaseSettings struct {
	// Server is the address of the SQL server is hosted.
	Server string `json:"server"`
	// Username is the username to authenticate to the server with.
	Username string `json:"username"`
	// Password is the password to authenticate to the server with.
	Password string `json:"password"`
	// Database is the name of the SQL database to connect to.
	Database string `json:"database"`
}

// New returns a new instance of DatabaseSettings.
func New(c context.Context) *DatabaseSettings {
	return &DatabaseSettings{
		Server:   "",
		Username: "",
		Password: "",
		Database: "",
	}
}

// Get returns the current settings. This may hit an outdated cache.
func Get(c context.Context) (*DatabaseSettings, error) {
	databaseSettings := &DatabaseSettings{}
	switch err := settings.Get(c, settingsKey, databaseSettings); err {
	case nil:
		return databaseSettings, nil
	case settings.ErrNoSettings:
		return New(c), nil
	default:
		return nil, err
	}
}

// GetUncached returns the latest settings, bypassing the cache.
func GetUncached(c context.Context) (*DatabaseSettings, error) {
	databaseSettings := &DatabaseSettings{}
	switch err := settings.GetUncached(c, settingsKey, databaseSettings); err {
	case nil:
		return databaseSettings, nil
	case settings.ErrNoSettings:
		return New(c), nil
	default:
		return nil, err
	}
}

// Title returns the title of this settings page.
func (*DatabaseSettings) Title(c context.Context) (string, error) {
	return "Database settings", nil
}

// Overview returns an overview of these settings.
func (*DatabaseSettings) Overview(c context.Context) (template.HTML, error) {
	return "<p>SQL database configuration</p>", nil
}

// Fields returns the form fields for configuring these settings.
func (*DatabaseSettings) Fields(c context.Context) ([]portal.Field, error) {
	fields := []portal.Field{
		{
			ID:    "Server",
			Title: "Server",
			Type:  portal.FieldText,
			Help:  "<p>Server to use (for Cloud SQL should be of the form project:region:database)</p>",
		},
		{
			ID:    "Username",
			Title: "Username",
			Type:  portal.FieldText,
			Help:  "<p>Username to authenticate to the SQL database with</p>",
		},
		{
			ID:    "Password",
			Title: "Password",
			Type:  portal.FieldPassword,
			Help:  "<p>Password to authenticate to the SQL database with</p>",
		},
		{
			ID:    "Database",
			Title: "Database",
			Type:  portal.FieldText,
			Help:  "<p>SQL database to use</p>",
		},
	}
	return fields, nil
}

// Actions is additional list of actions to present on the page.
func (*DatabaseSettings) Actions(c context.Context) ([]portal.Action, error) {
	return nil, nil
}

// ReadSettings returns settings for display.
func (*DatabaseSettings) ReadSettings(c context.Context) (map[string]string, error) {
	databaseSettings, err := GetUncached(c)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"Server":   databaseSettings.Server,
		"Username": databaseSettings.Username,
		"Password": databaseSettings.Password,
		"Database": databaseSettings.Database,
	}, nil
}

// WriteSettings commits any changes to the settings.
func (*DatabaseSettings) WriteSettings(c context.Context, values map[string]string, who, why string) error {
	databaseSettings := &DatabaseSettings{
		Server:   values["Server"],
		Username: values["Username"],
		Password: values["Password"],
		Database: values["Database"],
	}

	return settings.SetIfChanged(c, settingsKey, databaseSettings, who, why)
}

// init registers the database settings UI.
func init() {
	portal.RegisterPage(settingsKey, &DatabaseSettings{})
}
