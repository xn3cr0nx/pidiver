package multiraspi

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/shufps/bcm2835"
	"github.com/shufps/pidiver/pidiver"
)

const (
	INSTANCES = 4
	SPI_CS    = 5

	BCM2835_SPI_BIT_ORDER_LSBFIRST = 0 ///< LSB First
	BCM2835_SPI_BIT_ORDER_MSBFIRST = 1 ///< MSB First

	BCM2835_SPI_MODE0 = 0 ///< CPOL = 0, CPHA = 0
	BCM2835_SPI_MODE1 = 1 ///< CPOL = 0, CPHA = 1
	BCM2835_SPI_MODE2 = 2 ///< CPOL = 1, CPHA = 0
	BCM2835_SPI_MODE3 = 3 ///< CPOL = 1, CPHA = 1

	BCM2835_SPI_CS0     = 0 ///< Chip Select 0
	BCM2835_SPI_CS1     = 1 ///< Chip Select 1
	BCM2835_SPI_CS2     = 2 ///< Chip Select 2 (ie pins CS1 and CS2 are asserted)
	BCM2835_SPI_CS_NONE = 3 ///< No CS, control it yourself

	BCM2835_SPI_CLOCK_DIVIDER_65536 uint16 = 0     ///< 65536 = 262.144us = 3.814697260kHz
	BCM2835_SPI_CLOCK_DIVIDER_32768 uint16 = 32768 ///< 32768 = 131.072us = 7.629394531kHz
	BCM2835_SPI_CLOCK_DIVIDER_16384 uint16 = 16384 ///< 16384 = 65.536us = 15.25878906kHz
	BCM2835_SPI_CLOCK_DIVIDER_8192  uint16 = 8192  ///< 8192 = 32.768us = 30/51757813kHz
	BCM2835_SPI_CLOCK_DIVIDER_4096  uint16 = 4096  ///< 4096 = 16.384us = 61.03515625kHz
	BCM2835_SPI_CLOCK_DIVIDER_2048  uint16 = 2048  ///< 2048 = 8.192us = 122.0703125kHz
	BCM2835_SPI_CLOCK_DIVIDER_1024  uint16 = 1024  ///< 1024 = 4.096us = 244.140625kHz
	BCM2835_SPI_CLOCK_DIVIDER_512   uint16 = 512   ///< 512 = 2.048us = 488.28125kHz
	BCM2835_SPI_CLOCK_DIVIDER_256   uint16 = 256   ///< 256 = 1.024us = 976.5625MHz
	BCM2835_SPI_CLOCK_DIVIDER_128   uint16 = 128   ///< 128 = 512ns = = 1.953125MHz
	BCM2835_SPI_CLOCK_DIVIDER_64    uint16 = 64    ///< 64 = 256ns = 3.90625MHz
	BCM2835_SPI_CLOCK_DIVIDER_32    uint16 = 32    ///< 32 = 128ns = 7.8125MHz
	BCM2835_SPI_CLOCK_DIVIDER_16    uint16 = 16    ///< 16 = 64ns = 15.625MHz
	BCM2835_SPI_CLOCK_DIVIDER_8     uint16 = 8     ///< 8 = 32ns = 31.25MHz
	BCM2835_SPI_CLOCK_DIVIDER_4     uint16 = 4     ///< 4 = 16ns = 62.5MHz
	BCM2835_SPI_CLOCK_DIVIDER_2     uint16 = 2     ///< 2 = 8ns = 125MHz, fastest you can get
	BCM2835_SPI_CLOCK_DIVIDER_1     uint16 = 1     ///< 0 = 262.144us = 3.814697260kHz, same as 0/65536
)

type Instance struct {
	id    int
	mutex *sync.Mutex
}

var initialized = false

var mutex sync.Mutex

//var instances []Instance

func GetMaxInstances() int {
	return INSTANCES
}

func (i *Instance) send(data uint32) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	var bytedata []byte = make([]byte, 4*INSTANCES)
	binary.BigEndian.PutUint32(bytedata[4*i.id:], data)
	bcm2835.GpioClr(SPI_CS)
	//	log.Printf("sent: % x", bytedata)
	bcm2835.SpiTransfern(bytedata)
	bcm2835.GpioSet(SPI_CS)
	return nil
}

func (i *Instance) sendBlock(data []uint32) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	var bytedata []byte = make([]byte, 4*INSTANCES)
	bcm2835.GpioClr(SPI_CS)
	for j := 0; j < len(data); j++ {
		binary.BigEndian.PutUint32(bytedata[4*i.id:], data[j])
		//	log.Printf("sent: % x", bytedata)
		bcm2835.SpiTransfern(bytedata)
	}
	bcm2835.GpioSet(SPI_CS)
	return nil
}

func (i *Instance) sendReceive(cmd uint32) (uint32, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	bytedata := make([]byte, INSTANCES*4)
	bytedata_read := make([]byte, INSTANCES*4)
	binary.BigEndian.PutUint32(bytedata[i.id*4:], cmd)

	bcm2835.GpioClr(SPI_CS)
	//	log.Printf("sent: % x", bytedata)

	bcm2835.SpiTransfern(bytedata)
	bcm2835.GpioSet(SPI_CS)
	bcm2835.GpioClr(SPI_CS)
	bcm2835.SpiTransfernb(bytedata_read, bytedata_read)
	bcm2835.GpioSet(SPI_CS)
	//	log.Printf("read: % x", bytedata_read)
	return binary.BigEndian.Uint32(bytedata_read[i.id*4:]), nil
}

func GetLowLevel(idx int) pidiver.LLStruct {
	instance := Instance{id: idx, mutex: &mutex}
	//	append(instances, instance)
	return pidiver.LLStruct{LLInit: llInit, LLSPISend: instance.send, LLSPISendBlock: instance.sendBlock, LLSPISendReceive: instance.sendReceive}
}

func llInit(config *pidiver.PiDiverConfig) error {
	if initialized {
		return nil
	}
	err := bcm2835.Init() // Initialize the library
	if err != nil {
		return errors.New("Couldn't initialize BCM2835 Lib")
	}

	/* init spi interface */
	bcm2835.SpiBegin()
	bcm2835.SpiSetBitOrder(BCM2835_SPI_BIT_ORDER_MSBFIRST) /* default */
	bcm2835.SpiSetDataMode(BCM2835_SPI_MODE0)              /* default */
	bcm2835.SpiSetClockDivider(BCM2835_SPI_CLOCK_DIVIDER_32)
	bcm2835.SpiChipSelect(BCM2835_SPI_CS_NONE) /* default */

	bcm2835.GpioFsel(SPI_CS, bcm2835.Output)

	bcm2835.GpioSet(SPI_CS)
	time.Sleep(10 * time.Millisecond)
	bcm2835.GpioClr(SPI_CS)
	time.Sleep(10 * time.Millisecond)
	bcm2835.GpioSet(SPI_CS)
	time.Sleep(10 * time.Millisecond)

	initialized = true
	return nil
}
