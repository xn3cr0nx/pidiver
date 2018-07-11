package pidiver

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/iotaledger/giota"
	"github.com/lunixbochs/struc"
	"github.com/tarm/goserial"
)

const (
	MAX_DATA_LENGTH = 8192
)

type PiDiverConfig struct {
	Device         string
	ConfigFile     string
	ForceFlash     bool
	ForceConfigure bool
}

type Com struct {
	Id     uint8       `struc:"uint8"`
	Cmd    uint8       `struc:"uint8"`
	Crc8   uint8       `struc:"uint8"`
	Length uint16      `struc:"uint16,little"`
	Data   [8192]uint8 `struc:"[8192]uint8"`
}

type Status struct {
	IsFPGAConfigured uint8 `struc:"uint8"`
}

type Meta struct {
	Timestamp uint64   `struc:"uint64,little"`
	Filename  [32]rune `struc:"[32]uint8"`
	Filesize  uint32   `struc:"uint32,little"`
}

type Page struct {
	Page uint32 `struc:"uint32"`
}

type Trytes struct {
	Data  [891]uint32 `struc:"[891]uint32,little"`
	CRC32 [33]uint32  `struc:"[33]uint32,little"`
	MWM   uint32      `struc:"uint32,little"`
}

type PoWResult struct {
	Nonce    uint32 `struc:"uint32,little"`
	Mask     uint32 `struc:"uint32,little"`
	Parallel uint32 `struc:"uint32,little"`
	Time     uint32 `struc:"uint32,little"`
}

type Version struct {
	Major uint32 `struc:"uint32,little"`
	Minor uint32 `struc:"uint32,little"`
}

var port io.ReadWriteCloser
var id = uint8(0)

func usbRequest(com *Com, timeout int64) (*Com, error) {
	id++
	com.Id = id

	if com.Length > MAX_DATA_LENGTH {
		return &Com{}, errors.New("MAX_DATA_LENGTH exceeded")
	}

	crc := crc8_messagecalc(com.Data[:], int(com.Length))
	com.Crc8 = crc

	var buf bytes.Buffer
	err := struc.Pack(&buf, com)
	if err != nil {
		return &Com{}, errors.New("Error packing struct")
	}
	toWrite := 5 + int(com.Length)
	written, err := port.Write(buf.Bytes()[0:toWrite])
	//	log.Printf("% X\n", buf.Bytes()[0:toWrite])

	if written != toWrite {
		return &Com{}, errors.New("Mismatch of written Bytes and Bytes to write")
	}

	state := STATE_ID
	count := uint16(0)

	t := makeTimestamp()
	for {
		if makeTimestamp()-t > timeout {
			return &Com{}, errors.New("Read Timeout")
		}
		response := make([]byte, 128)
		n, err := port.Read(response)
		/*		if err != nil {
				return &Com{}, errors.New("Error reading USB device")
			}*/
		if err != nil || n == 0 {
			time.Sleep(time.Millisecond * 10)
			continue
		}
		if n == 1 && response[0] == 'X' {
			return &Com{}, errors.New("Protocol Error from USB Device reported")
		}

		for i := 0; i < n; i++ {
			data := uint8(response[i])
			switch state {
			case STATE_ID:
				com.Id = data
				state = STATE_COMMAND
				t = makeTimestamp()
			case STATE_COMMAND:
				com.Cmd = data
				com.Length = 0
				com.Crc8 = 0
				count = 0
				state = STATE_CRC8
			case STATE_CRC8:
				com.Crc8 = data
				state = STATE_LENGTH_LOW
			case STATE_LENGTH_LOW:
				com.Length = uint16(data)
				state = STATE_LENGTH_HIGH
			case STATE_LENGTH_HIGH:
				com.Length |= uint16(data) << 8
				if com.Length > MAX_DATA_LENGTH {
					return &Com{}, errors.New("MAX_DATA_LENGTH exceeded")
				}
				state = STATE_DATA
			case STATE_DATA:
				com.Data[count] = data
				count++
				if count == com.Length {
					if crc8_messagecalc(com.Data[:], int(com.Length)) == com.Crc8 {
						return com, nil
					}
					return &Com{}, errors.New("CRC8 error")
				}
			}
		}
	}

}

func fpgaReadStatus() (Status, error) {
	com := Com{Cmd: CMD_READ_STATUS, Length: 1}
	if _, err := usbRequest(&com, 1000); err != nil {
		return Status{}, err
	}

	var status Status
	if err := struc.Unpack(bytes.NewReader(com.Data[0:com.Length]), &status); err != nil {
		return Status{}, err
	}

	return status, nil
}

func getVersion() (Version, error) {
	com := Com{Cmd: CMD_GET_VERSION, Length: 1}
	if _, err := usbRequest(&com, 1000); err != nil {
		return Version{}, err
	}

	var version Version
	if err := struc.Unpack(bytes.NewReader(com.Data[0:com.Length]), &version); err != nil {
		return Version{}, err
	}

	return version, nil
}

