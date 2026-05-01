package inventory

import "testing"

func TestValidateMoneyRejectsNegativeAmount(t *testing.T) {
	t.Parallel()

	err := ValidateMoney("price", Money{Amount: -1, Currency: "BRL"})
	if err == nil {
		t.Fatal("expected negative money validation error")
	}
}

func TestValidateMoneyRequiresBRL(t *testing.T) {
	t.Parallel()

	err := ValidateMoney("cost", Money{Amount: 1000, Currency: "USD"})
	if err == nil {
		t.Fatal("expected currency validation error")
	}
}

func TestValidateQuantityRejectsNegativeValue(t *testing.T) {
	t.Parallel()

	err := ValidateQuantity(-1)
	if err == nil {
		t.Fatal("expected negative quantity validation error")
	}
}

func TestNormalizeProductIdentity(t *testing.T) {
	t.Parallel()

	identity, err := ProductIdentityFor(CategoryShirt, "  Camiseta   Logo  ")
	if err != nil {
		t.Fatalf("product identity: %v", err)
	}

	if identity.NormalizedName != "camiseta logo" {
		t.Fatalf("expected normalized name, got %q", identity.NormalizedName)
	}
}

func TestVariantIdentityNormalizesEmptyColour(t *testing.T) {
	t.Parallel()

	identity, err := VariantIdentityFor(SizeM, " ")
	if err != nil {
		t.Fatalf("variant identity: %v", err)
	}

	if identity.NormalizedColour != "not_applicable" {
		t.Fatalf("expected not_applicable colour identity, got %q", identity.NormalizedColour)
	}
}

func TestPhotoMetadataIsRequired(t *testing.T) {
	t.Parallel()

	err := ValidatePhotoMetadata(PhotoMetadata{
		ObjectKey:   "",
		ContentType: "image/jpeg",
		SizeBytes:   1024,
	})
	if err == nil {
		t.Fatal("expected photo validation error")
	}
}
