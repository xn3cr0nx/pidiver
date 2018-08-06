package pidiver

import (
	"errors"
	"time"

	"github.com/iotaledger/giota"
)

const (
	NONCE_TRINARY_SIZE = 81
	STATE_LENGTH       = 729
	HASH_LENGTH        = 243
	TRYTE_CHARS        = "9ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	DATA_WIDTH         = 9

	// USB communication
	// state machine
	STATE_ID          = uint8(0x00)
	STATE_COMMAND     = uint8(0x01)
	STATE_CRC8        = uint8(0x02)
	STATE_LENGTH_LOW  = uint8(0x03)
	STATE_LENGTH_HIGH = uint8(0x04)
	STATE_DATA        = uint8(0x05)
	STATE_EXECUTE     = uint8(0x06)
	STATE_ERROR       = uint8(0xff)

	// usb commands
	CMD_GET_VERSION    = uint8(0x01)
	CMD_FLASH_ERASE    = uint8(0x10)
	CMD_SET_PAGE       = uint8(0x11)
	CMD_WRITE_PAGE     = uint8(0x12)
	CMD_READ_PAGE      = uint8(0x13)
	CMD_CONFIGURE_FPGA = uint8(0x14)
	CMD_READ_STATUS    = uint8(0x15)

	CMD_CONFIGURE_FPGA_BLOCK = uint8(0x16)
	CMD_CONFIGURE_FPGA_START = uint8(0x17)

	CMD_DO_POW = uint8(0x20)

	FLASH_SIZE         = (1024 * 1024) // 8MBit SPI Flash
	FLASH_META_PAGE    = ((FLASH_SIZE / FLASH_SPI_PAGESIZE) - 1)
	FLASH_SPI_PAGESIZE = 0x100

	FLAG_RUNNING       uint32 = (1 << 0)
	FLAG_FOUND         uint32 = (1 << 1)
	FLAG_OVERFLOW      uint32 = (1 << 2)
	FLAG_CURL_FINISHED uint32 = (1 << 3)

	FLAG_START        uint32 = (1 << 0)
	FLAG_CURL_RESET   uint32 = (1 << 1)
	FLAG_CURL_WRITE   uint32 = (1 << 2)
	FLAG_CURL_DO_CURL uint32 = (1 << 3)

	FLAG_RESERVATION_RESET       uint32 = (1 << 25)
	FLAG_RESERVATION_WRITE       uint32 = (1 << 24) | (1 << 23)
	FLAG_RESERVATION_WRITE_SHIFT        = 23
	FLAG_RESERVATION_PI                 = 0x1
	FLAG_RESERVATION_USB                = 0x2

	FLAG_RESERVATION_READ       uint32 = (1 << 23) | (1 << 22)
	FLAG_RESERVATION_READ_SHIFT        = 22


	CMD_NOP                        uint32 = 0x00000000
	CMD_WRITE_FLAGS                uint32 = 0x04000000
	CMD_RESET_WRPTR                uint32 = 0x08000000
	CMD_WRITE_DATA                 uint32 = 0x10000000
	CMD_WRITE_MIN_WEIGHT_MAGNITUDE uint32 = 0x20000000
	CMD_READ_FLAGS                 uint32 = 0x84000000
	CMD_READ_NONCE                 uint32 = 0x88000000
	CMD_READ_CRC32                 uint32 = 0x90000000

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

type PiDiverConfig struct {
	Device         string
	ConfigFile     string
	ForceFlash     bool
	ForceConfigure bool
	UseCRC         bool
	UseSharedLock  bool	// pidiver/usbdiver sharing lock
}

var crctab = []uint32{
	0x00000000,
	0x04C11DB7, 0x09823B6E, 0x0D4326D9, 0x130476DC, 0x17C56B6B,
	0x1A864DB2, 0x1E475005, 0x2608EDB8, 0x22C9F00F, 0x2F8AD6D6,
	0x2B4BCB61, 0x350C9B64, 0x31CD86D3, 0x3C8EA00A, 0x384FBDBD,
	0x4C11DB70, 0x48D0C6C7, 0x4593E01E, 0x4152FDA9, 0x5F15ADAC,
	0x5BD4B01B, 0x569796C2, 0x52568B75, 0x6A1936C8, 0x6ED82B7F,
	0x639B0DA6, 0x675A1011, 0x791D4014, 0x7DDC5DA3, 0x709F7B7A,
	0x745E66CD, 0x9823B6E0, 0x9CE2AB57, 0x91A18D8E, 0x95609039,
	0x8B27C03C, 0x8FE6DD8B, 0x82A5FB52, 0x8664E6E5, 0xBE2B5B58,
	0xBAEA46EF, 0xB7A96036, 0xB3687D81, 0xAD2F2D84, 0xA9EE3033,
	0xA4AD16EA, 0xA06C0B5D, 0xD4326D90, 0xD0F37027, 0xDDB056FE,
	0xD9714B49, 0xC7361B4C, 0xC3F706FB, 0xCEB42022, 0xCA753D95,
	0xF23A8028, 0xF6FB9D9F, 0xFBB8BB46, 0xFF79A6F1, 0xE13EF6F4,
	0xE5FFEB43, 0xE8BCCD9A, 0xEC7DD02D, 0x34867077, 0x30476DC0,
	0x3D044B19, 0x39C556AE, 0x278206AB, 0x23431B1C, 0x2E003DC5,
	0x2AC12072, 0x128E9DCF, 0x164F8078, 0x1B0CA6A1, 0x1FCDBB16,
	0x018AEB13, 0x054BF6A4, 0x0808D07D, 0x0CC9CDCA, 0x7897AB07,
	0x7C56B6B0, 0x71159069, 0x75D48DDE, 0x6B93DDDB, 0x6F52C06C,
	0x6211E6B5, 0x66D0FB02, 0x5E9F46BF, 0x5A5E5B08, 0x571D7DD1,
	0x53DC6066, 0x4D9B3063, 0x495A2DD4, 0x44190B0D, 0x40D816BA,
	0xACA5C697, 0xA864DB20, 0xA527FDF9, 0xA1E6E04E, 0xBFA1B04B,
	0xBB60ADFC, 0xB6238B25, 0xB2E29692, 0x8AAD2B2F, 0x8E6C3698,
	0x832F1041, 0x87EE0DF6, 0x99A95DF3, 0x9D684044, 0x902B669D,
	0x94EA7B2A, 0xE0B41DE7, 0xE4750050, 0xE9362689, 0xEDF73B3E,
	0xF3B06B3B, 0xF771768C, 0xFA325055, 0xFEF34DE2, 0xC6BCF05F,
	0xC27DEDE8, 0xCF3ECB31, 0xCBFFD686, 0xD5B88683, 0xD1799B34,
	0xDC3ABDED, 0xD8FBA05A, 0x690CE0EE, 0x6DCDFD59, 0x608EDB80,
	0x644FC637, 0x7A089632, 0x7EC98B85, 0x738AAD5C, 0x774BB0EB,
	0x4F040D56, 0x4BC510E1, 0x46863638, 0x42472B8F, 0x5C007B8A,
	0x58C1663D, 0x558240E4, 0x51435D53, 0x251D3B9E, 0x21DC2629,
	0x2C9F00F0, 0x285E1D47, 0x36194D42, 0x32D850F5, 0x3F9B762C,
	0x3B5A6B9B, 0x0315D626, 0x07D4CB91, 0x0A97ED48, 0x0E56F0FF,
	0x1011A0FA, 0x14D0BD4D, 0x19939B94, 0x1D528623, 0xF12F560E,
	0xF5EE4BB9, 0xF8AD6D60, 0xFC6C70D7, 0xE22B20D2, 0xE6EA3D65,
	0xEBA91BBC, 0xEF68060B, 0xD727BBB6, 0xD3E6A601, 0xDEA580D8,
	0xDA649D6F, 0xC423CD6A, 0xC0E2D0DD, 0xCDA1F604, 0xC960EBB3,
	0xBD3E8D7E, 0xB9FF90C9, 0xB4BCB610, 0xB07DABA7, 0xAE3AFBA2,
	0xAAFBE615, 0xA7B8C0CC, 0xA379DD7B, 0x9B3660C6, 0x9FF77D71,
	0x92B45BA8, 0x9675461F, 0x8832161A, 0x8CF30BAD, 0x81B02D74,
	0x857130C3, 0x5D8A9099, 0x594B8D2E, 0x5408ABF7, 0x50C9B640,
	0x4E8EE645, 0x4A4FFBF2, 0x470CDD2B, 0x43CDC09C, 0x7B827D21,
	0x7F436096, 0x7200464F, 0x76C15BF8, 0x68860BFD, 0x6C47164A,
	0x61043093, 0x65C52D24, 0x119B4BE9, 0x155A565E, 0x18197087,
	0x1CD86D30, 0x029F3D35, 0x065E2082, 0x0B1D065B, 0x0FDC1BEC,
	0x3793A651, 0x3352BBE6, 0x3E119D3F, 0x3AD08088, 0x2497D08D,
	0x2056CD3A, 0x2D15EBE3, 0x29D4F654, 0xC5A92679, 0xC1683BCE,
	0xCC2B1D17, 0xC8EA00A0, 0xD6AD50A5, 0xD26C4D12, 0xDF2F6BCB,
	0xDBEE767C, 0xE3A1CBC1, 0xE760D676, 0xEA23F0AF, 0xEEE2ED18,
	0xF0A5BD1D, 0xF464A0AA, 0xF9278673, 0xFDE69BC4, 0x89B8FD09,
	0x8D79E0BE, 0x803AC667, 0x84FBDBD0, 0x9ABC8BD5, 0x9E7D9662,
	0x933EB0BB, 0x97FFAD0C, 0xAFB010B1, 0xAB710D06, 0xA6322BDF,
	0xA2F33668, 0xBCB4666D, 0xB8757BDA, 0xB5365D03, 0xB1F740B4}

var tryteMap map[string]uint32

// wtf ...^^
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// swap bytes
func swapBytes(data uint32) uint32 {
	return ((data & 0xff000000) >> 24) |
		((data & 0x00ff0000) >> 8) |
		((data & 0x0000ff00) << 8) |
		((data & 0x000000ff) << 24)
}

// convert bits to trit
func bitsToTrits(h uint8, l uint8) int8 {
	if h != 0 && l != 0 {
		return 0
	}
	if h != 0 && l == 0 {
		return 1
	}
	if h == 0 && l != 0 {
		return -1
	}
	return -128
}

// convert trit to bits
func tritToBits(trit int8) (uint32, uint32) {
	switch trit {
	case 0:
		return 0x1, 0x1
	case -1:
		return 0x0, 0x1
	case 1:
		return 0x1, 0x0
	default:
		return 0x0, 0x0
	}
}

// calculate CRC32-MPEG
func crc(bytes []byte, l int) uint32 {
	value := uint32(0xffffffff)
	for i := 0; i < l; i++ {
		value = (value << 8) ^ crctab[((value>>24)^uint32(bytes[i]))&0xff]
	}
	return value
}

func crc8_bytecalc(reg uint8, byte uint8) uint8 {
	var flag uint8 = 0x00
	var polynom uint8 = 0xd5 // Generatorpolynom

	// gehe fuer jedes Bit der Nachricht durch
	for i := 0; i < 8; i++ {
		if reg&0x80 != 0x00 {
			flag = 1
		} else {
			flag = 0 // Teste MSB des Registers
		}
		reg <<= 1 // Schiebe Register 1 Bit nach Links und
		if byte&0x80 != 0x00 {
			reg |= 1 // FÃžlle das LSB mit dem naechsten Bit der Nachricht auf
		}
		byte <<= 1 // nÃĪchstes Bit der Nachricht
		if flag != 0x00 {
			reg ^= polynom // falls flag==1, dann XOR mit Polynom
		}
	}
	return reg
}

func crc8_messagecalc(msg []uint8, l int) uint8 {
	var reg uint8 = 0x00
	for i := 0; i < l; i++ {
		reg = crc8_bytecalc(reg, msg[i]) // Berechne fuer jeweils 8 Bit der Nachricht
	}
	return crc8_bytecalc(reg, 0) // die Berechnung muss um die Bitlaenge des Polynoms mit 0-Wert fortgefuehrt werden
}

func initTryteMap() {
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
				uint32Data = (tritslo & 0x000001ff) | ((tritshi & 0x000001ff) << 9) | CMD_WRITE_DATA
				tryteMap[key] = uint32Data
			}
		}
	}
}

