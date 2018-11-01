package pidiver

import (
	"errors"
	"fmt"
	"log"
	"time"
	"unsafe"

	//	giota "github.com/iotaledger/iota.go/transaction"
	. "github.com/iotaledger/iota.go/trinary"
)

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

type PiDiver struct {
	LLStruct     LLStruct
	Config       *PiDiverConfig
	parallel     uint32
	VersionMajor uint32
	VersionMinor uint32
}

func (p *PiDiver) send(data uint32) error {
	return p.LLStruct.LLSPISend(data)
}

// send block of data for midstate
func (p *PiDiver) sendBlock(data []uint32) error {
	return p.LLStruct.LLSPISendBlock(data)
}

// send and receive
func (p *PiDiver) sendReceive(cmd uint32) (uint32, error) {
	return p.LLStruct.LLSPISendReceive(cmd)
}

func (p *PiDiver) InitPiDiver() error {
	err := p.LLStruct.LLInit(p.Config)
	if err != nil {
		return err
	}

	p.VersionMajor, p.VersionMinor, err = p.readFPGAVersion()
	log.Printf("FPGA version: %d.%d\n", p.VersionMajor, p.VersionMinor)

	p.parallel, err = p.readParallelLevel()
	if err != nil {
		return err
	}
	log.Printf("Parallel Level Detected: %d\n", p.parallel)

	initTryteMap()
	return nil
}

func (p *PiDiver) GetCoreVersion() string {
	return fmt.Sprintf("%v.%v", p.VersionMajor, p.VersionMinor)
}

// start PoW
func (p *PiDiver) startPow() error {
	return p.send(CMD_WRITE_FLAGS | FLAG_START)
}

// reset write pointer on FPGA
func (p *PiDiver) resetWritePointer() error {
	return p.send(CMD_RESET_WRPTR)
}

// not used ... faster version with tables is used
func (p *PiDiver) writeData(tritshi uint32, tritslo uint32) error {
	cmd := CMD_WRITE_DATA

	cmd |= tritslo & 0x000001ff
	cmd |= (tritshi & 0x000001ff) << 9

	return p.send(cmd)
}

// read parallel level of FPGA
func (p *PiDiver) readParallelLevel() (uint32, error) {
	val, err := p.sendReceive(CMD_READ_FLAGS)
	return (val & 0x000000f0) >> 4, err
}

// read binary nonce
func (p *PiDiver) readBinaryNonce() (uint32, error) {
	return p.sendReceive(CMD_READ_NONCE)
}

// red CRC32
func (p *PiDiver) readCRC32() (uint32, error) {
	return p.sendReceive(CMD_READ_CRC32)
}

func (p *PiDiver) writeMinWeightMagnitude(bits uint32) {
	if bits > 26 {
		bits = 26
	}
	p.send(CMD_WRITE_MIN_WEIGHT_MAGNITUDE | ((1 << bits) - 1))
}

// get Mask
func (p *PiDiver) getMask() (uint32, error) {
	val, err := p.sendReceive(CMD_READ_FLAGS)
	return ((val >> 8) & ((1 << p.parallel) - 1)), err
}

// get Flags
func (p *PiDiver) getFlags() (uint32, error) {
	val, err := p.sendReceive(CMD_READ_FLAGS)
	return (val & 0x0000000f), err
}

func (p *PiDiver) unlockReservation() error {
	return p.send(CMD_WRITE_FLAGS | FLAG_RESERVATION_RESET)
}

func (p *PiDiver) readFPGAVersion() (uint32, uint32, error) {
	val, err := p.sendReceive(CMD_READ_FLAGS)
	minor := (val >> 24) & 0xf
	major := (val >> 28) & 0xf
	return major, minor, err
}

