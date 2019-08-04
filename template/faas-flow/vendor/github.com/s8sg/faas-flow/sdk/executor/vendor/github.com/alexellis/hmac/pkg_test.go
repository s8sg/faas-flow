package hmac

import (
	"encoding/hex"
	"testing"
)

func Test_GenerateInvalid_GivesError(t *testing.T) {

	input := []byte("test")
	signature := "ab"
	secretKey := "key"
	err := Validate(input, signature, secretKey)
	if err == nil {
		t.Errorf("expected error when signature didn't have at least 5 characters in length")
		t.Fail()
		return
	}

	wantErr := "invalid encodedHash, should have at least 5 characters"
	if err.Error() != wantErr {
		t.Errorf("want: %s, got: %s", wantErr, err.Error())
		t.Fail()
	}
}

func Test_ValidateWithoutSha1PrefixFails(t *testing.T) {
	digest := "sign this message"
	key := "my key"

	encodedHash := "6791a762f7568f945c2e1e396cea243e944100a6"

	valid := Validate([]byte(digest), encodedHash, key)

	if valid == nil {
		t.Errorf("Expected error due to missing prefix")
		t.Fail()
	}
}
func Test_ValidateWithSha1Prefix(t *testing.T) {
	digest := "sign this message"
	key := "my key"

	encodedHash := "sha1=" + "6791a762f7568f945c2e1e396cea243e944100a6"

	valid := Validate([]byte(digest), encodedHash, key)

	if valid != nil {
		t.Errorf("Expected no error, but got: %s", valid.Error())
		t.Fail()
	}
}

func Test_SignWithKey(t *testing.T) {
	digest := "sign this message"
	key := []byte("my key")

	wantHash := "6791a762f7568f945c2e1e396cea243e944100a6"

	hash := Sign([]byte(digest), key)
	encodedHash := hex.EncodeToString(hash)

	if encodedHash != wantHash {
		t.Errorf("Sign want hash: %s, got: %s", wantHash, encodedHash)
		t.Fail()
	}
}
