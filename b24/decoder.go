package b24

import (
	"errors"
	"fmt"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"strings"
)

type b24GCharset uint8

const (
	b24CSNone b24GCharset = iota
	b24CSIgnore1Byte
)

const (
	b24CSHiragana b24GCharset = iota + 0x30
	b24CSKatakana
)

const (
	b24CSProAlphabet b24GCharset = iota + 0x36
	b24CSProHiragana
	b24CSProKatakana
	b24CSX0213Plane1
	b24CSX0213Plane2
	b24CSGaiji
)

const (
	b24CSKanji         b24GCharset = 0x42
	b24CSAlphabet      b24GCharset = 0x4a
	b24CSX0201Katakana b24GCharset = 0x49
	b24CSDRCS0         b24GCharset = 0xfe // not supported, 2 bytes
	b24CSDRCS          b24GCharset = 0xff // not supported, 1 byte
)

var (
	b24HiraganaSet = []string{" ", "ぁ", "あ", "ぃ", "い", "ぅ", "う", "ぇ", "え", "ぉ", "お", "か", "が", "き", "ぎ", "く",
		"ぐ", "け", "げ", "こ", "ご", "さ", "ざ", "し", "じ", "す", "ず", "せ", "ぜ", "そ", "ぞ", "た",
		"だ", "ち", "ぢ", "っ", "つ", "づ", "て", "で", "と", "ど", "な", "に", "ぬ", "ね", "の", "は",
		"ば", "ぱ", "ひ", "び", "ぴ", "ふ", "ぶ", "ぷ", "へ", "べ", "ぺ", "ほ", "ぼ", "ぽ", "ま", "み",
		"む", "め", "も", "ゃ", "や", "ゅ", "ゆ", "ょ", "よ", "ら", "り", "る", "れ", "ろ", "ゎ", "わ",
		"ゐ", "ゑ", "を", "ん", "", "", "", "ゝ", "ゞ", "ー", "。", "「", "」", "、", "・", ""}
	b24KatakanaSet = []string{"", "ァ", "ア", "ィ", "イ", "ゥ", "ウ", "ェ", "エ", "ォ", "オ", "カ", "ガ", "キ", "ギ", "ク",
		"グ", "ケ", "ゲ", "コ", "ゴ", "サ", "ザ", "シ", "ジ", "ス", "ズ", "セ", "ゼ", "ソ", "ゾ", "タ",
		"ダ", "チ", "ヂ", "ッ", "ツ", "ヅ", "テ", "デ", "ト", "ド", "ナ", "ニ", "ヌ", "ネ", "ノ", "ハ",
		"バ", "パ", "ヒ", "ビ", "ピ", "フ", "ブ", "プ", "ヘ", "ベ", "ペ", "ホ", "ボ", "ポ", "マ", "ミ",
		"ム", "メ", "モ", "ャ", "ヤ", "ュ", "ユ", "ョ", "ヨ", "ラ", "リ", "ル", "レ", "ロ", "ヮ", "ワ",
		"ヰ", "ヱ", "ヲ", "ン", "バ", "ヵ", "ヶ", "ヽ", "ヾ", "ー", "。", "「", "」", "、", "・", ""}
)

type decodeState struct {
	G0          b24GCharset
	G1          b24GCharset
	G2          b24GCharset
	G3          b24GCharset
	GL          b24GCharset
	GR          b24GCharset
	GLSSVictim  b24GCharset
	SJISDecoder *encoding.Decoder
}

func newDecodeState() *decodeState {
	return &decodeState{b24CSKanji, b24CSAlphabet, b24CSHiragana, b24CSKatakana, b24CSKanji, b24CSHiragana, b24CSNone, japanese.ShiftJIS.NewDecoder()}
}

