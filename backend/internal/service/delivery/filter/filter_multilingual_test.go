package filter

import (
	"testing"

	"golang.org/x/text/unicode/norm"
)

func TestML_Chinese(t *testing.T) {
	t.Run("single CJK keyword substring", func(t *testing.T) {
		runCases(t, "监控告警", []matchCase{
			{"生产环境监控告警", true},
			{"监控告警系统", true},
			{"系统监控与告警", false},
		})
	})
	t.Run("OR over CJK keywords", func(t *testing.T) {
		runCases(t, "错误, 警告, 异常", []matchCase{
			{"发生错误了", true},
			{"CPU 警告", true},
			{"服务异常", true},
			{"正常运行", false},
		})
	})
	t.Run("realistic combined CJK rule", func(t *testing.T) {
		runCases(t, "线上 & (错误, 警告) & !测试", []matchCase{
			{"线上服务错误", true},
			{"线上 警告", true},
			{"线上测试错误", false},
			{"线下错误", false},
		})
	})
}

func TestML_Japanese(t *testing.T) {
	t.Run("mixed scripts with long vowel mark", func(t *testing.T) {
		runCases(t, "サーバーエラー", []matchCase{
			{"サーバーエラーが発生", true},
			{"サーバーエラー", true},
			{"サーバエラー", false},
		})
	})
	t.Run("kanji + hiragana", func(t *testing.T) {
		runCases(t, "通信エラー & 再試行", []matchCase{
			{"通信エラーのため再試行します", true},
			{"通信エラーのみ", false},
		})
	})
	t.Run("small kana distinction", func(t *testing.T) {
		runCases(t, "ッ", []matchCase{
			{"ちょっとッと", true},
			{"ゆとり", false},
		})
	})
}

func TestML_Korean_Normalization(t *testing.T) {
	composed := "오류"
	decomposed := norm.NFD.String(composed)
	if composed == decomposed {
		t.Skip("env already NFC-stable; NFD form identical")
	}

	t.Run("keyword NFC, title NFD — must still match", func(t *testing.T) {
		m, err := Compile(composed)
		if err != nil {
			t.Fatalf("compile: %v", err)
		}
		title := "시스템 " + decomposed + " 발생"
		if !m.Match(title) {
			t.Errorf("NFC keyword %q should match NFD title", composed)
		}
	})
	t.Run("keyword NFD, title NFC — must still match", func(t *testing.T) {
		m, err := Compile(decomposed)
		if err != nil {
			t.Fatalf("compile: %v", err)
		}
		if !m.Match("시스템 " + composed + " 발생") {
			t.Errorf("NFD keyword should match NFC title")
		}
	})
}

func TestML_Thai(t *testing.T) {
	runCases(t, "ข้อผิดพลาด, เตือน", []matchCase{
		{"ระบบเกิดข้อผิดพลาด", true},
		{"แจ้งเตือนระบบ", true},
		{"ปกติ", false},
	})
}

func TestML_Arabic_Hebrew_RTL(t *testing.T) {
	t.Run("Arabic", func(t *testing.T) {
		runCases(t, "خطأ & نظام", []matchCase{
			{"حدث خطأ في النظام", true},
			{"خطأ فقط", false},
		})
	})
	t.Run("Hebrew", func(t *testing.T) {
		runCases(t, "שגיאה", []matchCase{
			{"אירעה שגיאה במערכת", true},
			{"הצלחה", false},
		})
	})
}

func TestML_Cyrillic(t *testing.T) {
	runCases(t, "ошибка", []matchCase{
		{"СИСТЕМНАЯ ОШИБКА", true},
		{"Ошибка", true},
		{"предупреждение", false},
	})
}

func TestML_German(t *testing.T) {
	t.Run("umlauts case-insensitive", func(t *testing.T) {
		runCases(t, "Überwachung", []matchCase{
			{"die ÜBERWACHUNG", true},
			{"überwachung aktiv", true},
		})
	})
	t.Run("eszett (ß) folding limitation", func(t *testing.T) {
		runCases(t, "Straße", []matchCase{
			{"die Straße ist", true},
			{"STRASSE", false},
		})
	})
}

func TestML_LatinDiacritics(t *testing.T) {
	t.Run("Spanish accented vowel precomposed", func(t *testing.T) {
		runCases(t, "canción", []matchCase{
			{"la canción suena", true},
			{"la cancion suena", false},
		})
	})
	t.Run("composition-equivalent forms match", func(t *testing.T) {
		precomposed := "café"
		decomposed := "cafe\u0301"
		m, err := Compile(precomposed)
		if err != nil {
			t.Fatalf("compile: %v", err)
		}
		if !m.Match("un " + decomposed + " svp") {
			t.Errorf("precomposed %q should match decomposed title", precomposed)
		}
	})
	t.Run("Vietnamese stacked diacritics", func(t *testing.T) {
		runCases(t, "lỗi, hệ thống", []matchCase{
			{"hệ thống lỗi", true},
			{"bình thường", false},
		})
	})
}

func TestML_Emoji(t *testing.T) {
	t.Run("simple emoji substring", func(t *testing.T) {
		runCases(t, "🚀", []matchCase{
			{"launch 🚀 now", true},
			{"launch now", false},
		})
	})
	t.Run("flag regional indicators exact", func(t *testing.T) {
		runCases(t, "🇯🇵", []matchCase{
			{"hello 🇯🇵 world", true},
			{"🇰🇷 korea", false},
		})
	})
	t.Run("ZWJ family sequence byte-exact", func(t *testing.T) {
		runCases(t, "👨‍👩‍👧", []matchCase{
			{"family 👨‍👩‍👧 here", true},
			{"👨👩👧 no zwj", false},
		})
	})
	t.Run("skin tone modifier exact", func(t *testing.T) {
		runCases(t, "👋🏽", []matchCase{
			{"say 👋🏽 hi", true},
			{"say 👋 hi", false},
		})
	})
}

func TestML_MixedScripts(t *testing.T) {
	runCases(t, "Error, エラー, 错误, 🚀", []matchCase{
		{"Server Error 発生 🚀", true},
		{"ネットワークエラー", true},
		{"数据库连接错误", true},
		{"launch 🚀 please", true},
		{"一切正常", false},
	})
}

func TestML_FullWidth(t *testing.T) {
	t.Run("full-width comma is literal keyword content", func(t *testing.T) {
		runCases(t, "监控，告警", []matchCase{
			{"监控，告警系统", true},
			{"监控 告警", false},
		})
	})
	t.Run("full-width letters are NOT folded to ASCII", func(t *testing.T) {
		runCases(t, "ＡＢＣ", []matchCase{
			{"全角 ＡＢＣ 文字", true},
			{"half ABC here", false},
		})
	})
	t.Run("ideographic space U+3000 trimmed like ASCII space", func(t *testing.T) {
		runCases(t, "　监控　", []matchCase{
			{"系统监控", true},
		})
	})
}
