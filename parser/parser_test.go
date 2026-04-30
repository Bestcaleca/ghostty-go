package parser

import (
	"strconv"
	"testing"
)

func TestParserGroundPrint(t *testing.T) {
	p := New()
	actions := p.Next('A')
	if actions[0] == nil {
		t.Fatal("expected PrintAction")
	}
	pa, ok := actions[0].(PrintAction)
	if !ok {
		t.Fatalf("expected PrintAction, got %T", actions[0])
	}
	if pa.Char != 'A' {
		t.Errorf("expected 'A', got %c", pa.Char)
	}
	if p.State() != StateGround {
		t.Errorf("expected Ground state, got %d", p.State())
	}
}

func TestParserC0Control(t *testing.T) {
	p := New()
	actions := p.Next(0x0A) // LF
	if actions[0] == nil {
		t.Fatal("expected ExecuteAction")
	}
	ea, ok := actions[0].(ExecuteAction)
	if !ok {
		t.Fatalf("expected ExecuteAction, got %T", actions[0])
	}
	if ea.Byte != 0x0A {
		t.Errorf("expected 0x0A, got 0x%02X", ea.Byte)
	}
}

func TestParserCSISequence(t *testing.T) {
	// CSI 2 J = Erase Display
	p := New()
	// ESC
	p.Next(0x1B)
	if p.State() != StateEscape {
		t.Fatalf("expected Escape state, got %d", p.State())
	}
	// [
	p.Next(0x5B)
	if p.State() != StateCSIEntry {
		t.Fatalf("expected CSIEntry state, got %d", p.State())
	}
	// 2
	p.Next(0x32)
	if p.State() != StateCSIParam {
		t.Fatalf("expected CSIParam state, got %d", p.State())
	}
	// J (final byte)
	var csiAction *CSIDispatchAction
	actions := p.Next(0x4A)
	for _, a := range actions {
		if a != nil {
			if csi, ok := a.(CSIDispatchAction); ok {
				csiAction = &csi
			}
		}
	}
	if csiAction == nil {
		t.Fatal("expected CSIDispatchAction")
	}
	if csiAction.Final != 'J' {
		t.Errorf("expected final 'J', got %c", csiAction.Final)
	}
	if csiAction.Param(1, 0) != 2 {
		t.Errorf("expected param 2, got %d", csiAction.Param(1, 0))
	}
	if p.State() != StateGround {
		t.Errorf("expected Ground state, got %d", p.State())
	}
}

func TestParserCSISGR(t *testing.T) {
	// CSI 31 m = Set foreground red
	p := New()
	p.Next(0x1B) // ESC
	p.Next(0x5B) // [
	p.Next(0x33) // 3
	p.Next(0x31) // 1

	var csiAction *CSIDispatchAction
	actions := p.Next(0x6D) // m
	for _, a := range actions {
		if a != nil {
			if csi, ok := a.(CSIDispatchAction); ok {
				csiAction = &csi
			}
		}
	}
	if csiAction == nil {
		t.Fatal("expected CSIDispatchAction")
	}
	if csiAction.Final != 'm' {
		t.Errorf("expected final 'm', got %c", csiAction.Final)
	}
	if csiAction.Param(1, 0) != 31 {
		t.Errorf("expected param 31, got %d", csiAction.Param(1, 0))
	}
}

func TestParserCSIMultipleParams(t *testing.T) {
	// CSI 1 ; 32 m = Bold green
	p := New()
	p.Next(0x1B) // ESC
	p.Next(0x5B) // [
	p.Next(0x31) // 1
	p.Next(0x3B) // ;
	p.Next(0x33) // 3
	p.Next(0x32) // 2

	var csiAction *CSIDispatchAction
	actions := p.Next(0x6D) // m
	for _, a := range actions {
		if a != nil {
			if csi, ok := a.(CSIDispatchAction); ok {
				csiAction = &csi
			}
		}
	}
	if csiAction == nil {
		t.Fatal("expected CSIDispatchAction")
	}
	if csiAction.ParamCount < 2 {
		t.Fatalf("expected 2 params, got %d", csiAction.ParamCount)
	}
	if csiAction.Param(1, 0) != 1 {
		t.Errorf("expected param1=1, got %d", csiAction.Param(1, 0))
	}
	if csiAction.Param(2, 0) != 32 {
		t.Errorf("expected param2=32, got %d", csiAction.Param(2, 0))
	}
}