func getCharset1Byte(bytes []byte) (b24GCharset, int, error) {
	switch bytes[0] {
	case 0x20:
		// 1-byte DRCS
		return b24CSDRCS, 1, nil
	case uint8(b24CSProAlphabet):
		fallthrough
	case uint8(b24CSAlphabet):
		return b24CSAlphabet, 0, nil
	case uint8(b24CSProHiragana):
		fallthrough
	case uint8(b24CSHiragana):
		return b24CSHiragana, 0, nil
	case uint8(b24CSProKatakana):
		fallthrough
	case uint8(b24CSKatakana):
		return b24CSKatakana, 0, nil
	case uint8(b24CSX0201Katakana):
		return b24CSX0201Katakana, 0, nil
	default:
		if 0x32 <= bytes[0] && bytes[0] <= 0x35 {
			// mosaic, ignore
			return b24CSIgnore1Byte, 0, nil
		} else {
			return b24CSNone, 0, errors.New(fmt.Sprintf("unexpected g set %02x", bytes[0]))
		}
	}
}

func getCharset2Bytes(bytes []byte, onlyCheckDRCS bool) (b24GCharset, int, error) {
	additionalSeqLen := 0
	if onlyCheckDRCS && bytes[0] != 0x20 {
		return b24CSNone, 0, errors.New(fmt.Sprintf("onlyCheckDRCS is set but control byte is %02x", bytes[0]))
	}
	switch bytes[0] {
	case uint8(b24CSKanji):
		return b24CSKanji, additionalSeqLen, nil
	case uint8(b24CSX0213Plane1):
		return b24CSX0213Plane1, additionalSeqLen, nil
	case uint8(b24CSX0213Plane2):
		return b24CSX0213Plane2, additionalSeqLen, nil
	case uint8(b24CSGaiji):
		return b24CSGaiji, additionalSeqLen, nil
	case 0x20:
		if bytes[1] == 0x40 {
			additionalSeqLen = 1
			return b24CSDRCS0, additionalSeqLen, nil
		} else {
			return b24CSNone, 0, errors.New(fmt.Sprintf("not supported drcs0 g0 sequence: %02x", bytes[0:2]))
		}
	default:
		return b24CSNone, 0, errors.New(fmt.Sprintf("not supported 2-byte encoding switch sequence: %02x", bytes[0:2]))
	}
}

