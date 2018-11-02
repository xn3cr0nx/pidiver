package logs

import (
	"os"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	logFormat = "%{color}[%{level:.4s}] %{time:15:04:05.000000} %{id:06x} [%{shortpkg}] %{longfunc} -> %{color:reset}%{message}"
	Log       = logging.MustGetLogger("hercules")
	config    *viper.Viper
)

func Start() {
	logging.SetFormatter(logging.MustStringFormatter(logFormat))
}

func SetConfig(viperConfig *viper.Viper) {
	config = viperConfig
	consoleBackEnd := logging.NewLogBackend(os.Stdout, "", 0)

	level, err := logging.LogLevel(config.GetString("log.level"))
	if err == nil {
		consoleBackEndLeveled := logging.AddModuleLevel(consoleBackEnd)
		consoleBackEndLeveled.SetLevel(level, "server")

		logging.SetBackend(consoleBackEndLeveled)

	} else {
		Log.Warningf("Could not set log level to %v: %v", config.GetString("level"), err)
		Log.Warning("Using default log level")
	}
}