func TestParserOSCSet(t *testing.T) {
	// ESC ] 0 ; H e l l o BEL = Set title "Hello"
	p := New()
	p.Next(0x1B) // ESC
	p.Next(0x5D) // ] -> OSC String
	if p.State() != StateOSCString {
		t.Fatalf("expected OSCString state, got %d", p.State())
	}
	p.Next(0x30) // 0
	p.Next(0x3B) // ;
	p.Next(0x48) // H
	p.Next(0x65) // e
	p.Next(0x6C) // l
	p.Next(0x6C) // l
	p.Next(0x6F) // o

	var oscAction *OSCDispatchAction
	actions := p.Next(0x07) // BEL
	for _, a := range actions {
		if a != nil {
			if osc, ok := a.(OSCDispatchAction); ok {
				oscAction = &osc
			}
		}
	}
	if oscAction == nil {
		t.Fatal("expected OSCDispatchAction")
	}
	if string(oscAction.Command) != "0;Hello" {
		t.Errorf("expected '0;Hello', got '%s'", string(oscAction.Command))
	}
}

func TestParserESCSequence(t *testing.T) {
	// ESC 7 = Save Cursor
	p := New()
	p.Next(0x1B) // ESC
	if p.State() != StateEscape {
		t.Fatalf("expected Escape state, got %d", p.State())
	}

	var escAction *EscDispatchAction
	actions := p.Next(0x37) // 7
	for _, a := range actions {
		if a != nil {
			if esc, ok := a.(EscDispatchAction); ok {
				escAction = &esc
			}
		}
	}
	if escAction == nil {
		t.Fatal("expected EscDispatchAction")
	}
	if escAction.Final != '7' {
		t.Errorf("expected final '7', got %c", escAction.Final)
	}
	if p.State() != StateGround {
		t.Errorf("expected Ground state, got %d", p.State())
	}
}

func TestParserCSICursorMovement(t *testing.T) {
	// CSI 5 A = Cursor Up 5
	p := New()
	p.Next(0x1B) // ESC
	p.Next(0x5B) // [
	p.Next(0x35) // 5

	var csiAction *CSIDispatchAction
	actions := p.Next(0x41) // A
	for _, a := range actions {
		if a != nil {
			if csi, ok := a.(CSIDispatchAction); ok {
				csiAction = &csi
			}
		}
	}
	if csiAction == nil {
		t.Fatal("expected CSIDispatchAction")
	}
	if csiAction.Final != 'A' {
		t.Errorf("expected final 'A', got %c", csiAction.Final)
	}
	if csiAction.Param(1, 1) != 5 {
		t.Errorf("expected param 5, got %d", csiAction.Param(1, 1))
	}
}

func TestParserDCS(t *testing.T) {
	// ESC P ... ESC \ = DCS passthrough
	p := New()
	p.Next(0x1B) // ESC
	if p.State() != StateEscape {
		t.Fatalf("expected Escape, got %d", p.State())
	}
	p.Next(0x50) // P -> DCS Entry
	if p.State() != StateDCSEntry {
		t.Fatalf("expected DCSEntry, got %d", p.State())
	}
	p.Next(0x31) // 1 -> DCS Param
	if p.State() != StateDCSParam {
		t.Fatalf("expected DCSParam, got %d", p.State())
	}

	// Final byte triggers DCS hook
	actions := p.Next(0x70) // p -> DCS Passthrough
	var hookFound bool
	for _, a := range actions {
		if _, ok := a.(DCSHookAction); ok {
			hookFound = true
		}
	}
	if !hookFound {
		t.Error("expected DCSHookAction")
	}
	if p.State() != StateDCSPassthrough {
		t.Fatalf("expected DCSPassthrough, got %d", p.State())
	}

	// Data bytes
	p.Next(0x41) // data
	p.Next(0x42) // data

	// ESC \ = ST -> exit DCS
	// DCSUnhookAction is emitted when ESC transitions out of DCSPassthrough
	actions = p.Next(0x1B) // ESC -> exit DCSPassthrough
	var unhookFound bool
	for _, a := range actions {
		if _, ok := a.(DCSUnhookAction); ok {
			unhookFound = true
		}
	}
	if !unhookFound {
		t.Error("expected DCSUnhookAction on ESC")
	}

	// \ completes the ESC sequence (ST)
	actions = p.Next(0x5C) // \ (ST) -> EscDispatch
	if p.State() != StateGround {
		t.Errorf("expected Ground state after ST, got %d", p.State())
	}
}

