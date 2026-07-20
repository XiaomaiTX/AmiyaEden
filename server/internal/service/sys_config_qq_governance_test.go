package service

import "testing"

func TestValidateQQGovernanceSettings(t *testing.T) {
	tests := []struct {
		name    string
		input   QQGovernanceSettings
		wantErr bool
	}{
		{name: "defaults", input: QQGovernanceSettings{ScanIntervalMinutes: 15, MismatchConfirmations: 2, MismatchObservationHours: 2}},
		{name: "maximums", input: QQGovernanceSettings{ScanIntervalMinutes: 360, MismatchConfirmations: 3, MismatchObservationHours: 6}},
		{name: "invalid interval", input: QQGovernanceSettings{ScanIntervalMinutes: 20, MismatchConfirmations: 2, MismatchObservationHours: 2}, wantErr: true},
		{name: "invalid confirmations", input: QQGovernanceSettings{ScanIntervalMinutes: 15, MismatchConfirmations: 1, MismatchObservationHours: 2}, wantErr: true},
		{name: "invalid observation", input: QQGovernanceSettings{ScanIntervalMinutes: 15, MismatchConfirmations: 2, MismatchObservationHours: 0}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateQQGovernanceSettings(tt.input); (err != nil) != tt.wantErr {
				t.Fatalf("validateQQGovernanceSettings(%+v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
