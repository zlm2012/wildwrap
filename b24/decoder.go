package b24

// almost copied from https://github.com/eagletmt/eagletmt-recutils/blob/master/assdumper/assdumper.go
// with slight modification
/*
Copyright (c) 2014 Kohei Suzuki

MIT License

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.
*/

import (
	"fmt"
	"golang.org/x/text/encoding/japanese"
	"log"
	"unicode/utf8"
)

func DecodeString(bytes []byte, length int) string {
	eucjpDecoder := japanese.EUCJP.NewDecoder()
	decoded := ""
	nonDefaultColor := false

	for i := 0; i < length; i++ {
		b := bytes[i]
		if 0 <= b && b <= 0x20 {
			// ARIB STD-B24 第一編 第2部 表 7-14
			// ARIB STD-B24 第一編 第2部 表 7-15
			// C0 制御集合
			switch b {
			case 0x0c:
				// CS
				decoded += "\f"
			case 0x0d:
				// APR
				decoded += "\\n"
			case 0x20:
				// SP
				decoded += " "
			default:
				log.Printf("Unhandled C0 code: 0x%02x\n", b)
			}
		} else if 0x20 < b && b < 0x80 {
			log.Printf("Unhandled GL code: 0x%02x\n", b)
		} else if 0x80 <= b && b < 0xA0 {
			// ARIB STD-B24 第一編 第2部 表 7-14
			// ARIB STD-B24 第一編 第2部 表 7-16
			// C1 制御集合
			switch b {
			case 0x80:
				// BKF, black
				decoded += "{\\c&H000000&}"
				nonDefaultColor = true
			case 0x81:
				// RDF, red
				decoded += "{\\c&H0000ff&}"
				nonDefaultColor = true
			case 0x82:
				// GRF, green
				decoded += "{\\c&H00ff00&}"
				nonDefaultColor = true
			case 0x83:
				// YLF, yellow
				decoded += "{\\c&H00ffff&}"
				nonDefaultColor = true
			case 0x84:
				// BLF, blue
				decoded += "{\\c&Hff0000&}"
				nonDefaultColor = true
			case 0x85:
				// MGF, magenta
				decoded += "{\\c&Hff00ff&}"
				nonDefaultColor = true
			case 0x86:
				// CNF, cyan
				decoded += "{\\c&Hffff00&}"
				nonDefaultColor = true
			case 0x87:
				// WHF, white
				if nonDefaultColor {
					decoded += "{\\c&HFFFFFF&}"
					nonDefaultColor = false
				}
			case 0x89:
				// MSZ
			case 0x8a:
				// NSZ
			case 0x9d:
				// TIME
				i += 2
			default:
				log.Printf("Unhandled C1 code: 0x%02x\n", b)
			}
		} else if 0xa0 < b && b <= 0xff {
			eucjp := make([]byte, 3)
			eucjp[0] = bytes[i]
			eucjp[1] = bytes[i+1]
			eucjp[2] = 0
			i++

			if eucjp[0] == 0xfc && eucjp[1] == 0xa1 {
				// FIXME
				decoded += "➡"
			} else {
				buf := make([]byte, 10)
				ndst, nsrc, err := eucjpDecoder.Transform(buf, eucjp, true)
				if err == nil {
					if nsrc == 3 {
						c, _ := utf8.DecodeRune(buf)
						if c == 0xfffd {
							gaiji := (int(eucjp[0]&0x7f) << 8) | int(eucjp[1]&0x7f)
							if gaiji != 0x7c21 {
								decoded += tryGaiji(gaiji)
							}
						} else {
							decoded += string(buf[:ndst-1])
						}
					} else {
						log.Printf("eucjp decode failed: ndst=%d, nsrc=%d\n", ndst, nsrc)
					}
				} else {
					log.Printf("eucjp decode error: %v\n", err)
				}
			}
		}
	}
	return decoded
}

