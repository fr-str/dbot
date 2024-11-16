package config

import (
	"github.com/fr-str/env"
)

var (
	TOKEN    = env.Get[string]("TOKEN")
	GUILD_ID = env.Get("GUILD_ID", "")
	_        = TOKEN
)
