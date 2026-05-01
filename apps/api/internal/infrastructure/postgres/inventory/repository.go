package inventory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
)

const (
	duplicateProductConstraint = "merch_products_active_identity_idx"
	duplicateVariantConstraint = "merch_variants_active_identity_idx"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

func (repository Repository) CreateProduct(ctx context.Context, command applicationinventory.CreateProductCommand) (applicationinventory.Product, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return applicationinventory.Product{}, fmt.Errorf("begin inventory create transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	productID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO merch_products (
			id, band_id, name, normalized_name, category,
			photo_object_key, photo_content_type, photo_size_bytes,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, productID, command.Account.BandID, command.Name, command.NormalizedName, command.Category, command.Photo.ObjectKey, command.Photo.ContentType, command.Photo.SizeBytes, command.CreatedAt)
	if err != nil {
		return applicationinventory.Product{}, mapPostgresError(err, fmt.Sprintf("insert inventory product band_id=%q name=%q category=%q", command.Account.BandID, command.Name, command.Category))
	}

	variants := make([]applicationinventory.Variant, 0, len(command.Variants))
	for _, variantCommand := range command.Variants {
		variantID := uuid.NewString()
		_, err = tx.Exec(ctx, `
			INSERT INTO merch_variants (
				id, band_id, product_id, size, colour, normalized_colour,
				price_amount, cost_amount, currency, quantity, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		`, variantID, command.Account.BandID, productID, variantCommand.Size, variantCommand.Colour, variantCommand.NormalizedColour, variantCommand.Price.Amount, variantCommand.Cost.Amount, variantCommand.Price.Currency, variantCommand.Quantity, command.CreatedAt)
		if err != nil {
			return applicationinventory.Product{}, mapPostgresError(err, fmt.Sprintf("insert inventory variant band_id=%q product_id=%q size=%q colour=%q", command.Account.BandID, productID, variantCommand.Size, variantCommand.NormalizedColour))
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO inventory_movements (
				id, band_id, product_id, variant_id, movement_type,
				quantity_delta, quantity_after, reason, actor_user_id, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, uuid.NewString(), command.Account.BandID, productID, variantID, "initial_stock", variantCommand.Quantity, variantCommand.Quantity, "inventory.product_created", command.Account.UserID, command.CreatedAt)
		if err != nil {
			return applicationinventory.Product{}, fmt.Errorf("insert initial inventory movement band_id=%q variant_id=%q: %w", command.Account.BandID, variantID, err)
		}

		variants = append(variants, applicationinventory.Variant{
			ID:               variantID,
			ProductID:        productID,
			Size:             variantCommand.Size,
			Colour:           variantCommand.Colour,
			NormalizedColour: variantCommand.NormalizedColour,
			Price:            variantCommand.Price,
			Cost:             variantCommand.Cost,
			Quantity:         variantCommand.Quantity,
			CreatedAt:        command.CreatedAt,
			UpdatedAt:        command.CreatedAt,
		})
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "inventory.product_created", "merch_product", productID, command.RequestID, command.IdempotencyKey, command.CreatedAt); err != nil {
		return applicationinventory.Product{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return applicationinventory.Product{}, fmt.Errorf("commit inventory create transaction band_id=%q product_id=%q: %w", command.Account.BandID, productID, err)
	}

	return applicationinventory.Product{
		ID:             productID,
		BandID:         command.Account.BandID,
		Name:           command.Name,
		NormalizedName: command.NormalizedName,
		Category:       command.Category,
		Photo:          command.Photo,
		Variants:       variants,
		CreatedAt:      command.CreatedAt,
		UpdatedAt:      command.CreatedAt,
	}, nil
}

func (repository Repository) ListInventory(ctx context.Context, query applicationinventory.ListInventoryQuery) ([]applicationinventory.Product, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id, band_id, name, normalized_name, category,
			photo_object_key, photo_content_type, photo_size_bytes,
			created_at, updated_at
		FROM merch_products
		WHERE band_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC
	`, query.Account.BandID)
	if err != nil {
		return nil, fmt.Errorf("query inventory products band_id=%q: %w", query.Account.BandID, err)
	}
	defer rows.Close()

	products := make([]applicationinventory.Product, 0)
	productIndexes := make(map[string]int)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan inventory product band_id=%q: %w", query.Account.BandID, err)
		}

		productIndexes[product.ID] = len(products)
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory products band_id=%q: %w", query.Account.BandID, err)
	}

	if len(products) == 0 {
		return products, nil
	}

	variantRows, err := repository.pool.Query(ctx, `
		SELECT id, product_id, size, colour, normalized_colour,
			price_amount, cost_amount, currency, quantity, created_at, updated_at
		FROM merch_variants
		WHERE band_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC
	`, query.Account.BandID)
	if err != nil {
		return nil, fmt.Errorf("query inventory variants band_id=%q: %w", query.Account.BandID, err)
	}
	defer variantRows.Close()

	for variantRows.Next() {
		variant, err := scanVariant(variantRows)
		if err != nil {
			return nil, fmt.Errorf("scan inventory variant band_id=%q: %w", query.Account.BandID, err)
		}

		productIndex, ok := productIndexes[variant.ProductID]
		if ok {
			products[productIndex].Variants = append(products[productIndex].Variants, variant)
		}
	}
	if err := variantRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory variants band_id=%q: %w", query.Account.BandID, err)
	}

	return products, nil
}

func (repository Repository) UpdateProduct(ctx context.Context, command applicationinventory.UpdateProductCommand) (applicationinventory.Product, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return applicationinventory.Product{}, fmt.Errorf("begin inventory product update transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	commandTag, err := tx.Exec(ctx, `
		UPDATE merch_products
		SET name = $1,
			normalized_name = $2,
			category = $3,
			photo_object_key = $4,
			photo_content_type = $5,
			photo_size_bytes = $6,
			updated_at = $7
		WHERE id = $8 AND band_id = $9 AND deleted_at IS NULL
	`, command.Name, command.NormalizedName, command.Category, command.Photo.ObjectKey, command.Photo.ContentType, command.Photo.SizeBytes, command.UpdatedAt, command.ProductID, command.Account.BandID)
	if err != nil {
		return applicationinventory.Product{}, mapPostgresError(err, fmt.Sprintf("update inventory product band_id=%q product_id=%q", command.Account.BandID, command.ProductID))
	}
	if commandTag.RowsAffected() == 0 {
		return applicationinventory.Product{}, fmt.Errorf("%w: band_id=%q product_id=%q", applicationinventory.ErrInventoryNotFound, command.Account.BandID, command.ProductID)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "inventory.product_updated", "merch_product", command.ProductID, command.RequestID, command.IdempotencyKey, command.UpdatedAt); err != nil {
		return applicationinventory.Product{}, err
	}

	product, err := getProductByID(ctx, tx, command.Account.BandID, command.ProductID)
	if err != nil {
		return applicationinventory.Product{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return applicationinventory.Product{}, fmt.Errorf("commit inventory product update transaction band_id=%q product_id=%q: %w", command.Account.BandID, command.ProductID, err)
	}

	return product, nil
}

func (repository Repository) UpdateVariant(ctx context.Context, command applicationinventory.UpdateVariantCommand) (applicationinventory.Variant, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return applicationinventory.Variant{}, fmt.Errorf("begin inventory variant update transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var productID string
	var oldQuantity int
	err = tx.QueryRow(ctx, `
		SELECT product_id, quantity
		FROM merch_variants
		WHERE id = $1 AND band_id = $2 AND deleted_at IS NULL
	`, command.VariantID, command.Account.BandID).Scan(&productID, &oldQuantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return applicationinventory.Variant{}, fmt.Errorf("%w: band_id=%q variant_id=%q", applicationinventory.ErrInventoryNotFound, command.Account.BandID, command.VariantID)
	}
	if err != nil {
		return applicationinventory.Variant{}, fmt.Errorf("query inventory variant before update band_id=%q variant_id=%q: %w", command.Account.BandID, command.VariantID, err)
	}

	commandTag, err := tx.Exec(ctx, `
		UPDATE merch_variants
		SET size = $1,
			colour = $2,
			normalized_colour = $3,
			price_amount = $4,
			cost_amount = $5,
			currency = $6,
			quantity = $7,
			updated_at = $8
		WHERE id = $9 AND band_id = $10 AND deleted_at IS NULL
	`, command.Size, command.Colour, command.NormalizedColour, command.Price.Amount, command.Cost.Amount, command.Price.Currency, command.Quantity, command.UpdatedAt, command.VariantID, command.Account.BandID)
	if err != nil {
		return applicationinventory.Variant{}, mapPostgresError(err, fmt.Sprintf("update inventory variant band_id=%q variant_id=%q", command.Account.BandID, command.VariantID))
	}
	if commandTag.RowsAffected() == 0 {
		return applicationinventory.Variant{}, fmt.Errorf("%w: band_id=%q variant_id=%q", applicationinventory.ErrInventoryNotFound, command.Account.BandID, command.VariantID)
	}

	quantityDelta := command.Quantity - oldQuantity
	if quantityDelta != 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO inventory_movements (
				id, band_id, product_id, variant_id, movement_type,
				quantity_delta, quantity_after, reason, actor_user_id, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, uuid.NewString(), command.Account.BandID, productID, command.VariantID, "manual_adjustment", quantityDelta, command.Quantity, "inventory.variant_updated", command.Account.UserID, command.UpdatedAt)
		if err != nil {
			return applicationinventory.Variant{}, fmt.Errorf("insert inventory adjustment movement band_id=%q variant_id=%q: %w", command.Account.BandID, command.VariantID, err)
		}
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "inventory.variant_updated", "merch_variant", command.VariantID, command.RequestID, command.IdempotencyKey, command.UpdatedAt); err != nil {
		return applicationinventory.Variant{}, err
	}

	variant, err := getVariantByID(ctx, tx, command.Account.BandID, command.VariantID)
	if err != nil {
		return applicationinventory.Variant{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return applicationinventory.Variant{}, fmt.Errorf("commit inventory variant update transaction band_id=%q variant_id=%q: %w", command.Account.BandID, command.VariantID, err)
	}

	return variant, nil
}

func (repository Repository) SoftDeleteProduct(ctx context.Context, command applicationinventory.SoftDeleteProductCommand) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin inventory product delete transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	commandTag, err := tx.Exec(ctx, `
		UPDATE merch_products
		SET deleted_at = $1, deleted_by = $2, updated_at = $1
		WHERE id = $3 AND band_id = $4 AND deleted_at IS NULL
	`, command.DeletedAt, command.Account.UserID, command.ProductID, command.Account.BandID)
	if err != nil {
		return fmt.Errorf("soft delete inventory product band_id=%q product_id=%q: %w", command.Account.BandID, command.ProductID, err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: band_id=%q product_id=%q", applicationinventory.ErrInventoryNotFound, command.Account.BandID, command.ProductID)
	}

	_, err = tx.Exec(ctx, `
		UPDATE merch_variants
		SET deleted_at = $1, deleted_by = $2, updated_at = $1
		WHERE product_id = $3 AND band_id = $4 AND deleted_at IS NULL
	`, command.DeletedAt, command.Account.UserID, command.ProductID, command.Account.BandID)
	if err != nil {
		return fmt.Errorf("soft delete inventory product variants band_id=%q product_id=%q: %w", command.Account.BandID, command.ProductID, err)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "inventory.product_deleted", "merch_product", command.ProductID, command.RequestID, command.IdempotencyKey, command.DeletedAt); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit inventory product delete transaction band_id=%q product_id=%q: %w", command.Account.BandID, command.ProductID, err)
	}

	return nil
}

func (repository Repository) SoftDeleteVariant(ctx context.Context, command applicationinventory.SoftDeleteVariantCommand) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin inventory variant delete transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	commandTag, err := tx.Exec(ctx, `
		UPDATE merch_variants
		SET deleted_at = $1, deleted_by = $2, updated_at = $1
		WHERE id = $3 AND band_id = $4 AND deleted_at IS NULL
	`, command.DeletedAt, command.Account.UserID, command.VariantID, command.Account.BandID)
	if err != nil {
		return fmt.Errorf("soft delete inventory variant band_id=%q variant_id=%q: %w", command.Account.BandID, command.VariantID, err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: band_id=%q variant_id=%q", applicationinventory.ErrInventoryNotFound, command.Account.BandID, command.VariantID)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "inventory.variant_deleted", "merch_variant", command.VariantID, command.RequestID, command.IdempotencyKey, command.DeletedAt); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit inventory variant delete transaction band_id=%q variant_id=%q: %w", command.Account.BandID, command.VariantID, err)
	}

	return nil
}

func getProductByID(ctx context.Context, tx pgx.Tx, bandID string, productID string) (applicationinventory.Product, error) {
	row := tx.QueryRow(ctx, `
		SELECT id, band_id, name, normalized_name, category,
			photo_object_key, photo_content_type, photo_size_bytes,
			created_at, updated_at
		FROM merch_products
		WHERE id = $1 AND band_id = $2 AND deleted_at IS NULL
	`, productID, bandID)

	product, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return applicationinventory.Product{}, fmt.Errorf("%w: band_id=%q product_id=%q", applicationinventory.ErrInventoryNotFound, bandID, productID)
	}
	if err != nil {
		return applicationinventory.Product{}, fmt.Errorf("scan inventory product band_id=%q product_id=%q: %w", bandID, productID, err)
	}

	rows, err := tx.Query(ctx, `
		SELECT id, product_id, size, colour, normalized_colour,
			price_amount, cost_amount, currency, quantity, created_at, updated_at
		FROM merch_variants
		WHERE product_id = $1 AND band_id = $2 AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC
	`, productID, bandID)
	if err != nil {
		return applicationinventory.Product{}, fmt.Errorf("query inventory product variants band_id=%q product_id=%q: %w", bandID, productID, err)
	}
	defer rows.Close()

	for rows.Next() {
		variant, err := scanVariant(rows)
		if err != nil {
			return applicationinventory.Product{}, fmt.Errorf("scan inventory product variant band_id=%q product_id=%q: %w", bandID, productID, err)
		}
		product.Variants = append(product.Variants, variant)
	}
	if err := rows.Err(); err != nil {
		return applicationinventory.Product{}, fmt.Errorf("iterate inventory product variants band_id=%q product_id=%q: %w", bandID, productID, err)
	}

	return product, nil
}

func getVariantByID(ctx context.Context, tx pgx.Tx, bandID string, variantID string) (applicationinventory.Variant, error) {
	row := tx.QueryRow(ctx, `
		SELECT id, product_id, size, colour, normalized_colour,
			price_amount, cost_amount, currency, quantity, created_at, updated_at
		FROM merch_variants
		WHERE id = $1 AND band_id = $2 AND deleted_at IS NULL
	`, variantID, bandID)

	variant, err := scanVariant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return applicationinventory.Variant{}, fmt.Errorf("%w: band_id=%q variant_id=%q", applicationinventory.ErrInventoryNotFound, bandID, variantID)
	}
	if err != nil {
		return applicationinventory.Variant{}, fmt.Errorf("scan inventory variant band_id=%q variant_id=%q: %w", bandID, variantID, err)
	}

	return variant, nil
}

func scanProduct(row pgx.Row) (applicationinventory.Product, error) {
	var categoryValue string
	var product applicationinventory.Product
	err := row.Scan(
		&product.ID,
		&product.BandID,
		&product.Name,
		&product.NormalizedName,
		&categoryValue,
		&product.Photo.ObjectKey,
		&product.Photo.ContentType,
		&product.Photo.SizeBytes,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return applicationinventory.Product{}, err
	}

	category, err := inventorydomain.ParseCategory(categoryValue)
	if err != nil {
		return applicationinventory.Product{}, err
	}
	product.Category = category

	return product, nil
}

func scanVariant(row pgx.Row) (applicationinventory.Variant, error) {
	var sizeValue string
	var currency string
	var variant applicationinventory.Variant
	err := row.Scan(
		&variant.ID,
		&variant.ProductID,
		&sizeValue,
		&variant.Colour,
		&variant.NormalizedColour,
		&variant.Price.Amount,
		&variant.Cost.Amount,
		&currency,
		&variant.Quantity,
		&variant.CreatedAt,
		&variant.UpdatedAt,
	)
	if err != nil {
		return applicationinventory.Variant{}, err
	}

	size, err := inventorydomain.ParseSize(sizeValue)
	if err != nil {
		return applicationinventory.Variant{}, err
	}
	variant.Size = size
	variant.Price.Currency = currency
	variant.Cost.Currency = currency

	return variant, nil
}

func insertAuditLog(ctx context.Context, tx pgx.Tx, userID string, bandID string, action string, entityType string, entityID string, requestID string, idempotencyKey string, createdAt time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, band_id, action, entity_type, entity_id, request_id, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, uuid.NewString(), userID, bandID, action, entityType, entityID, requestID, idempotencyKey, createdAt)
	if err != nil {
		return fmt.Errorf("insert inventory audit log user_id=%q band_id=%q action=%q entity_type=%q entity_id=%q: %w", userID, bandID, action, entityType, entityID, err)
	}

	return nil
}

func mapPostgresError(err error, contextMessage string) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("%s: %w", contextMessage, err)
	}

	switch pgErr.ConstraintName {
	case duplicateProductConstraint:
		return fmt.Errorf("%w: %s: %s", applicationinventory.ErrDuplicateProduct, contextMessage, pgErr.Message)
	case duplicateVariantConstraint:
		return fmt.Errorf("%w: %s: %s", applicationinventory.ErrDuplicateVariant, contextMessage, pgErr.Message)
	default:
		return fmt.Errorf("%s: status_code=%q constraint=%q message=%q: %w", contextMessage, pgErr.Code, pgErr.ConstraintName, pgErr.Message, err)
	}
}
