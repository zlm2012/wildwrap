package ts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/zlm2012/wildwrap/b24"
	"io"
	"log"
	"time"
)

type EITFrame struct {
	TableID           uint8
	ServiceID         uint16
	TSID              uint16
	OriginalNetworkID uint16
	Entries           []EITFrameEntry
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
		tagID, tagContent, err := extractDescriptor(entryReader)
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
			entry.Contents = EITContentDescriptor{entries}
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
