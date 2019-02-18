package config

import (
	"encoding/json"
	"os"

	"github.com/shufps/pidiver/server/logs"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	AppConfig = viper.New()
)

const (
	MAX_SNAPSHOT_PERIOD = 12
)

// TODO: Refactor it further
/*
PRECEDENCE (Higher number overrides the others):
1. default
2. key/value store
3. config
4. env
5. flag
6. explicit call to Set
*/
func Start() {
	// 1. Set defaults
	//config.SetDefault("test", 0)

	flag.Bool("debug", false, "Run hercules when debugging the source-code")

	declareAPIConfigs()
	declareLogConfigs()

	AppConfig.BindPFlags(flag.CommandLine)

	var configPath = flag.StringP("config", "c", "server.config.json", "Config file path")
	flag.Parse()

	loadAppConfigFile(configPath)

	logs.SetConfig(AppConfig)

	cfg, _ := json.MarshalIndent(AppConfig.AllSettings(), "", "  ")
	logs.Log.Debugf("Settings loaded: \n %+v", string(cfg))
}

func declareAPIConfigs() {
	flag.String("api.auth.username", "", "API Access Username")
	flag.String("api.auth.password", "", "API Access Password")

	flag.Bool("api.cors.setAllowOriginToAll", true, "Defines if 'Access-Control-Allow-Origin' is set to '*'")

	flag.Bool("api.http.useHttp", true, "Defines if the API will serve using HTTP protocol")
	flag.StringP("api.http.host", "h", "0.0.0.0", "HTTP API Host")
	flag.IntP("api.http.port", "p", 14265, "HTTP API Port")
	flag.StringP("api.http.node", "n", "https://iota1.thingslab.network", "IOTA node host")

	flag.Bool("api.https.useHttps", false, "Defines if the API will serve using HTTPS protocol")
	flag.String("api.https.host", "0.0.0.0", "HTTPS API Host")
	flag.Int("api.https.port", 14266, "HTTPS API Port")
	flag.String("api.https.certificatePath", "cert.pem", "Path to TLS certificate (non-encrypted)")
	flag.String("api.https.privateKeyPath", "key.pem", "Path to private key used to isse the TLS certificate (non-encrypted)")
	flag.String("api.https.node", "https://iota1.thingslab.network", "IOTA node host")

	flag.StringSlice("api.limitRemoteAccess", nil, "Limit access to these commands from remote")

	flag.Int("api.pow.maxMinWeightMagnitude", 14, "Maximum Min-Weight-Magnitude (Difficulty for PoW)")
	flag.Int("api.pow.maxTransactions", 10000, "Maximum number of Transactions in Bundle (for PoW)")

	flag.StringP("pidiver.core", "", "../pidiver1.1.rbf", "Core file to upload to FPGA")
	flag.StringP("pidiver.device", "", "/dev/ttyACM0", "Device file for usb communication")
	flag.StringP("pidiver.type", "", "usbdiver", "'pidiver', 'usbdiver'")

}

func declareLogConfigs() {
	flag.String("log.level", "INFO", "DEBUG, INFO, NOTICE, WARNING, ERROR or CRITICAL")
}

func loadAppConfigFile(configPath *string) {
	if len(*configPath) > 0 {
		_, err := os.Stat(*configPath)
		if !flag.CommandLine.Changed("config") && os.IsNotExist(err) {
			// Standard config file not found => skip
			logs.Log.Info("Standard config file not found. Loading default settings.")
		} else {
			logs.Log.Infof("Loading config from: %s", *configPath)
			AppConfig.SetConfigFile(*configPath)
			err := AppConfig.ReadInConfig()
			if err != nil {
				logs.Log.Fatalf("Config could not be loaded from: %s (%s)", *configPath, err)
			}
		}
	}
}
