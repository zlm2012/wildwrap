package ts

type EITShortEventDescriptor struct {
	LangCode  string
	EventName string
	Text      string
}

type EITExtendedEventEntry struct {
	Name        string
	Description string
}

type EITExtendedEventDescriptor struct {
	LangCode    string
	Entries     []EITExtendedEventEntry
	Description string
}

type EITContentDescriptorEntry struct {
	SubGenre   SubGenre
	UserDefine uint8
}

type EITContentDescriptor struct {
	Entries []EITContentDescriptorEntry
}

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

type ESInfo struct {
	StreamId uint8
	PID      uint16
}

type PMTFrame struct {
	ServiceID   uint16
	Version     uint8
	CurrentNext bool
	Session     uint8
	LastSession uint8
	PcrPID      uint16
	StreamList  []ESInfo
}

func (f *PMTFrame) IsParsed() bool {
	return true
}

func (f *PMTFrame) GetType() string {
	return "PMT"
}

type TSInfo struct {
	RemoteControlKeyId uint8
	TSName             string
}

type NITTransportEntry struct {
	TransportStreamId uint16
	OriginalNetworkId uint16
	NetworkName       string
	Service           ServiceDescriptor
	ServiceList       map[uint16]ServiceType
	TSInfo            TSInfo
}

type NITFrame struct {
	NetworkID        uint16
	CurrentNext      bool
	Section          uint8
	LastSection      uint8
	NetworkName      string
	TransportStreams []NITTransportEntry
}

func (f *NITFrame) IsParsed() bool {
	return true
}

func (f *NITFrame) GetType() string {
	return "NIT"
}

type ServiceDescriptor struct {
	ServiceType         ServiceType
	ServiceProviderName string
	ServiceName         string
}

type Genre uint8
type SubGenre uint8
type SDTRunningState uint8
type ServiceType uint8
