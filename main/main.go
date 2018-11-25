package main

import (
	//	"flag"
	"log"
	"math/rand"

	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/curl"
	"github.com/iotaledger/iota.go/pow"
	"github.com/iotaledger/iota.go/trinary"
	"github.com/shufps/pidiver/pidiver"
	"github.com/shufps/pidiver/raspberry"

	//	"github.com/shufps/pidiver/orange_pi_pc"
	//	"github.com/shufps/pidiver/raspberry_wiringPi"
	flag "github.com/spf13/pflag"
)

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var configFile *string = flag.StringP("fpga.core", "f", "../pidiver1.1.rbf", "Core file to upload to FPGA")
var device *string = flag.StringP("usb.device", "d", "/dev/ttyACM0", "Device file for usb communication")

//var diver *string = flag.StringP("pow.type", "t", "usbdiver", "'pidiver', 'usbdiver', 'pidiver_wp")
//var diver *string = flag.StringP("pow.type", "t", "usbdiver", "'pidiver', 'usbdiver', 'orange_pi_pc")
var diver *string = flag.StringP("pow.type", "t", "usbdiver", "'pidiver', 'usbdiver', 'powchip'")

func main() {
	flag.Parse() // Scan the arguments list

	config := pidiver.PiDiverConfig{
		Device:         *device,
		ConfigFile:     *configFile,
		ForceFlash:     false,
		ForceConfigure: false,
		UseCRC:         true,
		UseSharedLock:  true}

	var powFuncs []pow.ProofOfWorkFunc
	var err error
	if *diver == "usbdiver" {
		usb := pidiver.USBDiver{Config: &config}
		err = usb.InitUSBDiver()
		powFuncs = append(powFuncs, usb.PowUSBDiver)
	} else if *diver == "powchip" {
		usb := pidiver.USBDiver{Config: &config}
		powchip := pidiver.PoWChipDiver{USBDiver: &usb}
		err = powchip.USBDiver.InitUSBDiver()
		powFuncs = append(powFuncs, powchip.PowPoWChipDiver)
		//	} else if *diver == "orange_pi_pc" {
		//		raspi := pidiver.PiDiver{LLStruct: orange_pi_pc.GetLowLevel(), Config: &config}
		//		err = raspi.InitPiDiver()
		//		powFuncs = append(powFuncs, raspi.PowPiDiver)
		//	} else if *diver == "pidiver_wp" {
		//		raspi := pidiver.PiDiver{LLStruct: raspberry_wiringPi.GetLowLevel(), Config: &config}
		//		err = raspi.InitPiDiver()
		//		powFuncs = append(powFuncs, raspi.PowPiDiver)
	} else if *diver == "pidiver" {
		raspi := pidiver.PiDiver{LLStruct: raspberry.GetLowLevel(), Config: &config}
		err = raspi.InitPiDiver()
		powFuncs = append(powFuncs, raspi.PowPiDiver)
	} else {
		log.Fatalf("unknown type %s\n", *diver)
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
					hash, _ := curl.HashTrytes(trytes)
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
