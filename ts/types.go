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

type Genre uint8
type SubGenre uint8
type SDTRunningState uint8