func (s *decodeState) changeState(bytes []byte) (int, bool, error) {
	seqLen := 0
	ss := false
	switch bytes[0] {
	case 0x1B:
		if len(bytes) == 1 { // ignore
			return 0, false, nil
		}
		seqLen = 1
		switch bytes[1] {
		case 0x24:
			// 2-byte G0-3 set
			seqLen++
			switch bytes[2] {
			case 0x28:
				bcs, additionalSeqLen, err := getCharset2Bytes(bytes[3:], true)
				if err != nil {
					return 0, false, err
				}
				seqLen += additionalSeqLen + 1
				s.G0 = bcs
			case 0x29:
				bcs, additionalSeqLen, err := getCharset2Bytes(bytes[3:], false)
				if err != nil {
					return 0, false, err
				}
				seqLen += additionalSeqLen + 1
				s.G1 = bcs
			case 0x2A:
				bcs, additionalSeqLen, err := getCharset2Bytes(bytes[3:], false)
				if err != nil {
					return 0, false, err
				}
				seqLen += additionalSeqLen + 1
				s.G2 = bcs
			case 0x2B:
				bcs, additionalSeqLen, err := getCharset2Bytes(bytes[3:], false)
				if err != nil {
					return 0, false, err
				}
				seqLen += additionalSeqLen + 1
				s.G3 = bcs
			default:
				bcs, additionalSeqLen, err := getCharset2Bytes(bytes[2:], false)
				if err != nil {
					return 0, false, err
				}
				if bcs == b24CSDRCS0 {
					return 0, false, errors.New(fmt.Sprintf("unexpected g0 drcs0 seq: %02x", bytes[0:4]))
				}
				seqLen += additionalSeqLen
				s.G0 = bcs
			}
		case 0x28:
			// G0 1-byte set
			if len(bytes) == 1 { // ignore
				return 0, false, nil
			}
			bcs, additionalSeqLen, err := getCharset1Byte(bytes[2:])
			if err != nil {
				return 0, false, err
			}
			seqLen += additionalSeqLen + 1
			s.G0 = bcs
		case 0x29:
			// G1 1-byte set
			if len(bytes) == 1 { // ignore
				return 0, false, nil
			}
			bcs, additionalSeqLen, err := getCharset1Byte(bytes[2:])
			if err != nil {
				return 0, false, err
			}
			seqLen += additionalSeqLen + 1
			s.G1 = bcs
		case 0x2A:
			// G2 1-byte set
			if len(bytes) == 1 { // ignore
				return 0, false, nil
			}
			bcs, additionalSeqLen, err := getCharset1Byte(bytes[2:])
			if err != nil {
				return 0, false, err
			}
			seqLen += additionalSeqLen + 1
			s.G2 = bcs
		case 0x2B:
			// G3 1-byte set
			if len(bytes) == 1 { // ignore
				return 0, false, nil
			}
			bcs, additionalSeqLen, err := getCharset1Byte(bytes[2:])
			if err != nil {
				return 0, false, err
			}
			seqLen += additionalSeqLen + 1
			s.G3 = bcs
		case 0x6E:
			// LS2
			s.GL = s.G2
		case 0x6F:
			// LS3
			s.GL = s.G3
		case 0x7E:
			// LS1R
			s.GR = s.G1
		case 0x7D:
			// LS2R
			s.GR = s.G2
		case 0x7C:
			// LS3R
			s.GR = s.G3
		default:
			return 0, false, errors.New(fmt.Sprintf("unknown esc seq: %02x", bytes[0:2]))
		}
	case 0x0F:
		// LS0
		s.GL = s.G0
	case 0x0E:
		// LS1
		s.GL = s.G1
	case 0x19:
		// SS2
		s.GLSSVictim = s.GL
		s.GL = s.G2
		ss = true
	case 0x1D:
		// SS3
		s.GLSSVictim = s.GL
		s.GL = s.G3
		ss = true
	default:
		// ignore
	}
	return seqLen, ss, nil
}

func (s *decodeState) decodeGL(bytes []byte, repeatTime int) (string, int, error) {
	return s.decodeByGCharset(bytes, s.GL, repeatTime)
}

func (s *decodeState) decodeGR(bytes []byte, repeatTime int) (string, int, error) {
	return s.decodeByGCharset(bytes, s.GR, repeatTime)
}

func (s *decodeState) decodeByGCharset(bytes []byte, gset b24GCharset, repeatTime int) (string, int, error) {
	b0 := 0x7f&bytes[0] - 0x20
	b1 := uint8(0)
	if len(bytes) > 1 {
		b1 = 0x7f&bytes[1] - 0x20
	}
	resultString := ""
	seqLen := 0
	var err error = nil
	switch gset {
	case b24CSAlphabet:
		resultString = string([]byte{b0 + 0x20})
	case b24CSHiragana:
		resultString = b24HiraganaSet[b0]
	case b24CSKatakana:
		resultString = b24KatakanaSet[b0]
	case b24CSX0201Katakana:
		resultString, err = s.SJISDecoder.String(string([]byte{bytes[0] + 0x20}))
	case b24CSKanji:
		fallthrough
	case b24CSX0213Plane1:
		sjisBytes := make([]byte, 2)
		if b0 <= 62 {
			sjisBytes[0] = 0x80 + ((b0 + 1) >> 1)
		} else {
			sjisBytes[0] = 0xC0 + ((b0 + 1) >> 1)
		}
		if b0&1 == 1 {
			if b1 <= 63 {
				sjisBytes[1] = 0x3F + b1
			} else {
				sjisBytes[1] = 0x40 + b1
			}
		} else {
			sjisBytes[1] = 0x9E + b1
		}
		resultString, err = s.SJISDecoder.String(string(sjisBytes))
		seqLen = 1
	case b24CSX0213Plane2:
		sjisBytes := make([]byte, 2)
		sjisBytes[0] = 0xEF + ((b0 + 1) >> 1)
		if b0&1 == 1 {
			if b1 <= 63 {
				sjisBytes[1] = 0x3F + b1
			} else {
				sjisBytes[1] = 0x40 + b1
			}
		} else {
			sjisBytes[1] = 0x9E + b1
		}
		resultString, err = s.SJISDecoder.String(string(sjisBytes))
		seqLen = 1
	case b24CSGaiji:
		resultString = tryGaiji(b0, b1)
		seqLen = 1
	case b24CSIgnore1Byte:
		fallthrough
	case b24CSDRCS:
		return "", 0, nil
	case b24CSDRCS0:
		return "", 1, nil
	default:
		err = errors.New("unreachable in decodedByGSet")
	}
	return strings.Repeat(resultString, repeatTime), seqLen, err
}

