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
