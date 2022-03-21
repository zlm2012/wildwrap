package ts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/zlm2012/wildwrap/b24"
	"io"
	"log"
	"math"
	"time"
)

type PATFrame struct {
	TransportStreamID []byte
	Version           uint8
	CurrentNext       bool
	Section           uint8
	LastSection       uint8

	NetworkPID uint16
	SidPidMap  map[uint16]uint16
}

func (f *PATFrame) IsParsed() bool {
	return true
}

func (f *PATFrame) GetType() string {
	return "PAT"
}

type PMTFrame struct {
	PcrPID        uint16
	StreamPidList []uint16
}

func (f *PMTFrame) IsParsed() bool {
	return true
}

func (f *PMTFrame) GetType() string {
	return "PMT"
}

type FrameBuffer struct {
	buf         []byte
	lastCounter uint8
	sectionLen  uint16
}

type Decoder struct {
	tsReader    io.Reader
	lastPat     *PATFrame
	lastPmtMap  map[uint16]*PMTFrame
	selectedSid uint16
	pidBuffer   map[uint16]*FrameBuffer
	pidToParse  map[uint16]func([]byte, *Decoder) (Frame, error)
}

type Frame interface {
	IsParsed() bool
	GetType() string
}

type GeneralFrame struct {
	RawData []byte
}

func (f *GeneralFrame) IsParsed() bool {
	return false
}

func (f *GeneralFrame) GetType() string {
	return "generic"
}

type EITFrame struct {
	TableID uint8
	Entries []EITFrameEntry
}

type EITFrameEntry struct {
	EventID            uint16
	StartTime          time.Time
	Duration           time.Duration
	RunningState       SDTRunningState
	FreeCA             bool
	DualMono           bool
	Contents           EITContentDescriptor
	ShortDescriptor    EITShortEventDescriptor
	ExtendedDescriptor []EITExtendedEventDescriptor
}

func (f *EITFrame) IsParsed() bool {
	return true
}

func (f *EITFrame) GetType() string {
	return "EIT"
}

func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{reader, nil, make(map[uint16]*PMTFrame), 0, make(map[uint16]*FrameBuffer), map[uint16]func([]byte, *Decoder) (Frame, error){0x0: parsePAT, 0x12: parseEIT}}
}

func (d *Decoder) ParseNext() (Frame, error) {
	var PID uint16
	isPUSI := false
	for {
		buf, err := d.readNextTSPacket()
		if err != nil {
			return nil, err
		}
		FlagPIDCombo := binary.BigEndian.Uint16(buf[1:3])
		isPUSI = PUSI&FlagPIDCombo == PUSI
		PID = FlagPIDCombo & PIDMask
		if _, ok := d.pidToParse[PID]; !ok {
			continue
		}
		counter := buf[3] & CounterMask
		// has payload
		if PayloadFlagMask&buf[3] == PayloadFlagMask {
			pidBuf, pidBufOk := d.pidBuffer[PID]
			if !pidBufOk && !isPUSI {
				// ignore
				continue
			}
			payload := getPayload(buf)
			if !pidBufOk && isPUSI {
				payloadOffset := payload[0]
				newPayload := payload[1+payloadOffset:]
				sectionLen := binary.BigEndian.Uint16(newPayload[1:3])&0xfff + 3
				d.pidBuffer[PID] = &FrameBuffer{newPayload, counter, sectionLen}
			} else if pidBuf.lastCounter == counter {
				continue
			} else if (pidBuf.lastCounter == 0xf && counter == 0) || pidBuf.lastCounter+1 == counter {
				if isPUSI {
					// insert buf
					payloadOffset := payload[0]
					newPayload := payload[1+payloadOffset:]
					sectionLen := binary.BigEndian.Uint16(newPayload[1:3])&0xfff + 3
					d.pidBuffer[PID] = &FrameBuffer{newPayload, counter, sectionLen}

					// Parse
					parseFunc, _ := d.pidToParse[PID]
					if payloadOffset != 0 {
						pidBuf.buf = append(pidBuf.buf, payload[1:1+payloadOffset]...)
					}
					return parseFunc(pidBuf.buf[0:pidBuf.sectionLen], d)
				} else {
					pidBuf.lastCounter = counter
					pidBuf.buf = append(pidBuf.buf, payload...)
				}
			} else {
				log.Printf("counter is not in continuity for PID %d", PID)
				// drop buffer unable to be parsed
				delete(d.pidBuffer, PID)
			}
		}
	}
}

