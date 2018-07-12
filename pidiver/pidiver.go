package pidiver

import (
	"errors"
	"log"
	"time"
	"unsafe"

	"github.com/iotaledger/giota"
)

var parallel uint32

type LLInitFunc func(config *PiDiverConfig) error
type LLSPISendFunc func(data uint32) error
type LLSPISendBlockFunc func(data []uint32) error
type LLSPISendReceiveFunc func(cmd uint32) (uint32, error)

type LLStruct struct {
	LLInit           LLInitFunc
	LLSPISend        LLSPISendFunc
	LLSPISendReceive LLSPISendReceiveFunc
	LLSPISendBlock   LLSPISendBlockFunc
}

var llStruct *LLStruct

func send(data uint32) error {
	return llStruct.LLSPISend(data)
}

// send block of data for midstate
func sendBlock(data []uint32) error {
	return llStruct.LLSPISendBlock(data)
}

// send and receive
func sendReceive(cmd uint32) (uint32, error) {
	return llStruct.LLSPISendReceive(cmd)
}

func InitPiDiver(ll *LLStruct, config *PiDiverConfig) error {
	llStruct = ll

	err := llStruct.LLInit(config)
	if err != nil {
		return err
	}

	parallel, err = readParallelLevel()
	if err != nil {
		return err
	}
	log.Printf("Parallel Level Detected: %d\n", parallel)

	initTryteMap()
	return nil
}

// start PoW
func startPow() error {
	return send(CMD_WRITE_FLAGS | FLAG_START)
}

// reset write pointer on FPGA
func resetWritePointer() error {
	return send(CMD_RESET_WRPTR)
}

// not used ... faster version with tables is used
func writeData(tritshi uint32, tritslo uint32) error {
	cmd := CMD_WRITE_DATA

	cmd |= tritslo & 0x000001ff
	cmd |= (tritshi & 0x000001ff) << 9

	return send(cmd)
}

// read parallel level of FPGA
func readParallelLevel() (uint32, error) {
	val, err := sendReceive(CMD_READ_FLAGS)
	return (val & 0x000000f0) >> 4, err
}

// read binary nonce
func readBinaryNonce() (uint32, error) {
	return sendReceive(CMD_READ_NONCE)
}

// red CRC32
func readCRC32() (uint32, error) {
	return sendReceive(CMD_READ_CRC32)
}

func writeMinWeightMagnitude(bits uint32) {
	if bits > 26 {
		bits = 26
	}
	send(CMD_WRITE_MIN_WEIGHT_MAGNITUDE | ((1 << bits) - 1))
}

// get Mask
func getMask() (uint32, error) {
	val, err := sendReceive(CMD_READ_FLAGS)
	return ((val >> 8) & ((1 << parallel) - 1)), err
}

// get Flags
func getFlags() (uint32, error) {
	val, err := sendReceive(CMD_READ_FLAGS)
	return (val & 0x0000000f), err
}

// send trytes for midstate calculation and check for transmission errors
func sendTritData(trytes string) error {
	uint32Data := make([]uint32, HASH_LENGTH/DATA_WIDTH)
	verifyData := make([]uint32, HASH_LENGTH/DATA_WIDTH)
	for tries := 1; ; tries++ {
		for i := 0; i < HASH_LENGTH/DATA_WIDTH; i++ {
			key := trytes[i*3 : i*3+3]
			uint32Data[i] = tryteMap[key]
			verifyData[i] = (swapBytes(uint32Data[i]) & 0xffff0300) | (uint32(i)&0x3f)<<10 | (uint32(i)&0xc0)>>6
		}
		resetWritePointer()
		sendBlock(uint32Data)
		verifyBytes := *(*[HASH_LENGTH / DATA_WIDTH * 4]byte)(unsafe.Pointer(&verifyData[0]))

		crc32Verify := crc(verifyBytes[:], len(verifyBytes))
		crc32, err := readCRC32()
		if err != nil {
			return err
		}
		//		log.Printf("CRC32: %08x\n", crc32)
		//		log.Printf("CRC32 Verify: %08x\n", crc32Verify)

		if crc32Verify != crc32 {
			log.Printf("Transfer Error (%d/10).\n", tries)
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
	flags, err := getFlags()
	if flags&FLAG_CURL_FINISHED == 0 || err != nil {
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
		flags, err := getFlags()
		if err != nil {
			return giota.Trytes(""), err
		}

		if (flags&FLAG_RUNNING) == 0 && ((flags&FLAG_FOUND) != 0 || (flags&FLAG_OVERFLOW) != 0) {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	powEnd := makeTimestamp()

	binary_nonce, err := readBinaryNonce()
	if err != nil {
		return giota.Trytes(""), err
	}
	binary_nonce -= 2 // -2 because of pipelining for speed on FPGA
	mask, err := getMask()
	log.Printf("Found nonce: %08x (mask: %08x)\n", binary_nonce, mask)
	log.Printf("PoW-Time: %dms\n", (powEnd-powStart)+(midStateEnd-midStateStart))

	return assembleNonce(binary_nonce, mask, parallel)
}
