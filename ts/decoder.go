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

type frameBuffer struct {
	buf         []byte
	lastCounter uint8
	sectionLen  uint16
}

type Decoder struct {
	tsReader    io.Reader
	lastPat     *PATFrame
	lastPmtMap  map[uint16]*PMTFrame
	selectedSid uint16
	pidBuffer   map[uint16]*frameBuffer
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

func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{reader, nil, make(map[uint16]*PMTFrame), 0, make(map[uint16]*frameBuffer), map[uint16]func([]byte, *Decoder) (Frame, error){0x0: parsePAT, 0x10: parseNIT, 0x12: parseEIT}}
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
				d.pidBuffer[PID] = &frameBuffer{newPayload, counter, sectionLen}
			} else if pidBuf.lastCounter == counter {
				continue
			} else if (pidBuf.lastCounter == 0xf && counter == 0) || pidBuf.lastCounter+1 == counter {
				if isPUSI {
					// insert buf
					payloadOffset := payload[0]
					newPayload := payload[1+payloadOffset:]
					sectionLen := binary.BigEndian.Uint16(newPayload[1:3])&0xfff + 3
					d.pidBuffer[PID] = &frameBuffer{newPayload, counter, sectionLen}

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

func parseNIT(payload []byte, _ *Decoder) (Frame, error) {
	if (payload[0] != 0x40 && payload[0] != 0x41) || payload[1]&0xf0 != 0xf0 {
		return nil, errors.New("illegal NIT frame")
	}

	frame := NITFrame{}
	frame.TransportStreams = make([]NITTransportEntry, 0)
	frame.ServiceList = make(map[uint16]ServiceType)
	frame.NetworkID = binary.BigEndian.Uint16(payload[3:5])
	frame.CurrentNext = payload[5]&1 == 1
	frame.Section = payload[6]
	frame.LastSection = payload[7]
	networkDescLen := binary.BigEndian.Uint16(payload[8:10]) & 0xfff

	// network descriptor
	networkDescSlice := payload[10 : 10+networkDescLen]
	payload = payload[10+networkDescLen:]
	networkDescReader := bytes.NewReader(networkDescSlice)
	for {
		tagID, tagContent, err := extractDescriptor(networkDescReader)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		switch tagID {
		case NetworkNameDescTagID:
			name, err := b24.DecodeString(tagContent)
			if err != nil {
				return nil, err
			}
			frame.NetworkName = name
		case ServiceListDescTagID:
			for len(tagContent) > 0 {
				frame.ServiceList[binary.BigEndian.Uint16(tagContent[0:2])] = ServiceType(tagContent[2])
				tagContent = tagContent[3:]
			}
		default:
			log.Printf("NIT Network tagID: %x, content: %v", tagID, tagContent)
		}
	}
	tsLoopLen := binary.BigEndian.Uint16(payload[0:2]) & 0xfff
	tsPayload := payload[2 : 2+tsLoopLen]
	for len(tsPayload) > 0 {
		entry := NITTransportEntry{}
		entry.TransportStreamId = binary.BigEndian.Uint16(tsPayload[0:2])
		entry.OriginalNetworkId = binary.BigEndian.Uint16(tsPayload[2:4])
		tsDescLen := binary.BigEndian.Uint16(tsPayload[4:6]) & 0xfff
		tsDescSlice := tsPayload[6 : 6+tsDescLen]
		tsPayload = tsPayload[6+tsDescLen:]
		tsDescReader := bytes.NewReader(tsDescSlice)
		tsServiceCount := 0
		for {
			tagID, tagContent, err := extractDescriptor(tsDescReader)
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return nil, err
				}
			}
			switch tagID {
			case NetworkNameDescTagID:
				name, err := b24.DecodeString(tagContent)
				if err != nil {
					return nil, err
				}
				entry.NetworkName = name
			case ServiceDescTagID:
				if tsServiceCount > 0 {
					log.Fatalf("service desc has shown twice")
				}
				tsServiceCount++
				entry.Service = ServiceDescriptor{}
				entry.Service.ServiceType = ServiceType(tagContent[0])
				providerNameLen := tagContent[1]
				entry.Service.ServiceProviderName, err = b24.DecodeString(tagContent[2 : 2+providerNameLen])
				if err != nil {
					return nil, err
				}
				tagContent = tagContent[2+providerNameLen:]
				nameLen := tagContent[0]
				entry.Service.ServiceName, err = b24.DecodeString(tagContent[1 : 1+nameLen])
				if err != nil {
					return nil, err
				}
			default:
				log.Printf("NIT Network tagID: %x, content: %v", tagID, tagContent)
			}
		}
		frame.TransportStreams = append(frame.TransportStreams, entry)
	}
	return &frame, nil
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

func extractDescriptor(descriptorReader io.Reader) (uint8, []byte, error) {
	buf := make([]byte, 2)
	_, err := descriptorReader.Read(buf)
	if err != nil {
		return 0, nil, err
	}
	tagID := buf[0]
	tagLen := buf[1]
	buf = make([]byte, int(tagLen))
	_, err = descriptorReader.Read(buf)
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
