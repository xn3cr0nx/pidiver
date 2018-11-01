package raspberry_wiringPi

/*
#cgo CFLAGS: -I./wiringPi/wiringPi
#cgo LDFLAGS: -L./wiringPi/wiringPi -lwiringPi -lstdc++
#include <stdint.h>
#include "wiringPiSPI.h"
#include "wiringPi.h"
	
// wiringPi Numbers
#define	SPI_CS         21
#define	GPIO_nCONFIG   8
#define	GPIO_DATA0     9
#define	GPIO_DCK       7
#define	GPIO_nSTATUS   0
#define	GPIO_CONFDONE  11

uint32_t swap(uint32_t x) {
	return ((x & 0xff000000) >> 24) |
		((x & 0x00ff0000) >> 8) |
		((x & 0x0000ff00) << 8) |
		((x & 0x000000ff) << 24);
}

void send(uint32_t data) {
	uint32_t bytedata = swap(data);
	digitalWrite(SPI_CS, 0);
	wiringPiSPIDataRW(0, (char*) &bytedata, 4);
	digitalWrite(SPI_CS, 1);
}

uint32_t sendBlock(uint32_t* data, int len) {
	for (int i=0;i<len;i++) {
		send(data[i]);
	}
}

uint32_t sendReceive(uint32_t data) {
	uint32_t bytedata = swap(data);
	uint32_t bytedata_read = 0;

	digitalWrite(SPI_CS, 0);
	wiringPiSPIDataRW(0, (char*) &bytedata, 4);
	digitalWrite(SPI_CS, 1);

	digitalWrite(SPI_CS, 0);
	wiringPiSPIDataRW(0, (char*) &bytedata_read, 4);
	digitalWrite(SPI_CS, 1);

	return swap(bytedata_read);
}

*/
import "C"


import (
	"bufio"
	"errors"
	"unsafe"
	"log"
	"os"
	"time"

	"github.com/shufps/pidiver/pidiver"

)

const (
	// wiringPi Numbers
	SPI_CS        = 21
	GPIO_nCONFIG  = 8
	GPIO_DATA0    = 9
	GPIO_DCK      = 7
	GPIO_nSTATUS  = 0
	GPIO_CONFDONE = 11
)

func init() {
}

func GetLowLevel() pidiver.LLStruct {
	return pidiver.LLStruct{LLInit: llInit, LLSPISend: send, LLSPISendBlock: sendBlock, LLSPISendReceive: sendReceive}
}

// send command
func send(data uint32) error {
	C.send(C.uint32_t(data));
	return nil
}

// send block of data for midstate
func sendBlock(data []uint32) error {
	C.sendBlock((*C.uint32_t)(unsafe.Pointer(&data[0])), C.int(len(data)))
	return nil
}

// send and receive
func sendReceive(cmd uint32) (uint32, error) {
	ret := C.sendReceive(C.uint32_t(cmd))
	return uint32(ret), nil
}

func gpioClr(pin int) {
	C.digitalWrite(C.int(pin), C.int(0))
}

func gpioSet(pin int) {
	C.digitalWrite(C.int(pin), C.int(1))
}

func gpioLev(pin int) int {
	return int(C.digitalRead(C.int(pin)))
}

func gpioFsel(pin int, mode int) {
	C.pinMode(C.int(pin), C.int(mode))
}


func initWiringPi() error {
	ret := int(C.wiringPiSetup())
	if ret == -1 {
		return errors.New("error init wiringpi")
	}
	return nil
}

func configureFPGA(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	stats, statsErr := f.Stat()
	if statsErr != nil {
		return err
	}

	size := stats.Size()
	data := make([]byte, size)
	_, err = bufio.NewReader(f).Read(data)
	if err != nil {
		return err
	}

	log.Printf("bytes read: %d\n", size)
	gpioClr(GPIO_DCK)

	/* pulling FPGA_CONFIG to low resets the FPGA */
	gpioClr(GPIO_nCONFIG)	/* FPGA config => low */
	time.Sleep(time.Millisecond * 10) /* give it some time to do its reset stuff */

	for {
		if (gpioLev(GPIO_nSTATUS) != 0) && (gpioLev(GPIO_CONFDONE) != 0) {
			continue
		}
		break
	}

	gpioSet(GPIO_nCONFIG)
	for {
		if gpioLev(GPIO_nSTATUS) == 0 {
			continue
		}
		break
	}

	index := int64(0)
	log.Printf("configuring ...")
	for {
		//              log.Printf("index %d\n", index)
		var value uint8 = data[index]
		index++
		for i := uint8(0); i < 8; i++ {
			if (value>>i)&0x1 != 0 {
				gpioSet(GPIO_DATA0)
			} else {
				gpioClr(GPIO_DATA0)
			}
			gpioSet(GPIO_DCK)
			gpioClr(GPIO_DCK)
		}
		if gpioLev(GPIO_CONFDONE) == 0 && index < size {
			continue
		}
		break
	}
	log.Printf("configure done")

	return nil
}

func llInit(config *pidiver.PiDiverConfig) error {
	err := initWiringPi()
	if err != nil {
		return err
	}

	log.Printf("Using WiringPi")
	// configure pins for configuring
	gpioFsel(GPIO_CONFDONE, 0)

	if config.ForceConfigure || gpioLev(GPIO_CONFDONE) == 0 {
		gpioClr(GPIO_DCK)
		gpioSet(GPIO_nCONFIG)
		gpioFsel(GPIO_DATA0, 1)
		gpioFsel(GPIO_DCK, 1)
		gpioFsel(GPIO_nCONFIG, 1)
		gpioFsel(GPIO_nSTATUS, 0)

		log.Printf("fpga not configured (or force selected ... configuring ...")
		err := configureFPGA(config.ConfigFile)
		if err != nil {
			return err
		}
	}

	if gpioLev(GPIO_CONFDONE) == 0 {
		return errors.New("error configuring!")
	}

	/* init spi interface */
	C.wiringPiSPISetup (C.int(0), C.int(10000000))

	gpioFsel(SPI_CS, 1)

	gpioSet(SPI_CS)
	time.Sleep(10 * time.Millisecond)
	gpioClr(SPI_CS)
	time.Sleep(10 * time.Millisecond)
	gpioSet(SPI_CS)
	time.Sleep(10 * time.Millisecond)

	return nil
}