func flashSetPage(page uint32) error {
	com := Com{Cmd: CMD_SET_PAGE, Length: 4}
	com.Data[0] = uint8(page & 0x000000ff)
	com.Data[1] = uint8(page & 0x0000ff00 >> 8)
	com.Data[2] = uint8(page & 0x00ff0000 >> 16)
	com.Data[3] = uint8(page & 0xff000000 >> 24)
	_, err := usbRequest(&com, 1000)
	return err
}

func fpgaConfigure() error {
	com := Com{Cmd: CMD_CONFIGURE_FPGA, Length: 1}
	_, err := usbRequest(&com, 40000)
	return err
}

func fpgaIsConfigured() (bool, error) {
	status, err := fpgaReadStatus()
	if err != nil {
		return false, err
	}
	return (status.IsFPGAConfigured != 0), nil
}

func flashWritePageNumber(page uint32, data []uint8) error {
	err := flashSetPage(page)
	if err != nil {
		return err
	}
	return flashWritePage(data)
}

func flashReadPageNumber(page uint32) ([]uint8, error) {
	err := flashSetPage(page)
	if err != nil {
		return []uint8{}, err
	}
	return flashReadPage()
}

func flashWritePage(data []uint8) error {
	com := Com{Cmd: CMD_WRITE_PAGE, Length: FLASH_SPI_PAGESIZE}
	copy(com.Data[0:], data[0:]) // data can be shorted than FLASH_SPI_PAGESIZE])
	_, err := usbRequest(&com, 1000)
	return err
}

func flashReadPage() ([]uint8, error) {
	com := Com{Cmd: CMD_READ_PAGE, Length: FLASH_SPI_PAGESIZE}
	_, err := usbRequest(&com, 1000)
	if err != nil {
		return []uint8{}, err
	}

	data := make([]uint8, FLASH_SPI_PAGESIZE)
	copy(data[0:], com.Data[0:FLASH_SPI_PAGESIZE])

	return data, nil
}

func flashErase() error {
	com := Com{Cmd: CMD_FLASH_ERASE, Length: 1}
	_, err := usbRequest(&com, 1000)
	return err
}

