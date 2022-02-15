package ts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/zlm2012/wildwrap/b24"
	"io"
	"log"
)

type Decoder struct {
	tsReader io.Reader
}

type EITFrame struct {
	DualMono        bool
	Contents        EITContentDescriptor
	ShortDescriptor EITShortEventDescriptor
}

func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{reader}
}

func (d *Decoder) ReadNextCurrentStreamEITFrame() (*EITFrame, error) {
	entryPayload, err := d.SeekNextEITFrame(EITPID, EITCurrentStreamTID)
	if err != nil {
		return nil, err
	}

	firstEntryLen := int(binary.BigEndian.Uint16(entryPayload[10:12]) & 0xfff)
	fromFirstEntry := entryPayload[12 : 12+firstEntryLen]
	entryReader := bytes.NewReader(fromFirstEntry)
	genreRead := false
	audioRead := false
	eitFrame := EITFrame{}
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
		if tagID == ShortEventDescTagID {
			eitFrame.ShortDescriptor.LangCode = string(tagContent[0:3])
			eitFrame.ShortDescriptor.EventName, err = b24.DecodeString(tagContent[4 : 4+tagContent[3]])
			if err != nil {
				return nil, err
			}
			eitFrame.ShortDescriptor.Text, err = b24.DecodeString(tagContent[4+tagContent[3]+1:])
			if err != nil {
				return nil, err
			}
		}
		if tagID == ContentDescTagID {
			genreRead = true
			genreCount := len(tagContent) / 2
			entries := make([]EITContentDescriptorEntry, genreCount)
			for i := 0; i < genreCount; i++ {
				entries[i] = EITContentDescriptorEntry{SubGenre(tagContent[2*i]), tagContent[2*i+1]}
			}
		}
		if tagID == AudioDescTagID {
			audioRead = true
			if tagContent[1]&AudioCompModeMask == AudioCompDualMonoModeID {
				eitFrame.DualMono = true
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