func parsePAT(payload []byte, d *Decoder) (Frame, error) {
	if payload[0] != 0 || payload[1]&0xf0 != 0b10110000 {
		return nil, errors.New("illegal PAT frame")
	}

	frame := PATFrame{}
	frame.TransportStreamID = payload[3:5]
	frame.Version = payload[5] & 0b00111110 >> 1
	frame.CurrentNext = payload[5]&1 == 1
	frame.Section = payload[6]
	frame.LastSection = payload[7]
	frame.SidPidMap = make(map[uint16]uint16)
	payload = payload[8 : len(payload)-4]
	if len(payload)%4 != 0 {
		log.Fatalf("unexpected payload len after crc excluded: %d", len(payload))
	}
	for len(payload) != 0 {
		current := payload[0:4]
		payload = payload[4:]
		progNum := binary.BigEndian.Uint16(current[0:2])
		progPid := binary.BigEndian.Uint16(current[2:]) & 0x1fff
		if progNum == 0 {
			frame.NetworkPID = progPid
		} else {
			frame.SidPidMap[progNum] = progPid
		}
	}
	if frame.CurrentNext {
		d.lastPat = &frame
	}
	return &frame, nil
}

func parseMjd(raw []byte) time.Time {
	if raw[0] == 0xff && raw[1] == 0xff && raw[2] == 0xff && raw[3] == 0xff && raw[4] == 0xff {
		return time.UnixMicro(0) // N/A
	}
	mjd := binary.BigEndian.Uint16(raw[0:2])
	yp := int((float64(mjd) - 15078.2) / 365.25)
	mp := int((float64(mjd) - 14956.1 - math.Floor(float64(yp)*365.25)) / 30.6001)
	d := int(mjd) - 14956 - int(math.Floor(float64(yp)*365.25)) - int(math.Floor(float64(mp)*30.6001))
	k := 0
	if mp == 14 || mp == 15 {
		k = 1
	}
	y := yp + k
	m := mp - 1 - k*12
	h := raw[2]
	min := raw[3]
	s := raw[4]
	loc, _ := time.LoadLocation("Asia/Tokyo")
	return time.Date(y+1900, time.Month(m), d, int(h), int(min), int(s), 0, loc)
}

func parseDuration(raw []byte) time.Duration {
	return time.Duration(raw[0])*time.Hour + time.Duration(raw[1])*time.Minute + time.Duration(raw[2])*time.Second
}

func parseEITEntry(entryPayload []byte) (*EITFrameEntry, int, error) {
	entry := EITFrameEntry{}
	entry.EventID = binary.BigEndian.Uint16(entryPayload[0:2])
	entry.StartTime = parseMjd(entryPayload[2:7])
	entry.Duration = parseDuration(entryPayload[7:10])
	entry.RunningState = SDTRunningState(entryPayload[10] >> 5)
	entry.FreeCA = entryPayload[10]&0x10 == 0x10
	entryLen := int(binary.BigEndian.Uint16(entryPayload[10:12]) & 0xfff)
	fromFirstEntry := entryPayload[12 : 12+entryLen]
	entryReader := bytes.NewReader(fromFirstEntry)
	for {
		tagID, tagContent, err := decodeEventEntry(entryReader)
		if err != nil {
			if err == io.EOF {
				return &entry, 12 + entryLen, nil
			} else {
				return nil, 12 + entryLen, err
			}
		}
		switch tagID {
		case ShortEventDescTagID:
			entry.ShortDescriptor.LangCode = string(tagContent[0:3])
			entry.ShortDescriptor.EventName, err = b24.DecodeString(tagContent[4 : 4+tagContent[3]])
			if err != nil {
				return nil, 12 + entryLen, err
			}
			entry.ShortDescriptor.Text, err = b24.DecodeString(tagContent[4+tagContent[3]+1:])
			if err != nil {
				return nil, 12 + entryLen, err
			}
		case ContentDescTagID:
			genreCount := len(tagContent) / 2
			entries := make([]EITContentDescriptorEntry, genreCount)
			for i := 0; i < genreCount; i++ {
				entries[i] = EITContentDescriptorEntry{SubGenre(tagContent[2*i]), tagContent[2*i+1]}
			}
		case AudioDescTagID:
			if tagContent[1]&AudioCompModeMask == AudioCompDualMonoModeID {
				entry.DualMono = true
			}
		case ExtendedEventDescTagID:
			extDesc := EITExtendedEventDescriptor{}
			extDesc.LangCode = string(tagContent[1:4])
			itemsLen := tagContent[4]
			itemRaw := tagContent[5 : 5+itemsLen]
			extDesc.Entries = make([]EITExtendedEventEntry, 0)
			for len(itemRaw) != 0 {
				nameLen := itemRaw[0]
				name, err := b24.DecodeString(itemRaw[1 : 1+nameLen])
				if err != nil {
					return nil, 12 + entryLen, err
				}
				descLen := itemRaw[1+nameLen]
				desc, err := b24.DecodeString(itemRaw[2+nameLen : 2+nameLen+descLen])
				if err != nil {
					return nil, 12 + entryLen, err
				}
				extDesc.Entries = append(extDesc.Entries, EITExtendedEventEntry{name, desc})
				itemRaw = itemRaw[2+nameLen+descLen:]
			}
			descLen := tagContent[5+itemsLen]
			extDesc.Description, err = b24.DecodeString(tagContent[6+itemsLen : 6+itemsLen+descLen])
			if err != nil {
				return nil, 12 + entryLen, err
			}
			if err != nil {
				return nil, 12 + entryLen, err
			}
			if entry.ExtendedDescriptor == nil {
				entry.ExtendedDescriptor = []EITExtendedEventDescriptor{extDesc}
			} else {
				entry.ExtendedDescriptor = append(entry.ExtendedDescriptor, extDesc)
			}
		default:
			log.Printf("tag %d %v", tagID, tagContent)
		}
	}
}

