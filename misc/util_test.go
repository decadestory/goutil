package misc

import "testing"

func TestAesEncrypt(t *testing.T) {
	en, err := AesEncrypt("18961156547")
	if err != nil {
		t.Fatalf("AesEncrypt failed: %v", err)
	}
	t.Logf("AesEncrypt: %s", en)

	de, err := AesDecrypt("KIADMq2jJCO3qxazZclIAISRPU+yGm0nxXDm6dCg9K5n14oEQcRC")
	if err != nil {
		t.Fatalf("AesDecrypt failed: %v", err)
	}
	t.Logf("AesDecrypt: %s", de)

	if de != "18961156547" {
		t.Errorf("AesDecrypt result mismatch, got: %s, want: %s", de, "18961156547")
	}
}
