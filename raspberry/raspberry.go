package raspberry

import (
	"bufio"
	"encoding/binary"
	"errors"
	"log"
	"os"
	"time"

	"github.com/shufps/pidiver/pidiver"

	"github.com/shufps/bcm2835"
)

const (
	SPI_CS        = 5
	GPIO_nCONFIG  = 2
	GPIO_DATA0    = 3
	GPIO_DCK      = 4
	GPIO_nSTATUS  = 17
	GPIO_CONFDONE = 7

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

func init() {
}

func GetLowLevel() pidiver.LLStruct {
	return pidiver.LLStruct{LLInit: llInit, LLSPISend: send, LLSPISendBlock: sendBlock, LLSPISendReceive: sendReceive}
}

// send command
func send(data uint32) error {
	var bytedata []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(bytedata, data)
	bcm2835.GpioClr(SPI_CS)
	bcm2835.SpiTransfern(bytedata)
	bcm2835.GpioSet(SPI_CS)
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

	bcm2835.GpioClr(SPI_CS)
	bcm2835.SpiTransfern(bytedata)
	bcm2835.GpioSet(SPI_CS)
	bcm2835.GpioClr(SPI_CS)
	bcm2835.SpiTransfernb(bytedata_read, bytedata_read)
	bcm2835.GpioSet(SPI_CS)
	return binary.BigEndian.Uint32(bytedata_read), nil
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
	bcm2835.GpioClr(GPIO_DCK)

	/* pulling FPGA_CONFIG to low resets the FPGA */
	bcm2835.GpioClr(GPIO_nCONFIG)     /* FPGA config => low */
	time.Sleep(time.Millisecond * 10) /* give it some time to do its reset stuff */

	for {
		if (bcm2835.GpioLev(GPIO_nSTATUS) != 0) && (bcm2835.GpioLev(GPIO_CONFDONE) != 0) {
			continue
		}
		break
	}

	bcm2835.GpioSet(GPIO_nCONFIG)
	for {
		if bcm2835.GpioLev(GPIO_nSTATUS) == 0 {
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
				bcm2835.GpioSet(GPIO_DATA0)
			} else {
				bcm2835.GpioClr(GPIO_DATA0)
			}
			bcm2835.GpioSet(GPIO_DCK)
			bcm2835.GpioClr(GPIO_DCK)
		}
		if bcm2835.GpioLev(GPIO_CONFDONE) == 0 && index < size {
			continue
		}
		break
	}
	log.Printf("configure done")

	return nil
}

func llInit(config *pidiver.PiDiverConfig) error {
	err := bcm2835.Init() // Initialize the library
	if err != nil {
		return errors.New("Couldn't initialize BCM2835 Lib")
	}

	// configure pins for configuring
	bcm2835.GpioFsel(GPIO_CONFDONE, bcm2835.Input)

	if config.ForceConfigure || bcm2835.GpioLev(GPIO_CONFDONE) == 0 {
		bcm2835.GpioClr(GPIO_DCK)
		bcm2835.GpioSet(GPIO_nCONFIG)
		bcm2835.GpioFsel(GPIO_DATA0, bcm2835.Output)
		bcm2835.GpioFsel(GPIO_DCK, bcm2835.Output)
		bcm2835.GpioFsel(GPIO_nCONFIG, bcm2835.Output)
		bcm2835.GpioFsel(GPIO_nSTATUS, bcm2835.Input)

		log.Printf("fpga not configured (or force selected ... configuring ...")
		err := configureFPGA(config.ConfigFile)
		if err != nil {
			return err
		}
	}

	if bcm2835.GpioLev(GPIO_CONFDONE) == 0 {
		return errors.New("error configuring!")
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

	return nil
}
