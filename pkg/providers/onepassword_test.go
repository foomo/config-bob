package providers

import "testing"

func Test_parseOnePasswordPath(t *testing.T) {

	tests := []struct {
		name        string
		path        string
		wantTitle   string
		wantSection string
		wantField   string
		wantErr     bool
	}{
		{"empty", "", "", "", "", true},
		{"two-part", "secret.field", "secret", "", "field", false},
		{"three-part", "secret.section.field", "secret", "section", "field", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotSection, gotField, err := parseOnePasswordPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOnePasswordPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("parseOnePasswordPath() gotTitle = %v, want %v", gotTitle, tt.wantTitle)
			}
			if gotSection != tt.wantSection {
				t.Errorf("parseOnePasswordPath() gotSection = %v, want %v", gotSection, tt.wantSection)
			}
			if gotField != tt.wantField {
				t.Errorf("parseOnePasswordPath() gotField = %v, want %v", gotField, tt.wantField)
			}
		})
	}
}
