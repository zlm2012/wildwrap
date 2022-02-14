package ts

const (
	PacketLength uint32 = 188

	TsSyncCode          uint8  = 'G'
	PUSI                uint16 = 0x4000
	PIDMask             uint16 = 0x1fff
	AdaptationFieldMask uint8  = 0x20
	PayloadFlagMask     uint8  = 0x10
	CounterMask         uint8  = 0xf

	EITPID uint16 = 0x12

	EITCurrentStreamTID uint8 = 0x4E

	ContentDescTagID uint8 = 0x54
	AudioDescTagID   uint8 = 0xc4

	AnimeGenreIDMask uint8 = 0x70
	TokusatuGenreID  uint8 = 0x72

	AudioCompModeMask       uint8 = 0x1f
	AudioCompDualMonoModeID uint8 = 0b00010
)