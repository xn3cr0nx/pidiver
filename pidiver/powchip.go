package pidiver

import (
	"bytes"
	"errors"
	"log"

	"github.com/iotaledger/iota.go/trinary"
	"github.com/lunixbochs/struc"
)

type PoWChipDiver struct {
	USBDiver *USBDiver
}

// do PoW
func (u *PoWChipDiver) PowPoWChipDiver(trytes trinary.Trytes, minWeight int, parallelism ...int) (trinary.Trytes, error) {
	// do mid-state-calculation on FPGA
	//	var start int64 = makeTimestamp()

	com := Com{Cmd: CMD_DO_POW}

	var data TrytesData
	for i := 0; i < 891; i++ {
		data.Data[i] = tryteMap[string(trytes[i*3:i*3+3])]
	}
	data.MWM = uint32(minWeight)

	var tmpBuffer bytes.Buffer
	err := struc.Pack(&tmpBuffer, &data)
	if err != nil {
		return trinary.Trytes(""), err
	}
	copy(com.Data[0:], tmpBuffer.Bytes())

	com.Length = 3700                           // (891 + 33 + 1) * 4
	_, err = u.USBDiver.usbRequest(&com, 60000) // 10sec enough?
	if err != nil {
		return trinary.Trytes(""), err
	}

	var powResult PoWResult
	if err := struc.Unpack(bytes.NewReader(com.Data[0:com.Length]), &powResult); err != nil {
		return trinary.Trytes(""), errors.New("error unpack pow results")
	}

	log.Printf("Found nonce: %08x (mask: %08x)\n", powResult.Nonce, powResult.Mask)
	log.Printf("PoW-Time: %dms (%.2fMH/s)\n", powResult.Time, 1.0/(float32(powResult.Time+1)/1000.0)*float32(powResult.Nonce*powResult.Parallel)/1000000.0)

	return u.assembleNonce(powResult.Nonce, powResult.Mask, powResult.Parallel)
}

func (u *PoWChipDiver) assembleNonce(nonce uint32, mask uint32, parallel uint32) (trinary.Trytes, error) {
	// assemble nonce
	bitsLo := make([]uint8, NONCE_TRINARY_SIZE)
	bitsHi := make([]uint8, NONCE_TRINARY_SIZE)

	for i := 0; i < NONCE_TRINARY_SIZE; i++ {
		bitsLo[i] = 0x1
		bitsHi[i] = 0x1
	}

	sigHi := []byte{1, 0, 0, 1, 0, 0, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 1}
	sigLo := []byte{0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 1, 1, 0, 0, 1, 1, 1, 1, 1}

	for i := 0; i < 24; i++ {
		bitsLo[i] = sigLo[i]
		bitsHi[i] = sigHi[i]
	}

	// insert nonce counter
	for i := 0; i < 32; i++ {
		bitsLo[NONCE_TRINARY_SIZE-32+i] = uint8((nonce >> uint32(i)) & 0x1)
		bitsHi[NONCE_TRINARY_SIZE-32+i] = uint8(((^nonce) >> uint32(i)) & 0x1)
	}

	// convert trits to trytes
	nonceTrits := make([]int8, NONCE_TRINARY_SIZE)
	for i := 0; i < NONCE_TRINARY_SIZE; i++ {
		nonceTrits[i] = bitsToTrits(bitsHi[i], bitsLo[i])
	}

	trytes, _ := trinary.TritsToTrytes(nonceTrits)

	return trytes, nil
}
