package app

import (
	"autossh/src/utils"
)

func showVersion() {
	utils.Logln("autossh " + Version + " Build " + Build + "。")
}
