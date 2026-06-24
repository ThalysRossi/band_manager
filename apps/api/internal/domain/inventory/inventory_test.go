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
		Full: PhotoVariantMetadata{
			ObjectKey:   "",
			ContentType: PhotoContentTypeWebP,
			SizeBytes:   1024,
			Width:       1200,
			Height:      900,
		},
		Display: PhotoVariantMetadata{
			ObjectKey:   "bands/band_1/inventory/photos/photo/display.webp",
			ContentType: PhotoContentTypeWebP,
			SizeBytes:   512,
			Width:       1280,
			Height:      960,
		},
	})
	if err == nil {
		t.Fatal("expected photo validation error")
	}
}

func TestPhotoMetadataRequiresWebPVariants(t *testing.T) {
	t.Parallel()

	photo := validPhotoMetadata()
	photo.Full.ContentType = "image/jpeg"

	err := ValidatePhotoMetadata(photo)
	if err == nil {
		t.Fatal("expected WebP content type validation error")
	}
}

func TestPhotoMetadataRejectsOversizedFullVariant(t *testing.T) {
	t.Parallel()

	photo := validPhotoMetadata()
	photo.Full.Width = 3841

	err := ValidatePhotoMetadata(photo)
	if err == nil {
		t.Fatal("expected full variant dimension validation error")
	}
}

func TestPhotoMetadataRejectsInvalidDisplayAspectRatio(t *testing.T) {
	t.Parallel()

	photo := validPhotoMetadata()
	photo.Display.Width = 1000
	photo.Display.Height = 1000

	err := ValidatePhotoMetadata(photo)
	if err == nil {
		t.Fatal("expected display aspect ratio validation error")
	}
}

func validPhotoMetadata() PhotoMetadata {
	return PhotoMetadata{
		Full: PhotoVariantMetadata{
			ObjectKey:   "bands/band_1/inventory/photos/photo/full.webp",
			ContentType: PhotoContentTypeWebP,
			SizeBytes:   1024,
			Width:       1200,
			Height:      900,
		},
		Display: PhotoVariantMetadata{
			ObjectKey:   "bands/band_1/inventory/photos/photo/display.webp",
			ContentType: PhotoContentTypeWebP,
			SizeBytes:   512,
			Width:       1280,
			Height:      960,
		},
	}
}
