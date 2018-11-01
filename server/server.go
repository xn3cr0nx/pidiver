package main

import (
	//	"flag"
	"encoding/json"
	"log"
	"math/rand"

	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/curl"
	"github.com/iotaledger/iota.go/pow"
	"github.com/iotaledger/iota.go/trinary"
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
		Device:         config.GetString("usbdiver.device"),
		ConfigFile:     config.GetString("usbdiver.core"),
		ForceFlash:     false,
		ForceConfigure: false,
		UseCRC:         true,
		UseSharedLock:  true}

	var powFuncs []pow.PowFunc
	var err error

	diver := config.GetString("usbdiver.type")

	if diver == "usbdiver" {
		usb := pidiver.USBDiver{Config: &pconfig}
		err = usb.InitUSBDiver()
		powFuncs = append(powFuncs, usb.PowUSBDiver)
		//	} else if *diver == "orange_pi_pc" {
		//		raspi := pidiver.PiDiver{LLStruct: orange_pi_pc.GetLowLevel(), Config: &config}
		//		err = raspi.InitPiDiver()
		//		powFuncs = append(powFuncs, raspi.PowPiDiver)
		//	} else if *diver == "pidiver_wp" {
		//		raspi := pidiver.PiDiver{LLStruct: raspberry_wiringPi.GetLowLevel(), Config: &config}
		//		err = raspi.InitPiDiver()
		//		powFuncs = append(powFuncs, raspi.PowPiDiver)
	} else if diver == "pidiver" {
		raspi := pidiver.PiDiver{LLStruct: raspberry.GetLowLevel(), Config: &pconfig}
		err = raspi.InitPiDiver()
		powFuncs = append(powFuncs, raspi.PowPiDiver)
	} else {
		log.Fatalf("unknown type %s\n", diver)
	}
	if err != nil {
		log.Fatal(err)
	}
	channel := make(chan trinary.Trytes, 100)
	for worker := 0; worker < len(powFuncs); worker++ {
		go func(id int, mwm int, channel chan trinary.Trytes) {
			for {
				trytes, more := <-channel
				if !more {
					break
				}
				//				println(trytes)
				for {
					ret, err := powFuncs[id](trytes, mwm)
					if err != nil {
						//log.Fatalf("Error: %g", err)
						log.Printf("[%d] crc error", id)
						break
						//						continue
					}

					// verify result ... copy nonce to transaction
					trytes = trytes[:consts.NonceTrinaryOffset/3] + ret[0:consts.NonceTrinarySize/3]
					//				println(trytes)
					hash := curl.HashTrytes(trytes)
					tritsHash, _ := trinary.TrytesToTrits(hash)
					for i := 0; i < mwm; i++ {
						if tritsHash[len(tritsHash)-1-i] != 0 {
							log.Fatal("validation error")
							break
						}
					}

					log.Printf("[%d] Nonce-Trytes: %s\n", id, ret)
					log.Printf("[%d] hash: %s\n\n", id, hash)
					/*				if !transaction.HasValidNonce(int64(mwm)) {
									log.Fatal("verify error!")
								}*/
					break
				}
			}
		}(worker, 14, channel)
	}
	// test transaction data
	var tx = "999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999A9RGRKVGWMWMKOLVMDFWJUHNUNYWZTJADGGPZGXNLERLXYWJE9WQHWWBMCPZMVVMJUMWWBLZLNMLDCGDJ999999999999999999999999999999999999999999999999999999YGYQIVD99999999999999999999TXEFLKNPJRBYZPORHZU9CEMFIFVVQBUSTDGSJCZMBTZCDTTJVUFPTCCVHHORPMGCURKTH9VGJIXUQJVHK999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999"

	var rndTag = make([]rune, 128)
	for i := 0; i < 1000; i++ {
		for j := 0; j < 128; j++ {
			rndTag[j] = rune(pidiver.TRYTE_CHARS[rand.Intn(len(pidiver.TRYTE_CHARS))])
		}
		channel <- trinary.Trytes(string(rndTag[0:128]) + tx[128:])
	}
	close(channel)

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
	flag.Bool("api.pow.usePiDiver", false, "Use FPGA (PiDiver) for PoW")

	flag.StringP("usbdiver.core", "", "../pidiver1.1.rbf", "Core file to upload to FPGA")
	flag.StringP("usbdiver.device", "", "/dev/ttyACM0", "Device file for usb communication")
	flag.StringP("usbdiver.type", "", "usbdiver", "'pidiver', 'usbdiver'")

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
