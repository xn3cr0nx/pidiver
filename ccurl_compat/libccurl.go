package main

import (
	"C"

	"github.com/iotaledger/iota.go/trinary"
	"github.com/iotaledger/iota.go/consts"
	"github.com/shufps/pidiver/pidiver"
)

var device = "/dev/ttyACM0"
var configFile = "./pidiver1.1.rbf"
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
    var usb pidiver.USBDiver
    var err error
	if !initialized {
		usb = pidiver.USBDiver{Config: &config}
		err = usb.InitUSBDiver()
		if err != nil {
			println("error initializing usbdiver!")
			return nil
		}
		initialized = true
	}
	goTrytes := C.GoString(trytes)

	nonce, err := usb.PowUSBDiver(trinary.Trytes(goTrytes), int(mwm))
	if err != nil {
		println("error pow!")
		return nil
	}
	println("Nonce: " + nonce)

	result := goTrytes[0:consts.NonceTrinaryOffset/3] + string(nonce)[0:consts.NonceTrinarySize/3]

	return C.CString(result)
}

//export ccurl_pow_finalize
func ccurl_pow_finalize() {
}

//export ccurl_pow_interrupt
func ccurl_pow_interrupt() {
}

func main() {}
