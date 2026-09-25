package inventory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
)

func insertColourVariant(ctx context.Context, tx pgx.Tx, colourVariantID string, productID string, bandID string, command applicationinventory.CreateProductVariantCommand, createdAt time.Time) error {
	photo := command.Photo
	_, err := tx.Exec(ctx, `
		INSERT INTO merch_colour_variants (
			id, band_id, product_id, colour, normalized_colour, price_amount, cost_amount, currency,
			photo_full_object_key, photo_full_content_type, photo_full_size_bytes, photo_full_width, photo_full_height,
			photo_display_object_key, photo_display_content_type, photo_display_size_bytes, photo_display_width, photo_display_height,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $19
		)
	`, colourVariantID, bandID, productID, command.Colour, command.NormalizedColour, command.Price.Amount, command.Cost.Amount, command.Price.Currency,
		photo.Full.ObjectKey, photo.Full.ContentType, photo.Full.SizeBytes, photo.Full.Width, photo.Full.Height,
		photo.Display.ObjectKey, photo.Display.ContentType, photo.Display.SizeBytes, photo.Display.Width, photo.Display.Height, createdAt)
	if err != nil {
		return mapPostgresError(err, fmt.Sprintf("insert colour variant band_id=%q product_id=%q colour=%q", bandID, productID, command.NormalizedColour))
	}
	return nil
}

func findOrCreateColourVariant(ctx context.Context, tx pgx.Tx, command applicationinventory.CreateVariantCommand) (string, error) {
	var colourVariantID string
	var priceAmount int
	var costAmount int
	var currency string
	var photo inventorydomain.PhotoMetadata
	err := tx.QueryRow(ctx, `
		SELECT id, price_amount, cost_amount, currency,
			photo_full_object_key, photo_full_content_type, photo_full_size_bytes, photo_full_width, photo_full_height,
			photo_display_object_key, photo_display_content_type, photo_display_size_bytes, photo_display_width, photo_display_height
		FROM merch_colour_variants
		WHERE product_id = $1 AND band_id = $2 AND normalized_colour = $3 AND deleted_at IS NULL
		FOR UPDATE
	`, command.ProductID, command.Account.BandID, command.NormalizedColour).Scan(
		&colourVariantID, &priceAmount, &costAmount, &currency,
		&photo.Full.ObjectKey, &photo.Full.ContentType, &photo.Full.SizeBytes, &photo.Full.Width, &photo.Full.Height,
		&photo.Display.ObjectKey, &photo.Display.ContentType, &photo.Display.SizeBytes, &photo.Display.Width, &photo.Display.Height,
	)
	if err == nil {
		if priceAmount != command.Price.Amount || costAmount != command.Cost.Amount || currency != command.Price.Currency || photo != command.Photo {
			return "", fmt.Errorf("%w: colour %q already exists with different photo, price, or cost", applicationinventory.ErrDuplicateVariant, command.NormalizedColour)
		}
		return colourVariantID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("query colour variant product_id=%q colour=%q: %w", command.ProductID, command.NormalizedColour, err)
	}
	colourVariantID = uuid.NewString()
	variant := applicationinventory.CreateProductVariantCommand{
		Colour: command.Colour, NormalizedColour: command.NormalizedColour,
		Price: command.Price, Cost: command.Cost, Photo: command.Photo,
	}
	if err := insertColourVariant(ctx, tx, colourVariantID, command.ProductID, command.Account.BandID, variant, command.CreatedAt); err != nil {
		return "", err
	}
	return colourVariantID, nil
}
