package ts

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
	ServiceList       map[uint16]ServiceType
	TSInfo            TSInfo
}

type NITFrame struct {
	NetworkID        uint16
	Version          uint8
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

type LogoTransmissionDescriptor struct {
	LogoTransmissionType uint8
	LogoId               uint16
	LogoVersion          uint16
	DownloadDataId       uint16
	LogoStr              string
}

type SDTFrameEntry struct {
	ServiceID    uint16
	EITFlags     uint8
	RunningState SDTRunningState
	Scramble     bool
	Service      ServiceDescriptor
	Logo         LogoTransmissionDescriptor
}

type SDTFrame struct {
	TransportStreamID uint16
	Version           uint8
	CurrentNext       bool
	Section           uint8
	LastSection       uint8
	OriginalNetworkID uint16
	Entries           []SDTFrameEntry
}

func (f *SDTFrame) IsParsed() bool {
	return true
}

func (f *SDTFrame) GetType() string {
	return "SDT"
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
