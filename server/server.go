package main

import (
	//	"flag"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/iotaledger/iota.go/pow"
	"github.com/muxxer/powsrv/logs"
	"github.com/shufps/pidiver/pidiver"
	"github.com/shufps/pidiver/raspberry"
	"github.com/shufps/pidiver/server/api"

	//	"github.com/shufps/pidiver/orange_pi_pc"
	//	"github.com/shufps/pidiver/raspberry_wiringPi"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const APP_VERSION = "0.1"

var config *viper.Viper

func init() {
	api.Setup()
	config = loadConfig()
	api.SetConfig(config)

	cfg, _ := json.MarshalIndent(config.AllSettings(), "", "  ")
	api.Log.Debugf("Following settings loaded: \n %+v", string(cfg))
}

func main() {
	flag.Parse() // Scan the arguments list

	api.Start(config)

	pconfig := pidiver.PiDiverConfig{
		Device:         config.GetString("pidiver.device"),
		ConfigFile:     config.GetString("pidiver.core"),
		ForceFlash:     false,
		ForceConfigure: false,
		UseCRC:         true,
		UseSharedLock:  true}

	var powFuncs []pow.PowFunc
	var err error

	diver := config.GetString("pidiver.type")

	if diver == "usbdiver" {
		usb := pidiver.USBDiver{Config: &pconfig}
		err = usb.InitUSBDiver()
		powFuncs = append(powFuncs, usb.PowUSBDiver)
	} else if diver == "powchip" {
		usb := pidiver.USBDiver{Config: &pconfig}
		powchip := pidiver.PoWChipDiver{USBDiver: &usb}
		err = powchip.USBDiver.InitUSBDiver()
		powFuncs = append(powFuncs, powchip.PowPoWChipDiver)
	} else if diver == "pidiver" {
		raspi := pidiver.PiDiver{LLStruct: raspberry.GetLowLevel(), Config: &pconfig}
		err = raspi.InitPiDiver()
		powFuncs = append(powFuncs, raspi.PowPiDiver)
	} else {
		log.Fatalf("unknown type %s\n", diver)
	}

	if err != nil {
		logs.Log.Fatal(err)
	}

	api.SetPoWFunc(powFuncs)

	ch := make(chan os.Signal, 10)
	signal.Notify(ch, os.Interrupt)
	signal.Notify(ch, os.Kill)
	for range ch {
		// Clean exit
		logs.Log.Info("PiDiver server is shutting down. Please wait...")
		go func() {
			time.Sleep(time.Duration(5000) * time.Millisecond)
			logs.Log.Info("Bye!")
			os.Exit(0)
		}()
		go api.End()
	}

}

func loadConfig() *viper.Viper {
	// Setup Viper
	var config = viper.New()

	// 1. Set defaults
	config.SetDefault("test", 0)

	// 2. Get command line arguments
	flag.StringP("config", "c", "", "Config path")

	flag.IntP("api.port", "p", 14265, "API Port")
	flag.StringP("api.host", "h", "0.0.0.0", "API Host")
	flag.String("api.auth.username", "", "API Access Username")
	flag.String("api.auth.password", "", "API Access Password")
	flag.Bool("api.debug", false, "Whether to log api access")
	flag.StringSlice("api.limitRemoteAccess", nil, "Limit access to these commands from remote")

	flag.Int("api.pow.maxMinWeightMagnitude", 14, "Maximum Min-Weight-Magnitude (Difficulty for PoW)")
	flag.Int("api.pow.maxTransactions", 10000, "Maximum number of Transactions in Bundle (for PoW)")

	flag.StringP("pidiver.core", "", "../pidiver1.1.rbf", "Core file to upload to FPGA")
	flag.StringP("pidiver.device", "", "/dev/ttyACM0", "Device file for usb communication")
	flag.StringP("pidiver.type", "", "usbdiver", "'pidiver', 'usbdiver'")

	flag.String("log.level", "data", "DEBUG, INFO, NOTICE, WARNING, ERROR or CRITICAL")

	flag.Parse()
	config.BindPFlags(flag.CommandLine)

	// 3. Load config
	var configPath = config.GetString("config")
	if len(configPath) > 0 {
		logs.Log.Infof("Loading config from: %s", configPath)
		config.SetConfigFile(configPath)
		err := config.ReadInConfig()
		if err != nil {
			logs.Log.Fatalf("Config could not be loaded from: %s", configPath)
		}
	}

	return config
}
