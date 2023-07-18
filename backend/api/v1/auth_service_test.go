package v1

import "testing"

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		phone string
		want  bool
	}{
		{
			phone: "1234567890",
			want:  false,
		},
		{
			phone: "+8615655556666",
			want:  true,
		},
	}

	for _, test := range tests {
		got := validatePhone(test.phone)
		isValid := got == nil
		if isValid != test.want {
			t.Errorf("validatePhone %s, err %v", test.phone, got)
		}
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{
			domain: "www.google.com",
			want:   "google.com",
		},
		{
			domain: "code.google.com",
			want:   "google.com",
		},
		{
			domain: "code.google.com.cn",
			want:   "google.com.cn",
		},
		{
			domain: "google.com",
			want:   "google.com",
		},
	}

	for _, test := range tests {
		got := extractDomain(test.domain)
		if got != test.want {
			t.Errorf("extractDomain %s, got %s, want %s", test.domain, got, test.want)
		}
	}
}
