package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObfuscate(t *testing.T) {
	tests := []struct {
		src  string
		seed string
		dst  string
	}{
		{
			src:  "",
			seed: "01234567890123456789012345678901", // 32 bytes.
			dst:  "",
		},
		{
			src:  "hello",
			seed: "01234567890123456789012345678901", // 32 bytes.
			dst:  "WFReX1s=",
		},
		{
			src:  "你好!",
			seed: "ENuef1JjSvQ6VPfgrB33T2mkshhwRRjp", // 32 bytes.
			dst:  "ofPVgMOMaw==",
		},
		{
			src:  "Bytebase is a database tool for developers. Bytebase 是个数据库 DevOps 工具。",
			seed: "01234567890123456789012345678901", // 32 bytes.
			dst:  "ckhGVlZURVIYUEMRUxNQVEJWWlhDVBJHW1paF15WQhFUVERWWFpGUkpKHhFwSkBQVFZLXBDXqpzQjZzRrYnWvJ7UiKAUcVNBd0lDEdeEkdCzgNu5sg==",
		},
		{
			src:  `{   "type": "service_account",   "project_id": "spanner-test-371702",   "private_key_id": "klsdjfklasjdfas\nklsdjaflkajefjlaksdjf\nlsajdfklsjaldkfjkasldjf\nD PRIVATE KEY-----\n,   "client_email": "test-768@spanner-test-371702.iam.gserviceaccount.com",   "client_id": "102052620181224568340",   "auth_uri": "https://accounts.google.com/o/oauth2/auth",   "token_uri": "https://oauth2.googleapis.com/token",   "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",   "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test-768%40spanner-test-371702.iam.gserviceaccount.com" }`,
			seed: `aGgQpKjg7fuwNV6B31sIRQ1qm4Ttqw9s`, // 32 bytes.
			dst:  `GmdHcVI/ExdSRE9XbCVTMEVYECwNMFISAkE6AFNbGVNBZRcjHyEPBEM5HBNsbBZgQEESJzw0Q1wZUScAXEQOQlZ3VXNca0pHFRYHHjg3QidsWhYwDThVU1cUdh8dBF0ZBywLMAMhDgFWFSkZJTpFJllQFSU5MFsUC144FRoEXRkHGwk9AyoAA1ENGQQkN1omWFcZIjMiXRUHUggaNVdpISgRJgU1ayEibktYWmN7aiwfEVNpcDJdGAhaICsUGlgaDWVdcVI/DxRDS0JBdhZFMlJfHSwgfEUUHkB5R0ZGDkNTaQ4wHWUNFFIUAx4tM1chUF4GJyZ/Uh4AFnhUUVcbEA0uAj8EFAMDFVxVVX9mBHIGA0V7YmAJQF8GYEFHTwpHUWVLcVBrSAZCEh0oOyRfYAkRUSEmJUECVxt7FRIUVgYPMxR/FyQFAFsDWxQhOxktHF4SPCY5A14MQSAcU1sZU0FlEz4bLgQ4QhQcVXR2FCpHRQM6aH4eHgxBIBxDWV4cDiALNBE7AxQZBRoaYSJZKVZfUWVycRFTDEEgHC4HSxwXLgM0AhQSUgdfKhQrJEIdRkMfa2hxExkZQCQHS1gWBBYwSTYfJA0LUgcFHj14VS1eHhwoJyVZQ0JCZVsSEksHEmVLcVBrSARbDxAZOglOdwMILCo3I0UuGEY4VktXGxsVMxciSmRFEEARWxAhOVEuVlADICF/Uh4AGyYbExhNXBd2SDwVPwsDVhIUWDZjBnscRRY6JnwGR1URYEQCB1gdDyIVfAQuGRMaVUJGeWYEbFpQHmc1IlQDG103ERAUWhwUKRN/EyQHRRcb`,
		},
	}
	for _, test := range tests {
		obfuscated := obfuscate(test.src, test.seed)
		require.Equal(t, test.dst, obfuscated)
		ubobfuscated, err := unobfuscate(obfuscated, test.seed)
		require.NoError(t, err)
		require.Equal(t, test.src, ubobfuscated)
	}
}
