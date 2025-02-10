package quamina

import (
	"strings"
	"testing"
)

func TestRangeCreation(t *testing.T) {
	tests := []struct {
		name       string
		bottom     string
		openBottom bool
		top        string
		openTop    bool
		isCIDR     bool
		wantErr    bool
		errSubstr  string
	}{
		{
			name:       "valid range",
			bottom:     "1.0",
			openBottom: false,
			top:        "2.0",
			openTop:    false,
			wantErr:    false,
		},
		{
			name:       "invalid - bottom greater than top",
			bottom:     "2.0",
			openBottom: false,
			top:        "1.0",
			openTop:    false,
			wantErr:    true,
			errSubstr:  "bottom boundary must be less than top boundary",
		},
		{
			name:       "invalid - both bounds empty",
			bottom:     "",
			openBottom: false,
			top:        "",
			openTop:    false,
			wantErr:    true,
			errSubstr:  "at least one boundary must be specified",
		},
		{
			name:       "valid - only bottom bound",
			bottom:     "1.0",
			openBottom: false,
			top:        "",
			openTop:    false,
			wantErr:    false,
		},
		{
			name:       "valid - only top bound",
			bottom:     "",
			openBottom: false,
			top:        "1.0",
			openTop:    false,
			wantErr:    false,
		},
		{
			name:       "invalid - CIDR not implemented",
			bottom:     "192.168.0.0",
			openBottom: false,
			top:        "192.168.255.255",
			openTop:    false,
			isCIDR:     true,
			wantErr:    true,
			errSubstr:  "CIDR ranges not yet implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRange(tt.bottom, tt.openBottom, tt.top, tt.openTop, tt.isCIDR)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q but got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if r == nil {
				t.Error("expected non-nil Range but got nil")
			}
		})
	}
}

func TestRangeContains(t *testing.T) {
	tests := []struct {
		name       string
		bottom     string
		openBottom bool
		top        string
		openTop    bool
		testValue  string
		want       bool
	}{
		{
			name:       "inclusive range - value in middle",
			bottom:     "1.0",
			openBottom: false,
			top:        "3.0",
			openTop:    false,
			testValue:  "2.0",
			want:       true,
		},
		{
			name:       "inclusive range - value at bottom",
			bottom:     "1.0",
			openBottom: false,
			top:        "3.0",
			openTop:    false,
			testValue:  "1.0",
			want:       true,
		},
		{
			name:       "exclusive bottom - value at bottom",
			bottom:     "1.0",
			openBottom: true,
			top:        "3.0",
			openTop:    false,
			testValue:  "1.0",
			want:       false,
		},
		{
			name:       "exclusive top - value at top",
			bottom:     "1.0",
			openBottom: false,
			top:        "3.0",
			openTop:    true,
			testValue:  "3.0",
			want:       false,
		},
		{
			name:       "value below range",
			bottom:     "1.0",
			openBottom: false,
			top:        "3.0",
			openTop:    false,
			testValue:  "0.5",
			want:       false,
		},
		{
			name:       "value above range",
			bottom:     "1.0",
			openBottom: false,
			top:        "3.0",
			openTop:    false,
			testValue:  "3.5",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRange(tt.bottom, tt.openBottom, tt.top, tt.openTop, false)
			if err != nil {
				t.Fatalf("failed to create range: %v", err)
			}

			testNum, err := qNumFromBytes([]byte(tt.testValue))
			if err != nil {
				t.Fatalf("failed to create test number: %v", err)
			}

			got := r.Contains(testNum)
			if got != tt.want {
				t.Errorf("Contains(%s) = %v, want %v", tt.testValue, got, tt.want)
			}
		})
	}
}

func TestRangeFactoryMethods(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() (*Range, error)
		testValue string
		want      bool
	}{
		{
			name: "less than",
			setup: func() (*Range, error) {
				return LessThan("10.0", false)
			},
			testValue: "9.0",
			want:      true,
		},
		{
			name: "less than - equal value",
			setup: func() (*Range, error) {
				return LessThan("10.0", false)
			},
			testValue: "10.0",
			want:      false,
		},
		{
			name: "less than or equal",
			setup: func() (*Range, error) {
				return LessThanOrEqualTo("10.0", false)
			},
			testValue: "10.0",
			want:      true,
		},
		{
			name: "greater than",
			setup: func() (*Range, error) {
				return GreaterThan("10.0", false)
			},
			testValue: "11.0",
			want:      true,
		},
		{
			name: "greater than - equal value",
			setup: func() (*Range, error) {
				return GreaterThan("10.0", false)
			},
			testValue: "10.0",
			want:      false,
		},
		{
			name: "greater than or equal",
			setup: func() (*Range, error) {
				return GreaterThanOrEqualTo("10.0", false)
			},
			testValue: "10.0",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := tt.setup()
			if err != nil {
				t.Fatalf("failed to create range: %v", err)
			}

			testNum, err := qNumFromBytes([]byte(tt.testValue))
			if err != nil {
				t.Fatalf("failed to create test number: %v", err)
			}

			got := r.Contains(testNum)
			if got != tt.want {
				t.Errorf("Contains(%s) = %v, want %v", tt.testValue, got, tt.want)
			}
		})
	}
}

func TestRangeString(t *testing.T) {
	tests := []struct {
		name       string
		bottom     string
		openBottom bool
		top        string
		openTop    bool
		want       string
	}{
		{
			name:       "closed range",
			bottom:     "1.0",
			openBottom: false,
			top:        "2.0",
			openTop:    false,
			want:       "[1.0, 2.0]",
		},
		{
			name:       "open range",
			bottom:     "1.0",
			openBottom: true,
			top:        "2.0",
			openTop:    true,
			want:       "(1.0, 2.0)",
		},
		{
			name:       "half-open range",
			bottom:     "1.0",
			openBottom: false,
			top:        "2.0",
			openTop:    true,
			want:       "[1.0, 2.0)",
		},
		{
			name:       "unbounded below",
			bottom:     "",
			openBottom: true,
			top:        "2.0",
			openTop:    false,
			want:       "(-∞, 2.0]",
		},
		{
			name:       "unbounded above",
			bottom:     "1.0",
			openBottom: false,
			top:        "",
			openTop:    true,
			want:       "[1.0, +∞)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRange(tt.bottom, tt.openBottom, tt.top, tt.openTop, false)
			if err != nil {
				t.Fatalf("failed to create range: %v", err)
			}

			got := r.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRange_Contains(t *testing.T) {
	tests := []struct {
		name string
		r    *Range
		val  float64
		want bool
	}{
		{
			name: "equals match",
			r: &Range{
				bottom:     qNumFromFloat(100),
				top:        qNumFromFloat(100),
				openBottom: false,
				openTop:    false,
			},
			val:  100,
			want: true,
		},
		{
			name: "equals no match",
			r: &Range{
				bottom:     qNumFromFloat(100),
				top:        qNumFromFloat(100),
				openBottom: false,
				openTop:    false,
			},
			val:  99.9,
			want: false,
		},
		{
			name: "less than match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Implementation of the test case
		})
	}
}
