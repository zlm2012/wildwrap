package ts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/zlm2012/wildwrap/b24"
	"io"
	"log"
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
	TableID            uint8
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
	buf, err := d.readNextTSPacket()
	var PID uint16
	isPUSI := false
	for {
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
				buf, err = d.readNextTSPacket()
				continue
			}
			payload := getPayload(buf)
			if !pidBufOk && isPUSI {
				payloadOffset := payload[0]
				sectionLen := binary.BigEndian.Uint16(payload[2+payloadOffset:4+payloadOffset]) & 0xfff
				d.pidBuffer[PID] = &FrameBuffer{payload[1+payloadOffset:], counter, sectionLen}
			} else if pidBuf.lastCounter == counter {
				continue
			} else if (pidBuf.lastCounter == 0xf && counter == 0) || pidBuf.lastCounter+1 == counter {
				if isPUSI {
					// insert buf
					payloadOffset := payload[0]
					sectionLen := binary.BigEndian.Uint16(payload[2+payloadOffset:4+payloadOffset]) & 0xfff
					d.pidBuffer[PID] = &FrameBuffer{payload[1+payloadOffset:], counter, sectionLen}

					// Parse
					parseFunc, _ := d.pidToParse[PID]
					pidBuf.buf = append(pidBuf.buf, payload[1:1+payloadOffset]...)
					if len(pidBuf.buf) != int(pidBuf.sectionLen)+3 {
						log.Printf("pid buf len is not equal to section len: %d vs %d", len(pidBuf.buf), pidBuf.sectionLen+3)
					}
					return parseFunc(pidBuf.buf, d)
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
		buf, err = d.readNextTSPacket()
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

func (d *Decoder) ReadNextCurrentStreamEITFrame() (*EITFrame, error) {
	entryPayload, err := d.SeekNextEITFrame(EITPID, EITCurrentStreamTID)
	if err != nil {
		return nil, err
	}
	f, e := parseEIT(entryPayload, nil)
	return f.(*EITFrame), e
}

func parseEIT(entryPayload []byte, _ *Decoder) (Frame, error) {
	if entryPayload[1]&0xf0 != 0xf0 {
		return nil, errors.New("illegal EIT frame")
	}
	firstEntryLen := int(binary.BigEndian.Uint16(entryPayload[10:12]) & 0xfff)
	fromFirstEntry := entryPayload[12 : 12+firstEntryLen]
	entryReader := bytes.NewReader(fromFirstEntry)
	genreRead := false
	audioRead := false
	eitFrame := EITFrame{}
	eitFrame.TableID = entryPayload[0]
	for {
		tagID, tagContent, err := decodeEventEntry(entryReader)
		log.Println("EIT Descriptor", tagID, tagContent)
		if err != nil {
			if err == io.EOF {
				return nil, nil
			} else {
				return nil, err
			}
		}
		switch tagID {
		case ShortEventDescTagID:
			eitFrame.ShortDescriptor.LangCode = string(tagContent[0:3])
			eitFrame.ShortDescriptor.EventName, err = b24.DecodeString(tagContent[4 : 4+tagContent[3]])
			if err != nil {
				return nil, err
			}
			eitFrame.ShortDescriptor.Text, err = b24.DecodeString(tagContent[4+tagContent[3]+1:])
			if err != nil {
				return nil, err
			}
		case ContentDescTagID:
			genreRead = true
			genreCount := len(tagContent) / 2
			entries := make([]EITContentDescriptorEntry, genreCount)
			for i := 0; i < genreCount; i++ {
				entries[i] = EITContentDescriptorEntry{SubGenre(tagContent[2*i]), tagContent[2*i+1]}
			}
		case AudioDescTagID:
			audioRead = true
			if tagContent[1]&AudioCompModeMask == AudioCompDualMonoModeID {
				eitFrame.DualMono = true
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
					return nil, err
				}
				descLen := itemRaw[1+nameLen]
				desc, err := b24.DecodeString(itemRaw[2+nameLen : 2+nameLen+descLen])
				if err != nil {
					return nil, err
				}
				extDesc.Entries = append(extDesc.Entries, EITExtendedEventEntry{name, desc})
				itemRaw = itemRaw[2+nameLen+descLen:]
			}
			descLen := tagContent[5+itemsLen]
			extDesc.Description, err = b24.DecodeString(tagContent[6+itemsLen : 6+itemsLen+descLen])
			if err != nil {
				return nil, err
			}
			if err != nil {
				return nil, err
			}
			if eitFrame.ExtendedDescriptor == nil {
				eitFrame.ExtendedDescriptor = []EITExtendedEventDescriptor{extDesc}
			} else {
				eitFrame.ExtendedDescriptor = append(eitFrame.ExtendedDescriptor, extDesc)
			}
		}

		if genreRead && audioRead {
			return &eitFrame, nil
		}
	}
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