func flashReadMeta() (Meta, error) {
	data, err := flashReadPageNumber(FLASH_META_PAGE)
	if err != nil {
		return Meta{}, err
	}

	var meta Meta
	if err := struc.Unpack(bytes.NewReader(data), &meta); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

func flashWriteMeta(meta *Meta) error {
	var buf bytes.Buffer
	err := struc.Pack(&buf, meta)
	if err != nil {
		return err
	}
	err = flashWritePageNumber(FLASH_META_PAGE, buf.Bytes())
	return err
}

func fpgaConfigureStart() error {
	com := Com{Cmd: CMD_CONFIGURE_FPGA_START, Length: 1}
	_, err := usbRequest(&com, 1000)
	return err
}

func fpgaConfigureBlock(data []uint8, l uint16) error {
	com := Com{Cmd: CMD_CONFIGURE_FPGA_BLOCK, Length: l}
	copy(com.Data[0:l], data[0:l]) // data can be shorted than FLASH_SPI_PAGESIZE])
	_, err := usbRequest(&com, 1000)
	return err
}

func fpgaConfigureUpload(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	stats, statsErr := f.Stat()
	if statsErr != nil {
		return err
	}

	var size int = int(stats.Size())
	if size > FLASH_SIZE {
		return errors.New("file is too big! >1MB")
	}

	data := make([]byte, size)
	_, err = bufio.NewReader(f).Read(data)
	if err != nil {
		return err
	}

	err = fpgaConfigureStart()
	if err != nil {
		return err
	}

	var toFlash int = size
	var chunk int
	var offset int
	for toFlash > 0 {
		chunk = min(toFlash, 8192)
		log.Printf("configuring %d%%\n", int(float32(offset)/float32(size)*100))
		err = fpgaConfigureBlock(data[offset:offset+chunk], uint16(chunk))

		toFlash -= chunk
		offset += chunk

	}
	return nil
}

func flashUpload(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	stats, statsErr := f.Stat()
	if statsErr != nil {
		return err
	}

	var size uint32 = uint32(stats.Size())
	var originalSize uint32 = size
	if size > FLASH_SIZE {
		return errors.New("file is too big! >1MB")
	}

	// extend to page size
	if size%FLASH_SPI_PAGESIZE != 0 {
		size += FLASH_SPI_PAGESIZE - size%FLASH_SPI_PAGESIZE
	}

	data := make([]byte, size)
	_, err = bufio.NewReader(f).Read(data)
	if err != nil {
		return err
	}
	log.Printf("erasing flash ...")
	err = flashErase()
	if err != nil {
		return err
	}

	log.Printf("flashing configuration ...")
	var numPages uint32 = size / FLASH_SPI_PAGESIZE
	for page := uint32(0); page < numPages; page++ {
		if page%(numPages/100) == 0 {
			log.Printf("%d%% flashed ...\n", int(float32(page)/float32(size/FLASH_SPI_PAGESIZE)*100.0))
		}
		// TODO?
		err := flashWritePageNumber(page, data[page*FLASH_SPI_PAGESIZE:(page+1)*FLASH_SPI_PAGESIZE])
		if err != nil {
			return err
		}
	}

	log.Printf("verifying configuration ...")
	for page := uint32(0); page < size/FLASH_SPI_PAGESIZE; page++ {
		if page%(numPages/100) == 0 {
			log.Printf("%d%% veryfied ...\n", int(float32(page)/float32(size/FLASH_SPI_PAGESIZE)*100.0))
		}
		var read []uint8
		read, err = flashReadPageNumber(page)
		if !bytes.Equal(read[:], data[page*FLASH_SPI_PAGESIZE:(page+1)*FLASH_SPI_PAGESIZE]) {
			return errors.New("verify error at page " + string(page))
		}
	}

	var meta Meta
	meta.Timestamp = uint64(makeTimestamp())
	copy(meta.Filename[0:31], []rune(filename))
	meta.Filesize = originalSize

	log.Printf("flashing meta-page ...")
	err = flashWriteMeta(&meta)
	if err != nil {
		return err
	}

	var verifyMeta Meta
	verifyMeta, err = flashReadMeta()

	if !reflect.DeepEqual(meta, verifyMeta) {
		return errors.New("meta verification failed")
	}
	return nil
}

func loopTest() error {
	com := Com{Cmd: 0xaa, Length: 8192}
	start := makeTimestamp()
	_, err := usbRequest(&com, 1000)
	end := makeTimestamp()
	log.Printf("time %dms\n", (end - start))

	rate := 1.0 / float32(end-start) * 2.0 * 8192.0
	log.Printf("transfer rate: %.6f\n", rate)

	return err
}

func InitUSBDiver(config *PiDiverConfig) error {
	// baud rate has no effect when using USB-CDC
	c0 := &serial.Config{Name: config.Device, Baud: 115200, ReadTimeout: time.Millisecond * 500}

	var err error
	port, err = serial.OpenPort(c0)
	if err != nil {
		log.Fatal(err)
	}

	version, err := getVersion()
	if err != nil {
		return err
	}

	var status Status
	status, err = fpgaReadStatus()
	if err != nil {
		return err
	}

	if version.Major == 1 && version.Minor == 0 {
		// doesn't have flash
		if config.ForceConfigure || status.IsFPGAConfigured == 0 {
			log.Printf("fpga not configured (or configuring forced). configuring ... (10-40sec)")
			err = fpgaConfigureUpload(config.ConfigFile)
			if err != nil {
				log.Fatal("error configuring fpga")
			}

		}
	} else if version.Major == 1 && version.Minor == 1 {
		var meta Meta
		meta, err = flashReadMeta()
		if config.ForceFlash || meta.Timestamp == 0xffffffffffffffff {
			log.Printf("flash is empty (or flashing is forced\n")
			err := flashUpload(config.ConfigFile)
			if err != nil {
				log.Fatalf("error flashing file %s\n", config.ConfigFile)
			} else {
				log.Printf("flashing was successful!")
			}
		} else {
			log.Printf("configuration in flash found:\n")
			log.Printf("Timestamp: %d\n", meta.Timestamp)
			log.Printf("Filename: %s\n", string(meta.Filename[:]))
			log.Printf("Filesize: %d\n", meta.Filesize)
		}

		if config.ForceConfigure || status.IsFPGAConfigured == 0 {
			log.Printf("fpga not configured (or configuring forced). configuring ... (10-40sec)")
			err = fpgaConfigure()
			if err != nil {
				log.Fatal("error configuring fpga")
			}
		}
	}

	status, err = fpgaReadStatus()

	if status.IsFPGAConfigured == 0 {
		log.Fatal("fpga not configured!")
	}
	log.Printf("ready for PoW")

	initTryteMap()

	return nil
}

// do PoW
func PowUSBDiver(trytes giota.Trytes, minWeight int) (giota.Trytes, error) {
	// do mid-state-calculation on FPGA
	//	var start int64 = makeTimestamp()

	com := Com{Cmd: CMD_DO_POW}

	var data Trytes
	for i := 0; i < 891; i++ {
		data.Data[i] = tryteMap[string(trytes[i*3:i*3+3])]
	}
	data.MWM = uint32(minWeight)

	var tmpBuffer bytes.Buffer
	err := struc.Pack(&tmpBuffer, &data)
	if err != nil {
		return giota.Trytes(""), err
	}
	copy(com.Data[0:], tmpBuffer.Bytes())

	com.Length = 3700                // (891 + 33 + 1) * 4
	_, err = usbRequest(&com, 10000) // 10sec enough?
	if err != nil {
		return giota.Trytes(""), err
	}

	var powResult PoWResult
	if err := struc.Unpack(bytes.NewReader(com.Data[0:com.Length]), &powResult); err != nil {
		return giota.Trytes(""), errors.New("error unpack pow results")
	}

	log.Printf("Found nonce: %08x (mask: %08x)\n", powResult.Nonce, powResult.Mask)
	log.Printf("PoW-Time: %dms\n", powResult.Time)

	return assembleNonce(powResult.Nonce, powResult.Mask, powResult.Parallel)
}