func assembleNonce(nonce uint32, mask uint32, parallel uint32) (giota.Trytes, error) {
	if parallel == 0 || parallel > 8 {
		return giota.Trytes(""), errors.New("wrong parallel level read")
	}

	if mask == 0 {
		return giota.Trytes(""), errors.New("zero-mask returned")
	}

	// log2(parallel)
	log2 := 0
	for i := 0; i < 32; i++ {
		if ((parallel - 1) & (1 << uint32(i))) != 0 {
			log2 = i
		}
	}

	if mask == 0 {
		return "", errors.New("Returned Mask Zero")
	}

	// find set bit in mask
	found_bit := uint32(0)
	for i := uint32(0); i < parallel; i++ {
		if mask&(1<<i) != 0 {
			found_bit = i
			break
		}
	}

	// assemble nonce
	bitsLo := make([]uint8, NONCE_TRINARY_SIZE)
	bitsHi := make([]uint8, NONCE_TRINARY_SIZE)

	for i := 0; i < NONCE_TRINARY_SIZE; i++ {
		bitsLo[i] = 0x1
		bitsHi[i] = 0x1
	}

	// insert PIDIVER at beginning of nonce
	sigLo := []byte{0, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1}
	sigHi := []byte{1, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1, 0, 1, 1, 1}

	for i := 0; i < 24; i++ {
		bitsLo[i] = sigLo[i]
		bitsHi[i] = sigHi[i]
	}

	// insert initial nonce trits bit thingies
	for j := 0; j <= log2; j++ {
		bitsLo[j+24] = uint8((found_bit >> uint32(j)) & 0x1)
		bitsHi[j+24] = uint8((^(found_bit >> uint32(j))) & 0x1)
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

	return giota.Trits(nonceTrits).Trytes(), nil
}
