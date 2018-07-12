package main

import (
	"C"

	"github.com/iotaledger/giota"
	"github.com/shufps/pidiver/pidiver"
)

var device = "/dev/ttyACM0"
var configFile = "/tmp/pidiver1.0.rbf"
var forceFlash = false
var forceConfigure = false

var config = pidiver.PiDiverConfig{
	Device:         device,
	ConfigFile:     configFile,
	ForceFlash:     forceFlash,
	ForceConfigure: forceConfigure}

var initialized = false

//export ccurl_pow
func ccurl_pow(trytes *C.char, mwm uint) *C.char {
	if !initialized {
		err := pidiver.InitUSBDiver(&config)
		if err != nil {
			println("error initializing usbdiver!")
			return nil
		}
		initialized = true
	}
	goTrytes := C.GoString(trytes)
	//	println(goTrytes)
	//println(C.GoString(trytes))
	nonce, err := pidiver.PowUSBDiver(giota.Trytes(goTrytes), int(mwm))
	if err != nil {
		println("error pow!")
		return nil
	}
	println("Nonce: " + nonce)

	result := goTrytes[0:giota.NonceTrinaryOffset/3] + string(nonce)[0:giota.NonceTrinarySize/3]
	//	println(result)
	return C.CString(result)
}

//export ccurl_pow_finalize
func ccurl_pow_finalize() {
}

//export ccurl_pow_interrupt
func ccurl_pow_interrupt() {
}

func main() {}
