package orange_pi_pc

/*
#cgo CFLAGS: -I./WiringOP/wiringPi/wiringPi 
#cgo LDFLAGS: -L./WiringOP/wiringPi -lwiringPi -lstdc++
#include <stdint.h>
#include "wiringPiSPI.h"
#include "wiringPi.h"
#include <unistd.h>
#include <stdio.h>
#include <sys/time.h>
#include <time.h>
#include <stdlib.h>

// wiringPi Numbers
//#define SPI_CS         21
#define GPIO_nCONFIG   8
#define GPIO_DATA0     9
#define GPIO_DCK       7
#define GPIO_nSTATUS   0
#define GPIO_CONFDONE  11

uint32_t swap(uint32_t x) {
        return ((x & 0xff000000) >> 24) |
                ((x & 0x00ff0000) >> 8) |
                ((x & 0x0000ff00) << 8) |
                ((x & 0x000000ff) << 24);
} 

void send(uint32_t data) {                                                                                                                                                                                                       
        uint32_t bytedata = swap(data);                                                                                                                                                                                          
//        digitalWrite(SPI_CS, 0);                                                                                                                                                                                                 
	wiringPiSPIDataRW(1, (char*) &bytedata, 4);
//        digitalWrite(SPI_CS, 1);                                                                                                                                                                                                 
//	printf("sent: %08x\n", data);
}                                                                                                                                                                                                                                
                                                                                                                                                                                                                                 
                                                                                                                                                                                                                                 
uint32_t sendReceive(uint32_t data) {                                                                                                                                                                                            
        uint32_t bytedata = swap(data);                                                                                                                                                                                          
        uint32_t bytedata_read = 0x00000000;                                                                                                                                                                                             
//        digitalWrite(SPI_CS, 0);
	wiringPiSPIDataRW(1, (char*) &bytedata, 4);
//        digitalWrite(SPI_CS, 1);

//        digitalWrite(SPI_CS, 0);
	wiringPiSPIDataRW(1, (char*) &bytedata_read, 4);
//        digitalWrite(SPI_CS, 1);

//	printf("sent: %08x received %08x\n", data, swap(bytedata_read));
        return swap(bytedata_read);
}

uint32_t sendBlock(uint32_t* data, int len) {                                                                                                                                                                                    
        for (int i=0;i<len;i++) {                                                                                                                                                                                                
                send(data[i]);                                                                                                                                                                                                   
        }                                                                                                                                                                                                                        
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
//	"encoding/binary"
        "github.com/shufps/pidiver/pidiver"

)

const (
        // wiringPi Numbers
//        SPI_CS        = 21
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

/*
// send command
func send(data uint32) error {
	var bytedata []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(bytedata, data)
//	gpioClr(SPI_CS)
	C.wiringPiSPIDataRW(1, (*C.uchar)(unsafe.Pointer(&bytedata[0])),4)
//	gpioSet(SPI_CS)
	return nil
}

// send block of data for midstate
func sendBlock(data []uint32) error {
	for i := 0; i < len(data); i++ {
		send(data[i])
	}
	return nil
}

// send and receive
func sendReceive(cmd uint32) (uint32, error) {
	bytedata := make([]byte, 4)
	bytedata_read := make([]byte, 4)
	binary.BigEndian.PutUint32(bytedata, cmd)

//	gpioClr(SPI_CS)
	C.wiringPiSPIDataRW(1, (*C.uchar)(unsafe.Pointer(&bytedata[0])),4)
//	gpioSet(SPI_CS)
//	gpioClr(SPI_CS)
	C.wiringPiSPIDataRW(1, (*C.uchar)(unsafe.Pointer(&bytedata_read[0])),4)
//	gpioSet(SPI_CS)
	return binary.BigEndian.Uint32(bytedata_read), nil
}
*/



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
        gpioClr(GPIO_nCONFIG)   /* FPGA config => low */
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
        C.wiringPiSPISetup (C.int(1), C.int(10000000))
/*
        gpioFsel(SPI_CS, 1)

        gpioSet(SPI_CS)
        time.Sleep(10 * time.Millisecond)
        gpioClr(SPI_CS)
        time.Sleep(10 * time.Millisecond)
        gpioSet(SPI_CS)
        time.Sleep(10 * time.Millisecond)
*/
        return nil
}

