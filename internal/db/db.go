package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

type UploadedFile struct {
	ID        int64
	FilePath  string
	FileHash  string
	GoogleID  string
	Timestamp string
}

type UploadStats struct {
	TotalUploaded  int64
	LastUploadTime *time.Time
}

type UploadError struct {
	File    string
	Message string
	Time    time.Time
}

func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS uploaded_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		file_hash TEXT NOT NULL,
		google_id TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(file_hash)
	);
	CREATE INDEX IF NOT EXISTS idx_file_hash ON uploaded_files(file_hash);`

	_, err := db.Exec(schema)
	return err
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) IsFileUploaded(fileHash string) (bool, error) {
	var exists bool
	err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM uploaded_files WHERE file_hash = ?)", fileHash).Scan(&exists)
	return exists, err
}

func (d *DB) SaveUploadedFile(file *UploadedFile) error {
	_, err := d.db.Exec(
		"INSERT INTO uploaded_files (file_path, file_hash, google_id) VALUES (?, ?, ?)",
		file.FilePath, file.FileHash, file.GoogleID,
	)
	return err
}

func (d *DB) GetUploadedFiles() ([]UploadedFile, error) {
	rows, err := d.db.Query("SELECT id, file_path, file_hash, google_id, timestamp FROM uploaded_files")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []UploadedFile
	for rows.Next() {
		var file UploadedFile
		err := rows.Scan(&file.ID, &file.FilePath, &file.FileHash, &file.GoogleID, &file.Timestamp)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (d *DB) GetUploadStats() (*UploadStats, error) {
	stats := &UploadStats{}

	// Get total count
	err := d.db.QueryRow("SELECT COUNT(*) FROM uploaded_files").Scan(&stats.TotalUploaded)
	if err != nil {
		return nil, err
	}

	// Get last upload time
	var lastTime sql.NullString
	err = d.db.QueryRow("SELECT MAX(timestamp) FROM uploaded_files").Scan(&lastTime)
	if err != nil {
		return nil, err
	}

	if lastTime.Valid {
		t, err := time.Parse("2006-01-02 15:04:05", lastTime.String)
		if err != nil {
			return nil, err
		}
		stats.LastUploadTime = &t
	}

	return stats, nil
}

func (d *DB) GetPendingFiles() ([]string, error) {
	// Get files that have been imported but not uploaded (google_id is NULL)
	rows, err := d.db.Query("SELECT file_path FROM uploaded_files WHERE google_id IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		files = append(files, path)
	}
	return files, nil
}

func (d *DB) GetRecentErrors() ([]UploadError, error) {
	// This is a placeholder implementation. In a real implementation,
	// you would store and retrieve errors from a dedicated errors table.
	return []UploadError{}, nil
}
