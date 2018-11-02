package main

import (
	//	"flag"

	"log"
	"os"
	"os/signal"
	"time"

	"github.com/iotaledger/iota.go/pow"
	"github.com/shufps/pidiver/pidiver"
	"github.com/shufps/pidiver/raspberry"
	"github.com/shufps/pidiver/server/api"
	"github.com/shufps/pidiver/server/config"
	"github.com/shufps/pidiver/server/logs"

	//	"github.com/shufps/pidiver/orange_pi_pc"
	//	"github.com/shufps/pidiver/raspberry_wiringPi"
	flag "github.com/spf13/pflag"
)

const APP_VERSION = "0.1"

func main() {
	flag.Parse() // Scan the arguments list

	logs.Start()
	config.Start()

	pconfig := pidiver.PiDiverConfig{
		Device:         config.AppConfig.GetString("pidiver.device"),
		ConfigFile:     config.AppConfig.GetString("pidiver.core"),
		ForceFlash:     false,
		ForceConfigure: false,
		UseCRC:         true,
		UseSharedLock:  true}

	var powFuncs []pow.PowFunc
	var err error

	diver := config.AppConfig.GetString("pidiver.type")

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

	api.SetPowFuncs(powFuncs)

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
