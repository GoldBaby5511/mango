package gorsa

import (
	"testing"
)

var Publickey = `-----BEGIN RSA PRIVATE KEY-----
MIGdMA0GCSqGSIb3DQEBAQUAA4GLADCBhwKBgQCgNCfas9omwx2CuSa3VPjDEdC9crw+kdSFSCkAOnF1cIzR6UCuzbLEgGmXqdRKKuHnJIMpZHPWg3bW+fIkv2nP6jY+HZVy7LgEPDYTSYgo5lMMYdfgHJB5iURA89x/h7tnOO3i4lQLqqAEL6IOk+iDtaj7eUE/NgBuor98gW+yBwIBAw==
-----END RSA PRIVATE KEY-----
`

var str = `UJ24diX/vlJOPgycQujgtUWRsxAGqGdRYDjFUwClCBbUSOOYdxhT1vJJakgQjefUBYbPV9RIsNfjQe68NsRd9YKBC7g7VkDxYxK06KhAgUBQHRxLxkvhx3spJe6+DvCsTXFhKWvfPfMh1n6oXbWkG/COmQtp9DxuMGNBV6tCJcU=`

var Pirvatekey = `-----BEGIN RSA PRIVATE KEY-----
MIICdAIBADANBgkqhkiG9w0BAQEFAASCAl4wggJaAgEAAoGBAKNxgr/o1Wp2jpCPU3U/A9AmQOy7yOwzkRW67ThBUV+aEiIjV+6N2ZePC9qblV1in0U9GICKXIdVl5cSZfrJnwDJdZ2FhEaRiZvi2Zuf1OekpiAWvXcOlcCm3PAZHKOregYmB/pQfr+wQc9KQ9n7dnibFetEGf7YN2EzCtWG8VSVAgEDAoGAGz2VyqbOPGkXwsKN6N/V+AZgJ3ShfLNC2PR83rWNj+8DBbCOp8JO7pfXTxnuOjsai4ouwBcPa+OZQ9hmVHbv1TEajOWbXpknCC4L4drn4l6ZrJ1ds1eiqF4kfZTVvHebP7YSACK0dWyT+U4tA4eD6TMCdcdx+cQ4uNAMy0zW5m0CQQD+TZ/jwPRx9qOWbluZwMZbpx+DAzvH1uYeUTRimwt3r+U0esBKmD06A8XQCZquwYH3OLSMnoy6FFAzdZsmV3ujAkEApIiwQB8aiKjHOCP05KTTEWT044gHOO7oU7DKOX8tZiairSE5NavB6sYxpSwqH51/cc50Cs+XhM68H0h2k5ByZwJBAKmJFUKAovakbQ70PRErLufEv6ys0oU57r7gzZcSB6Uf7iL8gDG603wCg+AGZx8rq/olzbMUXdFi4CJOZ27k/RcCQG2wdYAUvFsbL3rCo0MYjLZDTe0FWiX0muJ13CZUyO7EbHNre3kdK/HZdm4dcWpo/6E0TVyKZQM0fWowTw0K9u8CQH5GSr9mziPBRHdX8xf0zlEUEwxvO584qsXDsjPBA1eX6KS2ndNdYEdLAHsEhJbQVhqa/KOR2AsMzUTdx3VsD7Y=
-----END RSA PRIVATE KEY-----
`

func TestEncrypt(t *testing.T) {
	var tests = []string{
		"abasdf中222国",
		"12345678",
		"sjgfjvbj",
	}
	for _, test := range tests {
		enc, _ := Encrypt([]byte(test), []byte(Publickey))

		t.Log(string(enc))
		got, _ := Decrypt(enc, []byte(Pirvatekey), PKCS8)
		if string(got) != test {
			t.Errorf("Failed (%q) = %v", test, string(got))
		}
	}
}

func TestSign(t *testing.T) {
	var tests = []string{
		"abasdf中222国",
		"12345678",
		"sjgfjvbj",
	}
	for _, test := range tests {
		sign, _ := Sign([]byte(test), []byte(Pirvatekey), PKCS1)
		err := SignVer([]byte(test), sign, []byte(Publickey))
		if err != nil {
			t.Errorf("Failed %s", test)
		}
	}
}

func TestDecrypt(t *testing.T) {
	data, err := Decrypt([]byte(str), []byte(Pirvatekey), PKCS8)
	if err != nil {
		t.Errorf("Failed %s", err)
	}
	t.Logf("%s", string(data))
}
