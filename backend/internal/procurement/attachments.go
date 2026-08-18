package procurement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dx-os-lab/dx-os/backend/internal/platform/auth"
	"github.com/jackc/pgx/v5"
)

type documentStore interface {
	Put(context.Context, string, string, []byte, string) (string, error)
	Get(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
}

type storedAttachment struct {
	Attachment
	StoragePath string
}

func (s *Store) UploadAttachment(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
	input UploadAttachmentInput,
) (Attachment, error) {
	if hasRole(principal.Roles, "auditor") {
		return Attachment{}, ErrForbidden
	}
	if err := ValidateAttachment(&input); err != nil {
		return Attachment{}, err
	}
	if s.documents == nil {
		return Attachment{}, ErrDocumentStore
	}

	checksumBytes := sha256.Sum256(input.Content)
	checksum := hex.EncodeToString(checksumBytes[:])
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return Attachment{}, fmt.Errorf("begin attachment upload: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return Attachment{}, err
	}
	current, err := lockRequest(ctx, tx, requestID)
	if err != nil {
		return Attachment{}, err
	}
	if current.RequesterID != user.ID ||
		(current.Status != StatusDraft && current.Status != StatusChangesRequested) {
		return Attachment{}, ErrForbidden
	}

	var attachmentID, storagePath string
	err = tx.QueryRow(ctx, `
		WITH generated AS (SELECT gen_random_uuid() AS id)
		INSERT INTO purchase_request_attachments (
			id,
			purchase_request_id,
			document_type,
			original_name,
			content_type,
			size_bytes,
			checksum_sha256,
			storage_path,
			uploaded_by,
			status
		)
		SELECT
			id,
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			$6,
			'purchase-requests/' || ($1::uuid)::text || '/' || id::text,
			$7,
			'UPLOADING'
		FROM generated
		RETURNING id, storage_path
	`, requestID, input.DocumentType, input.FileName, input.ContentType,
		len(input.Content), checksum, user.ID).Scan(&attachmentID, &storagePath)
	if err != nil {
		return Attachment{}, fmt.Errorf("insert attachment metadata: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Attachment{}, fmt.Errorf("commit attachment metadata: %w", err)
	}

	etag, err := s.documents.Put(
		ctx, storagePath, input.ContentType, input.Content, checksum,
	)
	if err != nil {
		_, _ = s.database.Exec(
			context.WithoutCancel(ctx),
			`DELETE FROM purchase_request_attachments WHERE id = $1 AND status = 'UPLOADING'`,
			attachmentID,
		)
		return Attachment{}, fmt.Errorf("%w: %v", ErrDocumentStore, err)
	}

	finalizeTx, err := s.database.Begin(ctx)
	if err != nil {
		s.cleanupUploadingAttachment(ctx, attachmentID, storagePath)
		return Attachment{}, fmt.Errorf("begin attachment finalize: %w", err)
	}
	defer func() { _ = finalizeTx.Rollback(ctx) }()
	var attachment Attachment
	err = finalizeTx.QueryRow(ctx, `
		UPDATE purchase_request_attachments pa
		SET status = 'ACTIVE',
			storage_etag = NULLIF($2, ''),
			uploaded_at = now(),
			updated_at = now()
		FROM users u
		WHERE pa.id = $1
		  AND pa.uploaded_by = u.id
		  AND pa.status = 'UPLOADING'
		RETURNING
			pa.id,
			pa.purchase_request_id,
			pa.document_type,
			pa.original_name,
			pa.content_type,
			pa.size_bytes,
			pa.checksum_sha256,
			pa.uploaded_by,
			u.display_name,
			pa.uploaded_at
	`, attachmentID, etag).Scan(
		&attachment.ID,
		&attachment.PurchaseID,
		&attachment.DocumentType,
		&attachment.FileName,
		&attachment.ContentType,
		&attachment.SizeBytes,
		&attachment.ChecksumSHA256,
		&attachment.UploadedBy,
		&attachment.UploadedByName,
		&attachment.UploadedAt,
	)
	if err != nil {
		_ = finalizeTx.Rollback(ctx)
		s.cleanupUploadingAttachment(ctx, attachmentID, storagePath)
		return Attachment{}, fmt.Errorf("finalize attachment metadata: %w", err)
	}
	if err = insertAudit(
		ctx, finalizeTx, requestID, "ATTACHMENT_UPLOADED", user.ID,
		principal.Roles, string(current.Status), current.Status, input.CorrelationID,
	); err != nil {
		_ = finalizeTx.Rollback(ctx)
		s.cleanupUploadingAttachment(ctx, attachmentID, storagePath)
		return Attachment{}, err
	}
	if err = finalizeTx.Commit(ctx); err != nil {
		s.cleanupUploadingAttachment(ctx, attachmentID, storagePath)
		return Attachment{}, fmt.Errorf("commit attachment finalize: %w", err)
	}
	return attachment, nil
}

func (s *Store) cleanupUploadingAttachment(
	ctx context.Context,
	attachmentID string,
	storagePath string,
) {
	cleanupContext := context.WithoutCancel(ctx)
	_ = s.documents.Delete(cleanupContext, storagePath)
	_, _ = s.database.Exec(
		cleanupContext,
		`DELETE FROM purchase_request_attachments
		 WHERE id = $1 AND status = 'UPLOADING'`,
		attachmentID,
	)
}

func (s *Store) ListAttachments(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
) (AttachmentList, error) {
	request, err := s.Get(ctx, principal, requestID)
	if err != nil {
		return AttachmentList{}, err
	}
	rows, err := s.database.Query(ctx, `
		SELECT
			pa.id,
			pa.purchase_request_id,
			pa.document_type,
			pa.original_name,
			pa.content_type,
			pa.size_bytes,
			pa.checksum_sha256,
			pa.uploaded_by,
			u.display_name,
			pa.uploaded_at
		FROM purchase_request_attachments pa
		JOIN users u ON u.id = pa.uploaded_by
		WHERE pa.purchase_request_id = $1
		  AND pa.status = 'ACTIVE'
		ORDER BY pa.uploaded_at DESC, pa.id
	`, requestID)
	if err != nil {
		return AttachmentList{}, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	result := AttachmentList{
		Items:               make([]Attachment, 0),
		MaxSizeBytes:        MaxAttachmentSize,
		AllowedContentTypes: append([]string(nil), AllowedAttachmentContentTypes...),
	}
	for rows.Next() {
		var attachment Attachment
		if err = rows.Scan(
			&attachment.ID,
			&attachment.PurchaseID,
			&attachment.DocumentType,
			&attachment.FileName,
			&attachment.ContentType,
			&attachment.SizeBytes,
			&attachment.ChecksumSHA256,
			&attachment.UploadedBy,
			&attachment.UploadedByName,
			&attachment.UploadedAt,
		); err != nil {
			return AttachmentList{}, fmt.Errorf("scan attachment: %w", err)
		}
		result.Items = append(result.Items, attachment)
	}
	if err = rows.Err(); err != nil {
		return AttachmentList{}, fmt.Errorf("iterate attachments: %w", err)
	}

	var requiredType DocumentType
	var threshold string
	err = s.database.QueryRow(ctx, `
		SELECT ar.required_document_type, ar.threshold_amount::text
		FROM attachment_rules ar
		JOIN departments d ON d.organization_id = ar.organization_id
		WHERE d.id = $1
		  AND ar.currency = $2
		  AND ar.active
		LIMIT 1
	`, request.DepartmentID, request.Currency).Scan(&requiredType, &threshold)
	if errors.Is(err, pgx.ErrNoRows) {
		result.RequirementMet = true
		return result, nil
	}
	if err != nil {
		return AttachmentList{}, fmt.Errorf("get attachment rule: %w", err)
	}
	result.RequiredDocumentType = requiredType
	result.ThresholdAmount = threshold
	var required bool
	if err = s.database.QueryRow(
		ctx,
		`SELECT $1::numeric >= $2::numeric`,
		request.TotalAmount,
		threshold,
	).Scan(&required); err != nil {
		return AttachmentList{}, fmt.Errorf("evaluate attachment rule: %w", err)
	}
	result.Required = required
	result.RequirementMet = !required
	for _, attachment := range result.Items {
		if attachment.DocumentType == requiredType {
			result.RequirementMet = true
			break
		}
	}
	return result, nil
}

func (s *Store) DownloadAttachment(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
	attachmentID string,
) (AttachmentContent, error) {
	if _, err := s.Get(ctx, principal, requestID); err != nil {
		return AttachmentContent{}, err
	}
	attachment, err := s.getStoredAttachment(ctx, requestID, attachmentID)
	if err != nil {
		return AttachmentContent{}, err
	}
	content, err := s.documents.Get(ctx, attachment.StoragePath)
	if err != nil {
		return AttachmentContent{}, fmt.Errorf("%w: %v", ErrDocumentStore, err)
	}
	checksumBytes := sha256.Sum256(content)
	if hex.EncodeToString(checksumBytes[:]) != attachment.ChecksumSHA256 {
		return AttachmentContent{}, fmt.Errorf("%w: checksum mismatch", ErrDocumentStore)
	}
	return AttachmentContent{Attachment: attachment.Attachment, Content: content}, nil
}

func (s *Store) DeleteAttachment(
	ctx context.Context,
	principal auth.Principal,
	requestID string,
	attachmentID string,
	correlationID string,
) error {
	if hasRole(principal.Roles, "auditor") {
		return ErrForbidden
	}
	tx, err := s.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin attachment delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	user, err := ensureUser(ctx, tx, principal)
	if err != nil {
		return err
	}
	current, err := lockRequest(ctx, tx, requestID)
	if err != nil {
		return err
	}
	if current.RequesterID != user.ID ||
		(current.Status != StatusDraft && current.Status != StatusChangesRequested) {
		return ErrForbidden
	}
	var storagePath string
	err = tx.QueryRow(ctx, `
		UPDATE purchase_request_attachments
		SET status = 'DELETING', updated_at = now()
		WHERE id = $1
		  AND purchase_request_id = $2
		  AND status = 'ACTIVE'
		RETURNING storage_path
	`, attachmentID, requestID).Scan(&storagePath)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAttachmentNotFound
	}
	if err != nil {
		return fmt.Errorf("mark attachment deleting: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit attachment deleting: %w", err)
	}

	if err = s.documents.Delete(ctx, storagePath); err != nil {
		_, _ = s.database.Exec(
			context.WithoutCancel(ctx),
			`UPDATE purchase_request_attachments
			 SET status = 'ACTIVE', updated_at = now()
			 WHERE id = $1 AND status = 'DELETING'`,
			attachmentID,
		)
		return fmt.Errorf("%w: %v", ErrDocumentStore, err)
	}

	finalizeTx, err := s.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin attachment delete finalize: %w", err)
	}
	defer func() { _ = finalizeTx.Rollback(ctx) }()
	command, err := finalizeTx.Exec(ctx, `
		UPDATE purchase_request_attachments
		SET status = 'DELETED',
			deleted_at = now(),
			updated_at = now()
		WHERE id = $1
		  AND status = 'DELETING'
	`, attachmentID)
	if err != nil {
		return fmt.Errorf("finalize attachment delete: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrAttachmentNotFound
	}
	if err = insertAudit(
		ctx, finalizeTx, requestID, "ATTACHMENT_DELETED", user.ID,
		principal.Roles, string(current.Status), current.Status, correlationID,
	); err != nil {
		return err
	}
	if err = finalizeTx.Commit(ctx); err != nil {
		return fmt.Errorf("commit attachment delete: %w", err)
	}
	return nil
}

func (s *Store) getStoredAttachment(
	ctx context.Context,
	requestID string,
	attachmentID string,
) (storedAttachment, error) {
	var attachment storedAttachment
	err := s.database.QueryRow(ctx, `
		SELECT
			pa.id,
			pa.purchase_request_id,
			pa.document_type,
			pa.original_name,
			pa.content_type,
			pa.size_bytes,
			pa.checksum_sha256,
			pa.uploaded_by,
			u.display_name,
			pa.uploaded_at,
			pa.storage_path
		FROM purchase_request_attachments pa
		JOIN users u ON u.id = pa.uploaded_by
		WHERE pa.id = $1
		  AND pa.purchase_request_id = $2
		  AND pa.status = 'ACTIVE'
	`, attachmentID, requestID).Scan(
		&attachment.ID,
		&attachment.PurchaseID,
		&attachment.DocumentType,
		&attachment.FileName,
		&attachment.ContentType,
		&attachment.SizeBytes,
		&attachment.ChecksumSHA256,
		&attachment.UploadedBy,
		&attachment.UploadedByName,
		&attachment.UploadedAt,
		&attachment.StoragePath,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedAttachment{}, ErrAttachmentNotFound
	}
	if err != nil {
		return storedAttachment{}, fmt.Errorf("get attachment: %w", err)
	}
	return attachment, nil
}

func requireAttachmentForSubmission(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	request lockedRequest,
) error {
	var requiredType DocumentType
	err := tx.QueryRow(ctx, `
		SELECT ar.required_document_type
		FROM attachment_rules ar
		WHERE ar.organization_id = $1
		  AND ar.currency = $2
		  AND ar.active
		  AND $3::numeric >= ar.threshold_amount
		LIMIT 1
	`, request.OrganizationID, request.Currency, request.TotalAmount).Scan(&requiredType)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("evaluate required attachment: %w", err)
	}
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM purchase_request_attachments
			WHERE purchase_request_id = $1
			  AND document_type = $2
			  AND status = 'ACTIVE'
		)
	`, requestID, requiredType).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check required attachment: %w", err)
	}
	if !exists {
		return ErrAttachmentRequired
	}
	return nil
}
