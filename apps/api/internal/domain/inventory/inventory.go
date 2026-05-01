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

type PhotoMetadata struct {
	ObjectKey   string
	ContentType string
	SizeBytes   int
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
	objectKey := strings.TrimSpace(photo.ObjectKey)
	if objectKey == "" {
		return fmt.Errorf("photo object key is required")
	}

	contentType := strings.TrimSpace(photo.ContentType)
	if contentType == "" {
		return fmt.Errorf("photo content type is required")
	}

	if photo.SizeBytes <= 0 {
		return fmt.Errorf("photo size bytes must be greater than zero")
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