func DecodeString(bytes []byte) (string, error) {
	state := newDecodeState()
	decoded := ""
	ss := false
	ssCount := 0
	inMarcoDef := false
	repeatTime := 1
	repeated := false

	for i := 0; i < len(bytes); i++ {
		if ss {
			ssCount++
		}
		if ssCount > 1 {
			ss = false
			state.GL = state.GLSSVictim
			ssCount = 0
		}
		if repeatTime > 1 {
			if repeated {
				repeatTime = 1
			}
			repeated = !repeated
		}
		b := bytes[i]
		b1 := uint8(0)
		if i+1 < len(bytes) {
			b1 = bytes[i+1]
		}
		if inMarcoDef {
			if b == 0x95 && b1 == 0x4f {
				// MARCO DEF END
				i++
				inMarcoDef = false
			}
			continue
		}
		if b <= 0x20 {
			// C0 Set
			switch b {
			case 0x0c:
				// CS
				decoded += "\f"
			case 0x0d:
				// APR
				decoded += "\n"
			case 0x16:
				// PAPF, takes one 1-byte parameter
				i++
			case 0x1c:
				// APS, takes two 1-byte parameters
				i += 2
			case 0x20:
				// SP
				decoded += " "
			default:
				seqLen, ss1, err := state.changeState(bytes[i:])
				if err != nil {
					return "", err
				}
				ss = ss1
				i += seqLen
			}
		} else if 0x20 < b && b < 0x80 {
			str, additionalLen, err := state.decodeGL(bytes[i:], repeatTime)
			if err != nil {
				return "", err
			}
			i += additionalLen
			decoded += str
		} else if 0x80 <= b && b < 0xA0 {
			// C1 Set, ignored
			switch b {
			case 0x8B:
				// SZX
				fallthrough
			case 0x90:
				// COL
				fallthrough
			case 0x91:
				// FLC
				fallthrough
			case 0x93:
				// POL
				fallthrough
			case 0x94:
				// WMM
				fallthrough
			case 0x97:
				// HLC
				// takes one 1-byte parameter
				i++
			case 0x9D:
				// TIME
				// takes two 1-byte parameters
				i += 2
			case 0x92:
				// CDC
				if b1&0xf0 == 0x20 {
					// two 1-byte parameters
					i += 2
				} else {
					i++
				}
			case 0x95:
				// MARCO
				if b1 == 0x40 {
					inMarcoDef = true
				}
			case 0x98:
				// RPC
				repeatTime = int(b1 & 0x3f)
				if repeatTime == 0 {
					// expected NL w/o CR effect by spec, though NLCR is applied
					repeatTime = 1
					decoded += "\n"
				}
			}
		} else if 0xa0 < b {
			str, additionalLen, err := state.decodeGR(bytes[i:], repeatTime)
			if err != nil {
				return "", err
			}
			i += additionalLen
			decoded += str
		}
	}
	return decoded, nil
}

func tryGaiji(b0, b1 uint8) string {
	if 90 <= b0 && b0 <= 94 {
		return _gaijiSet[b0-90][b1]
	}
	return fmt.Sprintf("{gaiji %02x%02x}", b0+0x20, b1+0x20)
}