func parseEIT(entryPayload []byte, _ *Decoder) (Frame, error) {
	if entryPayload[1]&0xf0 != 0xf0 {
		return nil, errors.New("illegal EIT frame")
	}
	eitFrame := EITFrame{}
	eitFrame.TableID = entryPayload[0]
	eitFrame.Entries = make([]EITFrameEntry, 0)
	remaining := entryPayload[14 : len(entryPayload)-4]
	for len(remaining) > 0 {
		entry, parsedLen, err := parseEITEntry(remaining)
		if err != nil {
			return nil, err
		}
		eitFrame.Entries = append(eitFrame.Entries, *entry)
		remaining = remaining[parsedLen:]
	}
	return &eitFrame, nil
}

func decodeEventEntry(entryReader io.Reader) (uint8, []byte, error) {
	buf := make([]byte, 2)
	_, err := entryReader.Read(buf)
	if err != nil {
		return 0, nil, err
	}
	tagID := buf[0]
	tagLen := buf[1]
	buf = make([]byte, int(tagLen))
	_, err = entryReader.Read(buf)
	return tagID, buf, err
}

func (d *Decoder) SeekNextEITFrame(PID uint16, TID uint8) ([]byte, error) {
	buf, isPUSI, err := d.SeekNextPacket(PID, true)
	var fullPayload []byte
	for {
		if err != nil {
			return nil, err
		}
		if !isPUSI {
			return nil, errors.New("PUSI should be set")
		}
		fullPayload = getPayload(buf)
		payloadOffset := fullPayload[0]
		fullPayload = fullPayload[1+payloadOffset:]
		if TID == fullPayload[0] {
			break
		}
		buf, isPUSI, err = d.SeekNextPacket(PID, true)
	}
	if fullPayload[1]&0xf0 != 0xf0 {
		return nil, errors.New("illegal EIT frame")
	}
	sectionLen := int(binary.BigEndian.Uint16(fullPayload[1:3])&0xfff) + 3
	lastCounter := buf[3] & CounterMask

	// read full frame
	for {
		buf, isPUSI, err = d.SeekNextPacket(PID, false)
		counter := buf[3] & CounterMask
		if counter == lastCounter {
			continue
		} else if (lastCounter == 0xf && counter == 0) || lastCounter+1 == counter {
			payload := getPayload(buf)
			if isPUSI {
				nextPayloadOffset := payload[0]
				if nextPayloadOffset == 0 {
					break
				}
				payload = payload[1 : 1+nextPayloadOffset]
			}
			fullPayload = append(fullPayload, payload...)
			if isPUSI && len(fullPayload) < sectionLen {
				return nil, errors.New("payload length is not enough but next frame is coming")
			}
			if len(fullPayload) >= sectionLen || isPUSI {
				break
			}
			lastCounter = counter
		} else {
			return nil, errors.New("counter is not in continuity")
		}
	}
	return fullPayload[14 : sectionLen-4], nil
}

func getPayload(packet []byte) []byte {
	hasAdaptationField := AdaptationFieldMask&packet[3] == AdaptationFieldMask
	adaptationLen := 0
	if hasAdaptationField {
		adaptationLen = 1 + int(packet[4])
	}
	return packet[4+adaptationLen:]
}

// SeekNextPacket only for tracing a specified PID, ignoring packet with adaptation field set
func (d *Decoder) SeekNextPacket(expectedPID uint16, needPUSI bool) ([]byte, bool, error) {
	buf, err := d.readNextTSPacket()
	var PID uint16
	isPUSI := false
	for {
		if err != nil {
			return nil, false, err
		}
		FlagPIDCombo := binary.BigEndian.Uint16(buf[1:3])
		isPUSI = PUSI&FlagPIDCombo == PUSI
		// has payload
		if PayloadFlagMask&buf[3] == PayloadFlagMask {
			// PUSI set if needPUSI == true
			if !needPUSI || isPUSI {
				PID = FlagPIDCombo & PIDMask
				if PID == expectedPID {
					break
				}
			}
		}
		buf, err = d.readNextTSPacket()
	}
	return buf, isPUSI, err
}

func (d *Decoder) readNextTSPacket() ([]byte, error) {
	buf := make([]byte, PacketLength)
	_, err := d.tsReader.Read(buf)
	if err != nil {
		return nil, err
	}
	if buf[0] != TsSyncCode {
		return nil, errors.New("no valid TS sync code")
	}
	return buf, nil
}
