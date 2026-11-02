//go:build darwin

package internal

import "testing"

func TestShouldStopDarwinApplication(t *testing.T) {
	tests := []struct {
		name                string
		remainingTrays      int
		wantApplicationStop bool
	}{
		{name: "last tray removed", remainingTrays: 0, wantApplicationStop: true},
		{name: "one tray remains", remainingTrays: 1, wantApplicationStop: false},
		{name: "multiple trays remain", remainingTrays: 2, wantApplicationStop: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldStopDarwinApplication(test.remainingTrays); got != test.wantApplicationStop {
				t.Errorf("shouldStopDarwinApplication(%d) = %v, want %v", test.remainingTrays, got, test.wantApplicationStop)
			}
		})
	}
}

func TestDispatchDarwinClick(t *testing.T) {
	tests := []struct {
		name            string
		clickCount      int
		buttonNumber    int
		modifierFlags   uintptr
		wantClick       int
		wantDoubleClick int
	}{
		{name: "single left click", clickCount: 1, buttonNumber: nsLeftMouseButton, wantClick: 1},
		{name: "double left click", clickCount: 2, buttonNumber: nsLeftMouseButton, wantDoubleClick: 1},
		{name: "triple left click", clickCount: 3, buttonNumber: nsLeftMouseButton},
		{name: "right click", clickCount: 1, buttonNumber: 1},
		{name: "control click", clickCount: 1, buttonNumber: nsLeftMouseButton, modifierFlags: nsEventModifierFlagControl},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clicks, doubleClicks := 0, 0
			callbacks := &Callbacks{
				OnClick:       func() { clicks++ },
				OnDoubleClick: func() { doubleClicks++ },
			}

			dispatchDarwinClick(test.clickCount, test.buttonNumber, test.modifierFlags, callbacks)

			if clicks != test.wantClick || doubleClicks != test.wantDoubleClick {
				t.Errorf("callbacks = (%d, %d), want (%d, %d)", clicks, doubleClicks, test.wantClick, test.wantDoubleClick)
			}
		})
	}
}