func tryGaiji(c int) string {
	switch c {
	case 0x7A50:
		return "[HV]"
	case 0x7A51:
		return "[SD]"
	case 0x7A52:
		return "[Ｐ]"
	case 0x7A53:
		return "[Ｗ]"
	case 0x7A54:
		return "[MV]"
	case 0x7A55:
		return "[手]"
	case 0x7A56:
		return "[字]"
	case 0x7A57:
		return "[双]"
	case 0x7A58:
		return "[デ]"
	case 0x7A59:
		return "[Ｓ]"
	case 0x7A5A:
		return "[二]"
	case 0x7A5B:
		return "[多]"
	case 0x7A5C:
		return "[解]"
	case 0x7A5D:
		return "[SS]"
	case 0x7A5E:
		return "[Ｂ]"
	case 0x7A5F:
		return "[Ｎ]"
	case 0x7A62:
		return "[天]"
	case 0x7A63:
		return "[交]"
	case 0x7A64:
		return "[映]"
	case 0x7A65:
		return "[無]"
	case 0x7A66:
		return "[料]"
	case 0x7A67:
		return "[年齢制限]"
	case 0x7A68:
		return "[前]"
	case 0x7A69:
		return "[後]"
	case 0x7A6A:
		return "[再]"
	case 0x7A6B:
		return "[新]"
	case 0x7A6C:
		return "[初]"
	case 0x7A6D:
		return "[終]"
	case 0x7A6E:
		return "[生]"
	case 0x7A6F:
		return "[販]"
	case 0x7A70:
		return "[声]"
	case 0x7A71:
		return "[吹]"
	case 0x7A72:
		return "[PPV]"

	case 0x7A60:
		return "■"
	case 0x7A61:
		return "●"
	case 0x7A73:
		return "（秘）"
	case 0x7A74:
		return "ほか"

	case 0x7C21:
		return "→"
	case 0x7C22:
		return "←"
	case 0x7C23:
		return "↑"
	case 0x7C24:
		return "↓"
	case 0x7C25:
		return "●"
	case 0x7C26:
		return "○"
	case 0x7C27:
		return "年"
	case 0x7C28:
		return "月"
	case 0x7C29:
		return "日"
	case 0x7C2A:
		return "円"
	case 0x7C2B:
		return "㎡"
	case 0x7C2C:
		return "㎥"
	case 0x7C2D:
		return "㎝"
	case 0x7C2E:
		return "㎠"
	case 0x7C2F:
		return "㎤"
	case 0x7C30:
		return "０."
	case 0x7C31:
		return "１."
	case 0x7C32:
		return "２."
	case 0x7C33:
		return "３."
	case 0x7C34:
		return "４."
	case 0x7C35:
		return "５."
	case 0x7C36:
		return "６."
	case 0x7C37:
		return "７."
	case 0x7C38:
		return "８."
	case 0x7C39:
		return "９."
	case 0x7C3A:
		return "氏"
	case 0x7C3B:
		return "副"
	case 0x7C3C:
		return "元"
	case 0x7C3D:
		return "故"
	case 0x7C3E:
		return "前"
	case 0x7C3F:
		return "[新]"
	case 0x7C40:
		return "０,"
	case 0x7C41:
		return "１,"
	case 0x7C42:
		return "２,"
	case 0x7C43:
		return "３,"
	case 0x7C44:
		return "４,"
	case 0x7C45:
		return "５,"
	case 0x7C46:
		return "６,"
	case 0x7C47:
		return "７,"
	case 0x7C48:
		return "８,"
	case 0x7C49:
		return "９,"
	case 0x7C4A:
		return "(社)"
	case 0x7C4B:
		return "(財)"
	case 0x7C4C:
		return "(有)"
	case 0x7C4D:
		return "(株)"
	case 0x7C4E:
		return "(代)"
	case 0x7C4F:
		return "(問)"
	case 0x7C50:
		return "▶"
	case 0x7C51:
		return "◀"
	case 0x7C52:
		return "〖"
	case 0x7C53:
		return "〗"
	case 0x7C54:
		return "⟐"
	case 0x7C55:
		return "^2"
	case 0x7C56:
		return "^3"
	case 0x7C57:
		return "(CD)"
	case 0x7C58:
		return "(vn)"
	case 0x7C59:
		return "(ob)"
	case 0x7C5A:
		return "(cb)"
	case 0x7C5B:
		return "(ce"
	case 0x7C5C:
		return "mb)"
	case 0x7C5D:
		return "(hp)"
	case 0x7C5E:
		return "(br)"
	case 0x7C5F:
		return "(p)"
	case 0x7C60:
		return "(s)"
	case 0x7C61:
		return "(ms)"
	case 0x7C62:
		return "(t)"
	case 0x7C63:
		return "(bs)"
	case 0x7C64:
		return "(b)"
	case 0x7C65:
		return "(tb)"
	case 0x7C66:
		return "(tp)"
	case 0x7C67:
		return "(ds)"
	case 0x7C68:
		return "(ag)"
	case 0x7C69:
		return "(eg)"
	case 0x7C6A:
		return "(vo)"
	case 0x7C6B:
		return "(fl)"
	case 0x7C6C:
		return "(ke"
	case 0x7C6D:
		return "y)"
	case 0x7C6E:
		return "(sa"
	case 0x7C6F:
		return "x)"
	case 0x7C70:
		return "(sy"
	case 0x7C71:
		return "n)"
	case 0x7C72:
		return "(or"
	case 0x7C73:
		return "g)"
	case 0x7C74:
		return "(pe"
	case 0x7C75:
		return "r)"
	case 0x7C76:
		return "(R)"
	case 0x7C77:
		return "(C)"
	case 0x7C78:
		return "(箏)"
	case 0x7C79:
		return "DJ"
	case 0x7C7A:
		return "[演]"
	case 0x7C7B:
		return "Fax"

	case 0x7D21:
		return "㈪"
	case 0x7D22:
		return "㈫"
	case 0x7D23:
		return "㈬"
	case 0x7D24:
		return "㈭"
	case 0x7D25:
		return "㈮"
	case 0x7D26:
		return "㈯"
	case 0x7D27:
		return "㈰"
	case 0x7D28:
		return "㈷"
	case 0x7D29:
		return "㍾"
	case 0x7D2A:
		return "㍽"
	case 0x7D2B:
		return "㍼"
	case 0x7D2C:
		return "㍻"
	case 0x7D2D:
		return "№"
	case 0x7D2E:
		return "℡"
	case 0x7D2F:
		return "〶"
	case 0x7D30:
		return "○"
	case 0x7D31:
		return "〔本〕"
	case 0x7D32:
		return "〔三〕"
	case 0x7D33:
		return "〔二〕"
	case 0x7D34:
		return "〔安〕"
	case 0x7D35:
		return "〔点〕"
	case 0x7D36:
		return "〔打〕"
	case 0x7D37:
		return "〔盗〕"
	case 0x7D38:
		return "〔勝〕"
	case 0x7D39:
		return "〔敗〕"
	case 0x7D3A:
		return "〔Ｓ〕"
	case 0x7D3B:
		return "［投］"
	case 0x7D3C:
		return "［捕］"
	case 0x7D3D:
		return "［一］"
	case 0x7D3E:
		return "［二］"
	case 0x7D3F:
		return "［三］"
	case 0x7D40:
		return "［遊］"
	case 0x7D41:
		return "［左］"
	case 0x7D42:
		return "［中］"
	case 0x7D43:
		return "［右］"
	case 0x7D44:
		return "［指］"
	case 0x7D45:
		return "［走］"
	case 0x7D46:
		return "［打］"
	case 0x7D47:
		return "㍑"
	case 0x7D48:
		return "㎏"
	case 0x7D49:
		return "㎐"
	case 0x7D4A:
		return "ha"
	case 0x7D4B:
		return "㎞"
	case 0x7D4C:
		return "㎢"
	case 0x7D4D:
		return "㍱"
	case 0x7D4E:
		return "・"
	case 0x7D4F:
		return "・"
	case 0x7D50:
		return "1/2"
	case 0x7D51:
		return "0/3"
	case 0x7D52:
		return "1/3"
	case 0x7D53:
		return "2/3"
	case 0x7D54:
		return "1/4"
	case 0x7D55:
		return "3/4"
	case 0x7D56:
		return "1/5"
	case 0x7D57:
		return "2/5"
	case 0x7D58:
		return "3/5"
	case 0x7D59:
		return "4/5"
	case 0x7D5A:
		return "1/6"
	case 0x7D5B:
		return "5/6"
	case 0x7D5C:
		return "1/7"
	case 0x7D5D:
		return "1/8"
	case 0x7D5E:
		return "1/9"
	case 0x7D5F:
		return "1/10"
	case 0x7D60:
		return "☀"
	case 0x7D61:
		return "☁"
	case 0x7D62:
		return "☂"
	case 0x7D63:
		return "☃"
	case 0x7D64:
		return "☖"
	case 0x7D65:
		return "☗"
	case 0x7D66:
		return "▽"
	case 0x7D67:
		return "▼"
	case 0x7D68:
		return "♦"
	case 0x7D69:
		return "♥"
	case 0x7D6A:
		return "♣"
	case 0x7D6B:
		return "♠"
	case 0x7D6C:
		return "⌺"
	case 0x7D6D:
		return "⦿"
	case 0x7D6E:
		return "‼"
	case 0x7D6F:
		return "⁉"
	case 0x7D70:
		return "(曇/晴)"
	case 0x7D71:
		return "☔"
	case 0x7D72:
		return "(雨)"
	case 0x7D73:
		return "(雪)"
	case 0x7D74:
		return "(大雪)"
	case 0x7D75:
		return "⚡"
	case 0x7D76:
		return "(雷雨)"
	case 0x7D77:
		return "　"
	case 0x7D78:
		return "・"
	case 0x7D79:
		return "・"
	case 0x7D7A:
		return "♬"
	case 0x7D7B:
		return "☎"

	case 0x7E21:
		return "Ⅰ"
	case 0x7E22:
		return "Ⅱ"
	case 0x7E23:
		return "Ⅲ"
	case 0x7E24:
		return "Ⅳ"
	case 0x7E25:
		return "Ⅴ"
	case 0x7E26:
		return "Ⅵ"
	case 0x7E27:
		return "Ⅶ"
	case 0x7E28:
		return "Ⅷ"
	case 0x7E29:
		return "Ⅸ"
	case 0x7E2A:
		return "Ⅹ"
	case 0x7E2B:
		return "Ⅺ"
	case 0x7E2C:
		return "Ⅻ"
	case 0x7E2D:
		return "⑰"
	case 0x7E2E:
		return "⑱"
	case 0x7E2F:
		return "⑲"
	case 0x7E30:
		return "⑳"
	case 0x7E31:
		return "⑴"
	case 0x7E32:
		return "⑵"
	case 0x7E33:
		return "⑶"
	case 0x7E34:
		return "⑷"
	case 0x7E35:
		return "⑸"
	case 0x7E36:
		return "⑹"
	case 0x7E37:
		return "⑺"
	case 0x7E38:
		return "⑻"
	case 0x7E39:
		return "⑼"
	case 0x7E3A:
		return "⑽"
	case 0x7E3B:
		return "⑾"
	case 0x7E3C:
		return "⑿"
	case 0x7E3D:
		return "㉑"
	case 0x7E3E:
		return "㉒"
	case 0x7E3F:
		return "㉓"
	case 0x7E40:
		return "㉔"
	case 0x7E41:
		return "(A)"
	case 0x7E42:
		return "(B)"
	case 0x7E43:
		return "(C)"
	case 0x7E44:
		return "(D)"
	case 0x7E45:
		return "(E)"
	case 0x7E46:
		return "(F)"
	case 0x7E47:
		return "(G)"
	case 0x7E48:
		return "(H)"
	case 0x7E49:
		return "(I)"
	case 0x7E4A:
		return "(J)"
	case 0x7E4B:
		return "(K)"
	case 0x7E4C:
		return "(L)"
	case 0x7E4D:
		return "(M)"
	case 0x7E4E:
		return "(N)"
	case 0x7E4F:
		return "(O)"
	case 0x7E50:
		return "(P)"
	case 0x7E51:
		return "(Q)"
	case 0x7E52:
		return "(R)"
	case 0x7E53:
		return "(S)"
	case 0x7E54:
		return "(T)"
	case 0x7E55:
		return "(U)"
	case 0x7E56:
		return "(V)"
	case 0x7E57:
		return "(W)"
	case 0x7E58:
		return "(X)"
	case 0x7E59:
		return "(Y)"
	case 0x7E5A:
		return "(Z)"
	case 0x7E5B:
		return "㉕"
	case 0x7E5C:
		return "㉖"
	case 0x7E5D:
		return "㉗"
	case 0x7E5E:
		return "㉘"
	case 0x7E5F:
		return "㉙"
	case 0x7E60:
		return "㉚"
	case 0x7E61:
		return "①"
	case 0x7E62:
		return "②"
	case 0x7E63:
		return "③"
	case 0x7E64:
		return "④"
	case 0x7E65:
		return "⑤"
	case 0x7E66:
		return "⑥"
	case 0x7E67:
		return "⑦"
	case 0x7E68:
		return "⑧"
	case 0x7E69:
		return "⑨"
	case 0x7E6A:
		return "⑩"
	case 0x7E6B:
		return "⑪"
	case 0x7E6C:
		return "⑫"
	case 0x7E6D:
		return "⑬"
	case 0x7E6E:
		return "⑭"
	case 0x7E6F:
		return "⑮"
	case 0x7E70:
		return "⑯"
	case 0x7E71:
		return "❶"
	case 0x7E72:
		return "❷"
	case 0x7E73:
		return "❸"
	case 0x7E74:
		return "❹"
	case 0x7E75:
		return "❺"
	case 0x7E76:
		return "❻"
	case 0x7E77:
		return "❼"
	case 0x7E78:
		return "❽"
	case 0x7E79:
		return "❾"
	case 0x7E7A:
		return "❿"
	case 0x7E7B:
		return "⓫"
	case 0x7E7C:
		return "⓬"
	case 0x7E7D:
		return "㉛"

	case 0x7521:
		return "㐂"
	case 0x7522:
		return "亭"
	case 0x7523:
		return "份"
	case 0x7524:
		return "仿"
	case 0x7525:
		return "侚"
	case 0x7526:
		return "俉"
	case 0x7527:
		return "傜"
	case 0x7528:
		return "儞"
	case 0x7529:
		return "冼"
	case 0x752A:
		return "㔟"
	case 0x752B:
		return "匇"
	case 0x752C:
		return "卡"
	case 0x752D:
		return "卬"
	case 0x752E:
		return "詹"
	case 0x752F:
		return "吉"
	case 0x7530:
		return "呍"
	case 0x7531:
		return "咖"
	case 0x7532:
		return "咜"
	case 0x7533:
		return "咩"
	case 0x7534:
		return "唎"
	case 0x7535:
		return "啊"
	case 0x7536:
		return "噲"
	case 0x7537:
		return "囤"
	case 0x7538:
		return "圳"
	case 0x7539:
		return "圴"
	case 0x753A:
		return "塚"
	case 0x753B:
		return "墀"
	case 0x753C:
		return "姤"
	case 0x753D:
		return "娣"
	case 0x753E:
		return "婕"
	case 0x753F:
		return "寬"
	case 0x7540:
		return "﨑"
	case 0x7541:
		return "㟢"
	case 0x7542:
		return "庬"
	case 0x7543:
		return "弴"
	case 0x7544:
		return "彅"
	case 0x7545:
		return "德"
	case 0x7546:
		return "怗"
	case 0x7547:
		return "恵"
	case 0x7548:
		return "愰"
	case 0x7549:
		return "昤"
	case 0x754A:
		return "曈"
	case 0x754B:
		return "曙"
	case 0x754C:
		return "曺"
	case 0x754D:
		return "曻"
	case 0x754E:
		return "桒"
	case 0x754F:
		return "・"
	case 0x7550:
		return "椑"
	case 0x7551:
		return "椻"
	case 0x7552:
		return "橅"
	case 0x7553:
		return "檑"
	case 0x7554:
		return "櫛"
	case 0x7555:
		return "・"
	case 0x7556:
		return "・"
	case 0x7557:
		return "・"
	case 0x7558:
		return "毱"
	case 0x7559:
		return "泠"
	case 0x755A:
		return "洮"
	case 0x755B:
		return "海"
	case 0x755C:
		return "涿"
	case 0x755D:
		return "淊"
	case 0x755E:
		return "淸"
	case 0x755F:
		return "渚"
	case 0x7560:
		return "潞"
	case 0x7561:
		return "濹"
	case 0x7562:
		return "灤"
	case 0x7563:
		return "・"
	case 0x7564:
		return "・"
	case 0x7565:
		return "煇"
	case 0x7566:
		return "燁"
	case 0x7567:
		return "爀"
	case 0x7568:
		return "玟"
	case 0x7569:
		return "・"
	case 0x756A:
		return "珉"
	case 0x756B:
		return "珖"
	case 0x756C:
		return "琛"
	case 0x756D:
		return "琡"
	case 0x756E:
		return "琢"
	case 0x756F:
		return "琦"
	case 0x7570:
		return "琪"
	case 0x7571:
		return "琬"
	case 0x7572:
		return "琹"
	case 0x7573:
		return "瑋"
	case 0x7574:
		return "㻚"
	case 0x7575:
		return "畵"
	case 0x7576:
		return "疁"
	case 0x7577:
		return "睲"
	case 0x7578:
		return "䂓"
	case 0x7579:
		return "磈"
	case 0x757A:
		return "磠"
	case 0x757B:
		return "祇"
	case 0x757C:
		return "禮"
	case 0x757D:
		return "・"
	case 0x757E:
		return "・"

	case 0x7621:
		return "・"
	case 0x7622:
		return "秚"
	case 0x7623:
		return "稞"
	case 0x7624:
		return "筿"
	case 0x7625:
		return "簱"
	case 0x7626:
		return "䉤"
	case 0x7627:
		return "綋"
	case 0x7628:
		return "羡"
	case 0x7629:
		return "脘"
	case 0x762A:
		return "脺"
	case 0x762B:
		return "・"
	case 0x762C:
		return "芮"
	case 0x762D:
		return "葛"
	case 0x762E:
		return "蓜"
	case 0x762F:
		return "蓬"
	case 0x7630:
		return "蕙"
	case 0x7631:
		return "藎"
	case 0x7632:
		return "蝕"
	case 0x7633:
		return "蟬"
	case 0x7634:
		return "蠋"
	case 0x7635:
		return "裵"
	case 0x7636:
		return "角"
	case 0x7637:
		return "諶"
	case 0x7638:
		return "跎"
	case 0x7639:
		return "辻"
	case 0x763A:
		return "迶"
	case 0x763B:
		return "郝"
	case 0x763C:
		return "鄧"
	case 0x763D:
		return "鄭"
	case 0x763E:
		return "醲"
	case 0x763F:
		return "鈳"
	case 0x7640:
		return "銈"
	case 0x7641:
		return "錡"
	case 0x7642:
		return "鍈"
	case 0x7643:
		return "閒"
	case 0x7644:
		return "雞"
	case 0x7645:
		return "餃"
	case 0x7646:
		return "饀"
	case 0x7647:
		return "髙"
	case 0x7648:
		return "鯖"
	case 0x7649:
		return "鷗"
	case 0x764A:
		return "麴"
	case 0x764B:
		return "麵"
	default:
		return fmt.Sprintf("{gaiji 0x%x}", c)
	}
}
