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

func TestVariantIdentityRejectsEmptyColour(t *testing.T) {
	t.Parallel()

	_, err := VariantIdentityFor(SizeM, " ")
	if err == nil {
		t.Fatal("expected missing colour validation error")
	}
}

func TestValidateCategorySizeUsesStockSizesOnlyForClothing(t *testing.T) {
	t.Parallel()

	if err := ValidateCategorySize(CategoryShirt, SizeP); err != nil {
		t.Fatalf("shirt P should be valid: %v", err)
	}
	if err := ValidateCategorySize(CategoryHoodie, SizeNotApplicable); err == nil {
		t.Fatal("hoodie must reject not_applicable")
	}
	if err := ValidateCategorySize(CategoryVinyl, SizeNotApplicable); err != nil {
		t.Fatalf("vinyl implicit stock size should be valid: %v", err)
	}
	if err := ValidateCategorySize(CategoryVinyl, SizeM); err == nil {
		t.Fatal("vinyl must reject clothing size")
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

func TestPhotoMetadataAcceptsPortraitDisplayVariant(t *testing.T) {
	t.Parallel()

	photo := validPhotoMetadata()
	photo.Display.Width = 720
	photo.Display.Height = 960

	err := ValidatePhotoMetadata(photo)
	if err != nil {
		t.Fatalf("expected portrait display variant to be valid: %v", err)
	}
}

func TestPhotoMetadataRejectsOversizedDisplayVariant(t *testing.T) {
	t.Parallel()

	photo := validPhotoMetadata()
	photo.Display.Height = 961

	err := ValidatePhotoMetadata(photo)
	if err == nil {
		t.Fatal("expected display variant dimension validation error")
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
