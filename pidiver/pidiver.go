package pidiver

import (
	"bufio"
	"encoding/binary"
	"errors"
	"github.com/iotaledger/giota"
	"github.com/shufps/bcm2835"
	"log"
	"os"
	"time"
	"unsafe"
)

var parallel uint32

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
                        if (value >> i) & 0x1 != 0 {
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


func InitPiDiver(config *PiDiverConfig) error {
	err := bcm2835.Init() // Initialize the library
	if err != nil {
		return errors.New("Couldn't initialize BCM2835 Lib")
	}

	// configure pins for configuring
	bcm2835.GpioFsel(GPIO_CONFDONE, bcm2835.Input)

	if config.ForceConfigure || bcm2835.GpioLev(GPIO_CONFDONE) == 1 {
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

	parallel = readParallelLevel()
	log.Printf("Parallel Level Detected: %d\n", parallel)

	// table calculates all bits for AAA -> ZZZ including byte-swap
	tryteMap = make(map[string]uint32)
	for i := 0; i < 27; i++ {
		for j := 0; j < 27; j++ {
			for k := 0; k < 27; k++ {
				key := string(TRYTE_CHARS[i:i+1] + TRYTE_CHARS[j:j+1] + TRYTE_CHARS[k:k+1])
				xtrits := giota.Trytes(key).Trits()
				uint32Data := uint32(0)
				tritslo := uint32(0)
				tritshi := uint32(0)
				for j := 0; j < 9; j++ {
					tmpHi, tmpLo := tritToBits(xtrits[j])
					tritslo |= tmpLo << uint32(j)
					tritshi |= tmpHi << uint32(j)
				}
				uint32Data = swapBytes((tritslo & 0x000001ff) | ((tritshi & 0x000001ff) << 9) | CMD_WRITE_DATA)
				tryteMap[key] = uint32Data
			}
		}
	}
	log.Printf("Table calculated\n")
	return nil
}

// send command
func send(data uint32) {
	var bytedata []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(bytedata, data)
	bcm2835.GpioClr(SPI_CS)
	bcm2835.SpiTransfern(bytedata)
	bcm2835.GpioSet(SPI_CS)
}

// send block of data for midstate
func sendBlock(data []uint32) {
	for i := 0; i < len(data); i++ {
		bytedata := *(*[4]byte)(unsafe.Pointer(&data[i]))
		bcm2835.GpioClr(SPI_CS)
		bcm2835.SpiTransfern(bytedata[:])
		bcm2835.GpioSet(SPI_CS)
	}
}

// send and receive
func sendReceive(cmd uint32) uint32 {
	bytedata := make([]byte, 4)
	bytedata_read := make([]byte, 4)
	binary.BigEndian.PutUint32(bytedata, cmd)

	bcm2835.GpioClr(SPI_CS)
	bcm2835.SpiTransfern(bytedata)
	bcm2835.GpioSet(SPI_CS)
	bcm2835.GpioClr(SPI_CS)
	bcm2835.SpiTransfernb(bytedata_read, bytedata_read)
	bcm2835.GpioSet(SPI_CS)
	return binary.BigEndian.Uint32(bytedata_read)
}

// start PoW
func startPow() {
	send(CMD_WRITE_FLAGS | FLAG_START)
}

// reset write pointer on FPGA
func resetWritePointer() {
	send(CMD_RESET_WRPTR)
}

// not used ... faster version with tables is used
func writeData(tritshi uint32, tritslo uint32) {
	cmd := CMD_WRITE_DATA

	cmd |= tritslo & 0x000001ff
	cmd |= (tritshi & 0x000001ff) << 9

	send(cmd)
}

// read parallel level of FPGA
func readParallelLevel() uint32 {
	return (sendReceive(CMD_READ_FLAGS) & 0x000000f0) >> 4
}

// read binary nonce
func readBinaryNonce() uint32 {
	return sendReceive(CMD_READ_NONCE)
}

// red CRC32
func readCRC32() uint32 {
	return sendReceive(CMD_READ_CRC32)
}

func writeMinWeightMagnitude(bits uint32) {
	if bits > 26 {
		bits = 26
	}
	send(CMD_WRITE_MIN_WEIGHT_MAGNITUDE | ((1 << bits) - 1))
}

// get Mask
func getMask() uint32 {
	return ((sendReceive(CMD_READ_FLAGS) >> 8) & ((1 << parallel) - 1))
}

// get Flags
func getFlags() uint32 {
	return sendReceive(CMD_READ_FLAGS) & 0x0000000f
}

// send trytes for midstate calculation and check for transmission errors
func sendTritData(trytes string) error {
	uint32Data := make([]uint32, HASH_LENGTH/DATA_WIDTH)
	verifyData := make([]uint32, HASH_LENGTH/DATA_WIDTH)
	for tries := 1; ; tries++ {
		for i := 0; i < HASH_LENGTH/DATA_WIDTH; i++ {
			key := trytes[i*3 : i*3+3]
			uint32Data[i] = tryteMap[key]
			verifyData[i] = (uint32Data[i] & 0xffff0300) | (uint32(i)&0x3f)<<10 | (uint32(i)&0xc0)>>6
		}
		resetWritePointer()
		sendBlock(uint32Data)
		verifyBytes := *(*[HASH_LENGTH / DATA_WIDTH * 4]byte)(unsafe.Pointer(&verifyData[0]))

		crc32Verify := crc(verifyBytes[:], len(verifyBytes))
		crc32 := readCRC32()
		//		log.Printf("CRC32: %08x\n", crc32)
		//		log.Printf("CRC32 Verify: %08x\n", crc32Verify)

		if crc32Verify != crc32 {
			log.Printf("Transfer Error (%d/10).\n", tries)
			tries++
			if tries == 11 {
				return errors.New("CRC32 error - giving up ...")
			}
			continue
		} else {
			break
		}
		break
	}
	return nil
}

// send block for midstate calculation
func curlSendBlock(trytes string, doCurl bool) error {
	if err := sendTritData(trytes); err != nil {
		return err
	}
	cmd := CMD_WRITE_FLAGS | FLAG_CURL_WRITE
	if doCurl {
		cmd |= FLAG_CURL_DO_CURL
	}
	send(cmd)

	// instantly read back ... curl needs <1µs on fpga and spi is slower
	if getFlags()&FLAG_CURL_FINISHED == 0 {
		return errors.New("Curl didn't finish")
	}
	return nil
}

// setup fpga for midstate calculation
func curlInitBlock() {
	send(CMD_WRITE_FLAGS | FLAG_CURL_RESET)
}

// do PoW
func PowPiDiver(trytes giota.Trytes, minWeight int) (giota.Trytes, error) {
	// do mid-state-calculation on FPGA
	midStateStart := makeTimestamp()
	curlInitBlock()
	for blocknr := 0; blocknr < 33; blocknr++ {
		doCurl := true
		if blocknr == 32 {
			doCurl = false
		}
		if err := curlSendBlock(string(trytes)[blocknr*(HASH_LENGTH/3):(blocknr+1)*(HASH_LENGTH/3)], doCurl); err != nil {
			return "", err
		}
	}
	midStateEnd := makeTimestamp()

	// write min weight magnitude
	writeMinWeightMagnitude(uint32(minWeight))

	// start PoW
	startPow()

	powStart := makeTimestamp()
	for {
		flags := getFlags()

		if (flags&FLAG_RUNNING) == 0 && ((flags&FLAG_FOUND) != 0 || (flags&FLAG_OVERFLOW) != 0) {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	powEnd := makeTimestamp()

	binary_nonce := readBinaryNonce() - 2 // -2 because of pipelining for speed on FPGA
	mask := getMask()
	log.Printf("Found nonce: %08x (mask: %08x)\n", binary_nonce, mask)
	log.Printf("PoW-Time: %dms\n", (powEnd-powStart)+(midStateEnd-midStateStart))

	return assembleNonce(binary_nonce, mask, parallel)
}