func (p *PiDiver) waitForReservation(timeout time.Duration) error {
	start := time.Now()
	for {
		// make reservation
		err := p.send(CMD_WRITE_FLAGS | (FLAG_RESERVATION_PI << FLAG_RESERVATION_WRITE_SHIFT))
		if err != nil {
			return err
		}
		// got device?
		flags, err := p.sendReceive(CMD_READ_FLAGS)
		if err != nil {
			return err
		}
		// if yes, then return
		if flags&FLAG_RESERVATION_READ == (FLAG_RESERVATION_PI << FLAG_RESERVATION_READ_SHIFT) {
			return nil
		}
		// no wait and try again
		if time.Since(start) > timeout {
			return errors.New("couldn't get device reservation")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// send trytes for midstate calculation and check for transmission errors
func (p *PiDiver) sendTritData(trytes string) error {
	uint32Data := make([]uint32, HASH_LENGTH/DATA_WIDTH)
	verifyData := make([]uint32, HASH_LENGTH/DATA_WIDTH)
	for tries := 1; ; tries++ {
		for i := 0; i < HASH_LENGTH/DATA_WIDTH; i++ {
			key := trytes[i*3 : i*3+3]
			uint32Data[i] = tryteMap[key]
			verifyData[i] = (swapBytes(uint32Data[i]) & 0xffff0300) | (uint32(i)&0x3f)<<10 | (uint32(i)&0xc0)>>6
		}
		p.resetWritePointer()
		p.sendBlock(uint32Data)

		if p.Config.UseCRC {
			verifyBytes := *(*[HASH_LENGTH / DATA_WIDTH * 4]byte)(unsafe.Pointer(&verifyData[0]))

			crc32Verify := crc(verifyBytes[:], len(verifyBytes))
			crc32, err := p.readCRC32()
			if err != nil {
				return err
			}
			//		log.Printf("CRC32: %08x\n", crc32)
			//		log.Printf("CRC32 Verify: %08x\n", crc32Verify)

			if crc32Verify != crc32 {
				return errors.New("CRC error")
				/*				log.Printf("Transfer Error (%d/10).\n", tries)
								if tries == 11 {
									return errors.New("CRC32 error - giving up ...")
								}
								continue*/
			} else {
				break
			}
		} else {
			break
		}
	}
	return nil
}

// send block for midstate calculation
func (p *PiDiver) curlSendBlock(trytes string, doCurl bool) error {
	if err := p.sendTritData(trytes); err != nil {
		return err
	}
	cmd := CMD_WRITE_FLAGS | FLAG_CURL_WRITE
	if doCurl {
		cmd |= FLAG_CURL_DO_CURL
	}
	p.send(cmd)

	// instantly read back ... curl needs <1µs on fpga and spi is slower
	flags, err := p.getFlags()
	if flags&FLAG_CURL_FINISHED == 0 || err != nil {
		return errors.New("Curl didn't finish")
	}
	return nil
}

// setup fpga for midstate calculation
func (p *PiDiver) curlInitBlock() {
	p.send(CMD_WRITE_FLAGS | FLAG_CURL_RESET)
}

// do PoW
func (p *PiDiver) PowPiDiver(trytes Trytes, minWeight int) (Trytes, error) {
	// doesn't work on ftdiver because sharing feature doesn't exist
	if p.Config.UseSharedLock && p.VersionMajor == 1 && p.VersionMinor == 1 {
		err := p.waitForReservation(5000 * time.Millisecond)
		if err != nil {
			p.unlockReservation()
			err := p.waitForReservation(5000 * time.Millisecond)
			if err != nil {
				return "", err
			}
		}
		defer p.unlockReservation()
	}

	// do mid-state-calculation on FPGA
	midStateStart := makeTimestamp()
	p.curlInitBlock()
	for blocknr := 0; blocknr < 33; blocknr++ {
		doCurl := true
		if blocknr == 32 {
			doCurl = false
		}
		if err := p.curlSendBlock(string(trytes)[blocknr*(HASH_LENGTH/3):(blocknr+1)*(HASH_LENGTH/3)], doCurl); err != nil {
			return "", err
		}
	}
	midStateEnd := makeTimestamp()

	// write min weight magnitude
	p.writeMinWeightMagnitude(uint32(minWeight))

	// start PoW
	p.startPow()

	powStart := makeTimestamp()
	for {
		flags, err := p.getFlags()
		if err != nil {
			return Trytes(""), err
		}

		if (flags&FLAG_RUNNING) == 0 && ((flags&FLAG_FOUND) != 0 || (flags&FLAG_OVERFLOW) != 0) {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	powEnd := makeTimestamp()

	binary_nonce, err := p.readBinaryNonce()
	if err != nil {
		return Trytes(""), err
	}
	binary_nonce -= 2 // -2 because of pipelining for speed on FPGA
	mask, err := p.getMask()
	log.Printf("Found nonce: %08x (mask: %08x)\n", binary_nonce, mask)
	log.Printf("PoW-Time: %dms\n", (powEnd-powStart)+(midStateEnd-midStateStart))

	return assembleNonce(binary_nonce, mask, p.parallel)
}
