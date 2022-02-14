package ts

type EITShortEventDescriptor struct {
	LangCode  string
	EventName string
	Text      string
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
