package main

import (
	"crypto/rsa"
	"embed"
	"errors"
	"testing"
)

//go:embed "assets/special/embed/*.ppm"
var embedded embed.FS

func TestVerifyRsa(t *testing.T) {
	
	e := env{}

	if err := e.initrsa(); err != nil {
		t.Fatalf("failed to initialize rsa key: %v", err)
	}
	
	ppm1, err := embedded.ReadFile("assets/special/embed/ppm1.ppm")
	if err != nil {
		t.Fatalf("failed to load embedded test file")
	}
	ppm2, err := embedded.ReadFile("assets/special/embed/ppm2.ppm")
	if err != nil {
		t.Fatalf("failed to load embedded test file")
	}
	ppm3, err := embedded.ReadFile("assets/special/embed/ppm3.ppm")
	if err != nil {
		t.Fatalf("failed to load embedded test file")
	}
	
	// big steppa
	tests := map[string]struct {
		in_ppm []byte
		expect_error bool
		err error
	}{
		"good": {
			in_ppm: ppm1, // unmodified
			expect_error: false,
		},
		"bad_sig": {
			in_ppm: ppm2, // bad signature
			expect_error: true,
			err: rsa.ErrVerification,
		},
		"bad_length": {
			in_ppm: ppm3, //
			expect_error: true,
			err: ErrInvalidPpmLength,
		},
	}
	
	for name, test := range tests {
		err := e.verifyrsa(test.in_ppm)
		if err != nil {
			if test.expect_error {
				if !errors.Is(err, test.err) {
					t.Fatalf("expected %q, got %q", test.err, err)
				}
			} else {
				t.Fatalf("test %s on verifyrsa failed with %v", name, err)
			}
		}
	}
}