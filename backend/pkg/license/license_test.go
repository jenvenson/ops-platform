package license

import (
	"testing"
)

func TestValidate(t *testing.T) {
	validKey := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE4MTI1Mzg3NDQsImZlYXR1cmVzIjpbIioiXSwiaWF0IjoxNzgxMDAyNzQ0LCJzdWIiOiJkZW1vLWN1c3RvbWVyIn0.b-tXA_2TJpEolVPABPdEugL2PvAlg0Od3oMeiT6_9XN-z2GdVNaNudPJnUaWMp9FGTZ9PNX50T_fd0_z34A8MjZ8J1M4PNEPAODm-hD-OslgPYpI1E4VygtRYbz_UxI6iMtpKcMlvkjU5UcBSXxotMhru_0NmbetiYaD2NIdGrHoC-aqZr5BR2d6pbFmLTsNjSw7lo0qAqlbX9FU7zSLo4XrEzXblJszJj8ZUVPViqpWOzX3SdlUM2epQNyuSAbvuixdjAN3shZAcABRaCWcPGiPn42c_ai1t__k0-5f0tlgZEucTIxk9_q_jU-XoJo5Q8VT4HFF6-itcBeAz2Ab8Q"
	
	t.Run("valid key", func(t *testing.T) {
		lic := Validate(validKey)
		if !lic.Valid {
			t.Fatalf("expected valid, got: %s", lic.ValidError)
		}
		if lic.Customer != "demo-customer" {
			t.Fatalf("expected demo-customer, got: %s", lic.Customer)
		}
		t.Logf("Customer: %s, Expires: %s, Features: %v", lic.Customer, lic.ExpiresAt, lic.Features)
	})

	t.Run("empty key", func(t *testing.T) {
		lic := Validate("")
		if lic.Valid {
			t.Fatal("expected invalid for empty key")
		}
	})

	t.Run("garbage key", func(t *testing.T) {
		lic := Validate("not.a.valid.jwt")
		if lic.Valid {
			t.Fatal("expected invalid for garbage key")
		}
	})
}
