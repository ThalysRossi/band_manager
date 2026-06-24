package inventory

import (
	"fmt"
	"strings"
)

type Category string

const (
	CategoryShirt     Category = "shirt"
	CategoryHoodie    Category = "hoodie"
	CategoryToteBag   Category = "tote_bag"
	CategoryPatch     Category = "patch"
	CategorySticker   Category = "sticker"
	CategoryVinyl     Category = "vinyl"
	CategoryCD        Category = "cd"
	CategoryCassette  Category = "cassette"
	CategoryAccessory Category = "accessory"
)

type Size string

const (
	SizeNotApplicable Size = "not_applicable"
	SizeOneSize       Size = "one_size"
	SizePP            Size = "pp"
	SizeP             Size = "p"
	SizeM             Size = "m"
	SizeG             Size = "g"
	SizeGG            Size = "gg"
	SizeXGG           Size = "xgg"
)

type Money struct {
	Amount   int
	Currency string
}

const (
	PhotoContentTypeWebP     = "image/webp"
	FullPhotoMaxSizeBytes    = 10 * 1024 * 1024
	FullPhotoMaxLongestEdge  = 3840
	DisplayPhotoMaxSizeBytes = 2 * 1024 * 1024
	DisplayPhotoMaxWidth     = 1280
	DisplayPhotoMaxHeight    = 960
	displayPhotoAspectWidth  = 4
	displayPhotoAspectHeight = 3
)

type PhotoMetadata struct {
	Full    PhotoVariantMetadata
	Display PhotoVariantMetadata
}

type PhotoVariantMetadata struct {
	ObjectKey   string
	ContentType string
	SizeBytes   int
	Width       int
	Height      int
}

type ProductIdentity struct {
	Category       Category
	NormalizedName string
}

type VariantIdentity struct {
	Size             Size
	NormalizedColour string
}

func ParseCategory(value string) (Category, error) {
	category := Category(strings.TrimSpace(value))
	if !category.IsValid() {
		return "", fmt.Errorf("invalid inventory category %q", value)
	}

	return category, nil
}

func (category Category) IsValid() bool {
	switch category {
	case CategoryShirt,
		CategoryHoodie,
		CategoryToteBag,
		CategoryPatch,
		CategorySticker,
		CategoryVinyl,
		CategoryCD,
		CategoryCassette,
		CategoryAccessory:
		return true
	default:
		return false
	}
}

func ParseSize(value string) (Size, error) {
	size := Size(strings.TrimSpace(value))
	if !size.IsValid() {
		return "", fmt.Errorf("invalid inventory size %q", value)
	}

	return size, nil
}

func (size Size) IsValid() bool {
	switch size {
	case SizeNotApplicable,
		SizeOneSize,
		SizePP,
		SizeP,
		SizeM,
		SizeG,
		SizeGG,
		SizeXGG:
		return true
	default:
		return false
	}
}

func ValidateMoney(label string, money Money) error {
	if strings.TrimSpace(label) == "" {
		return fmt.Errorf("money label is required")
	}

	if money.Amount < 0 {
		return fmt.Errorf("%s amount cannot be negative", label)
	}

	if strings.TrimSpace(money.Currency) != "BRL" {
		return fmt.Errorf("%s currency must be BRL", label)
	}

	return nil
}

func ValidateQuantity(quantity int) error {
	if quantity < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}

	return nil
}

func ValidatePhotoMetadata(photo PhotoMetadata) error {
	if err := validatePhotoVariantMetadata("full", photo.Full, FullPhotoMaxSizeBytes, FullPhotoMaxLongestEdge, FullPhotoMaxLongestEdge, false); err != nil {
		return err
	}

	if err := validatePhotoVariantMetadata("display", photo.Display, DisplayPhotoMaxSizeBytes, DisplayPhotoMaxWidth, DisplayPhotoMaxHeight, true); err != nil {
		return err
	}

	return nil
}

func validatePhotoVariantMetadata(label string, photo PhotoVariantMetadata, maxSizeBytes int, maxWidth int, maxHeight int, requireDisplayAspect bool) error {
	objectKey := strings.TrimSpace(photo.ObjectKey)
	if objectKey == "" {
		return fmt.Errorf("%s photo object key is required", label)
	}

	contentType := strings.TrimSpace(photo.ContentType)
	if contentType != PhotoContentTypeWebP {
		return fmt.Errorf("%s photo content type must be %s", label, PhotoContentTypeWebP)
	}

	if photo.SizeBytes <= 0 {
		return fmt.Errorf("%s photo size bytes must be greater than zero", label)
	}

	if photo.SizeBytes > maxSizeBytes {
		return fmt.Errorf("%s photo size bytes must be at most %d", label, maxSizeBytes)
	}

	if photo.Width <= 0 {
		return fmt.Errorf("%s photo width must be greater than zero", label)
	}

	if photo.Height <= 0 {
		return fmt.Errorf("%s photo height must be greater than zero", label)
	}

	if photo.Width > maxWidth {
		return fmt.Errorf("%s photo width must be at most %d", label, maxWidth)
	}

	if photo.Height > maxHeight {
		return fmt.Errorf("%s photo height must be at most %d", label, maxHeight)
	}

	if requireDisplayAspect && photo.Width*displayPhotoAspectHeight != photo.Height*displayPhotoAspectWidth {
		return fmt.Errorf("%s photo aspect ratio must be 4:3", label)
	}

	return nil
}

func NormalizeProductName(name string) (string, error) {
	normalized := normalizeIdentityText(name)
	if normalized == "" {
		return "", fmt.Errorf("product name is required")
	}

	return normalized, nil
}

func NormalizeColour(colour string) string {
	normalized := normalizeIdentityText(colour)
	if normalized == "" {
		return "not_applicable"
	}

	return normalized
}

func ProductIdentityFor(category Category, name string) (ProductIdentity, error) {
	if !category.IsValid() {
		return ProductIdentity{}, fmt.Errorf("invalid inventory category %q", category)
	}

	normalizedName, err := NormalizeProductName(name)
	if err != nil {
		return ProductIdentity{}, err
	}

	return ProductIdentity{
		Category:       category,
		NormalizedName: normalizedName,
	}, nil
}

func VariantIdentityFor(size Size, colour string) (VariantIdentity, error) {
	if !size.IsValid() {
		return VariantIdentity{}, fmt.Errorf("invalid inventory size %q", size)
	}

	return VariantIdentity{
		Size:             size,
		NormalizedColour: NormalizeColour(colour),
	}, nil
}

func normalizeIdentityText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