func TestParserMaxParams(t *testing.T) {
	// Test with many params
	p := New()
	p.Next(0x1B) // ESC
	p.Next(0x5B) // [

	// Send 24 params: 1;2;3;...;24
	for i := 0; i < MaxParams; i++ {
		if i > 0 {
			p.Next(0x3B) // ;
		}
		// Write the number
		n := i + 1
		digits := strconv.Itoa(n)
		for _, d := range digits {
			p.Next(byte(d))
		}
	}

	var csiAction *CSIDispatchAction
	actions := p.Next(0x6D) // m
	for _, a := range actions {
		if a != nil {
			if csi, ok := a.(CSIDispatchAction); ok {
				csiAction = &csi
			}
		}
	}
	if csiAction == nil {
		t.Fatal("expected CSIDispatchAction")
	}
	if csiAction.ParamCount != MaxParams {
		t.Errorf("expected %d params, got %d", MaxParams, csiAction.ParamCount)
	}
}

func TestParserDefaultParam(t *testing.T) {
	// CSI A = Cursor Up 1 (default)
	p := New()
	p.Next(0x1B) // ESC
	p.Next(0x5B) // [

	var csiAction *CSIDispatchAction
	actions := p.Next(0x41) // A
	for _, a := range actions {
		if a != nil {
			if csi, ok := a.(CSIDispatchAction); ok {
				csiAction = &csi
			}
		}
	}
	if csiAction == nil {
		t.Fatal("expected CSIDispatchAction")
	}
	// Param(1, 1) should return 1 (default) since no param was given
	if csiAction.Param(1, 1) != 1 {
		t.Errorf("expected default param 1, got %d", csiAction.Param(1, 1))
	}
}

func TestParserCSIIntermediate(t *testing.T) {
	// CSI SP q = Set cursor style 0 (DECSCUSR) — intermediate without params
	p := New()
	p.Next(0x1B) // ESC
	p.Next(0x5B) // [
	p.Next(0x20) // SP -> CSI Intermediate

	if p.State() != StateCSIIntermediate {
		t.Fatalf("expected CSIIntermediate, got %d", p.State())
	}

	var csiAction *CSIDispatchAction
	actions := p.Next(0x71) // q (final byte)
	for _, a := range actions {
		if a != nil {
			if csi, ok := a.(CSIDispatchAction); ok {
				csiAction = &csi
			}
		}
	}
	if csiAction == nil {
		t.Fatal("expected CSIDispatchAction")
	}
	if csiAction.Final != 'q' {
		t.Errorf("expected final 'q', got %c", csiAction.Final)
	}
	if csiAction.IntermediateCount != 1 || csiAction.Intermediates[0] != ' ' {
		t.Errorf("expected intermediate SP, got %v", csiAction.Intermediates[:csiAction.IntermediateCount])
	}
}

func TestParserIgnore(t *testing.T) {
	// 0x7F in ground should be ignored
	p := New()
	actions := p.Next(0x7F) // DEL
	for _, a := range actions {
		if a != nil {
			t.Errorf("expected no action for DEL in ground, got %T", a)
		}
	}
	if p.State() != StateGround {
		t.Errorf("expected Ground state, got %d", p.State())
	}
}

func TestParserReset(t *testing.T) {
	p := New()
	p.Next(0x1B) // ESC -> Escape state
	p.Reset()
	if p.State() != StateGround {
		t.Errorf("expected Ground state after reset, got %d", p.State())
	}
}

// Benchmark
func BenchmarkParserNext(b *testing.B) {
	p := New()
	// Typical terminal output: mix of printable, CSI sequences, and newlines
	data := []byte("Hello, World!\r\n\x1b[31mRed text\x1b[0m\r\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, c := range data {
			p.Next(c)
		}
	}
}

func BenchmarkParserPrint(b *testing.B) {
	p := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Next('A')
	}
}
