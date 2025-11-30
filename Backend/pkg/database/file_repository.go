package database

import (
	"context"
	"database/sql"
	"time"

	"echofs/internal/metadata"
)

type FileRepository struct {
	db *PostgresDB
}

func NewFileRepository(db *PostgresDB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS files (
		file_id VARCHAR(255) PRIMARY KEY,
		owner_id VARCHAR(255) NOT NULL,
		original_name VARCHAR(500) NOT NULL,
		size BIGINT NOT NULL,
		chunk_size INTEGER NOT NULL,
		total_chunks INTEGER NOT NULL,
		md5_hash VARCHAR(255),
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_files_owner_id ON files(owner_id);
	CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);
	`

	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *FileRepository) CreateFile(ctx context.Context, file *metadata.FileMetadata) error {
	query := `
		INSERT INTO files (file_id, owner_id, original_name, size, chunk_size, total_chunks, md5_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		file.FileID,
		file.OwnerID,
		file.OriginalName,
		file.Size,
		file.ChunkSize,
		file.TotalChunks,
		file.MD5Hash,
		file.Status,
		file.CreatedAt,
		file.UpdatedAt,
	)

	return err
}

func (r *FileRepository) GetFileByID(ctx context.Context, fileID string) (*metadata.FileMetadata, error) {
	file := &metadata.FileMetadata{}
	query := `
		SELECT file_id, owner_id, original_name, size, chunk_size, total_chunks, md5_hash, status, created_at, updated_at
		FROM files
		WHERE file_id = $1
	`

	err := r.db.QueryRowContext(ctx, query, fileID).Scan(
		&file.FileID,
		&file.OwnerID,
		&file.OriginalName,
		&file.Size,
		&file.ChunkSize,
		&file.TotalChunks,
		&file.MD5Hash,
		&file.Status,
		&file.CreatedAt,
		&file.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (r *FileRepository) GetFilesByOwner(ctx context.Context, ownerID string) ([]*metadata.FileMetadata, error) {
	query := `
		SELECT file_id, owner_id, original_name, size, chunk_size, total_chunks, md5_hash, status, created_at, updated_at
		FROM files
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*metadata.FileMetadata
	for rows.Next() {
		file := &metadata.FileMetadata{}
		err := rows.Scan(
			&file.FileID,
			&file.OwnerID,
			&file.OriginalName,
			&file.Size,
			&file.ChunkSize,
			&file.TotalChunks,
			&file.MD5Hash,
			&file.Status,
			&file.CreatedAt,
			&file.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

func (r *FileRepository) DeleteFile(ctx context.Context, fileID, ownerID string) error {
	query := `
		DELETE FROM files
		WHERE file_id = $1 AND owner_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, fileID, ownerID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *FileRepository) UpdateFileStatus(ctx context.Context, fileID, status string) error {
	query := `
		UPDATE files
		SET status = $1, updated_at = $2
		WHERE file_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), fileID)
	return err
}

func (r *FileRepository) CheckFileOwnership(ctx context.Context, fileID, ownerID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(SELECT 1 FROM files WHERE file_id = $1 AND owner_id = $2)
	`

	err := r.db.QueryRowContext(ctx, query, fileID, ownerID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
