package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS received_messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id     TEXT UNIQUE NOT NULL,
    message_id   TEXT NOT NULL,
    device_id    TEXT NOT NULL DEFAULT '',
    phone_number TEXT NOT NULL,
    message      TEXT NOT NULL,
    sim_number   INTEGER NOT NULL DEFAULT 1,
    received_at  DATETIME NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed    BOOLEAN NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_dedup ON received_messages(phone_number, message, received_at);
CREATE INDEX IF NOT EXISTS idx_received_phone ON received_messages(phone_number);
CREATE INDEX IF NOT EXISTS idx_received_at ON received_messages(received_at);
CREATE INDEX IF NOT EXISTS idx_processed ON received_messages(processed);
`

// ReceivedMessage represents a single received SMS stored in the database.
type ReceivedMessage struct {
	ID          int64     `json:"id"`
	EventID     string    `json:"eventId"`
	MessageID   string    `json:"messageId"`
	DeviceID    string    `json:"deviceId"`
	PhoneNumber string    `json:"phoneNumber"`
	Message     string    `json:"message"`
	SimNumber   int       `json:"simNumber"`
	ReceivedAt  time.Time `json:"receivedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	Processed   bool      `json:"processed"`
}

// ListFilter controls which messages are returned by ListMessages.
type ListFilter struct {
	Phone     string
	Since     *time.Time
	Before    *time.Time
	Processed *bool
	Limit     int
	Offset    int
}

// Stats holds summary statistics about the message store.
type Stats struct {
	Total       int `json:"total"`
	Unprocessed int `json:"unprocessed"`
	Processed   int `json:"processed"`
}

// Store wraps a SQLite database for received message storage.
type Store struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at path, creating parent directories as needed.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveMessage inserts a received message. Duplicate event_ids are silently ignored.
func (s *Store) SaveMessage(msg ReceivedMessage) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO received_messages
			(event_id, message_id, device_id, phone_number, message, sim_number, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msg.EventID, msg.MessageID, msg.DeviceID, msg.PhoneNumber,
		msg.Message, msg.SimNumber, msg.ReceivedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("saving message: %w", err)
	}
	return nil
}

// GetMessage returns a single message by its database ID.
func (s *Store) GetMessage(id int64) (*ReceivedMessage, error) {
	row := s.db.QueryRow(`
		SELECT id, event_id, message_id, device_id, phone_number, message,
		       sim_number, received_at, created_at, processed
		FROM received_messages WHERE id = ?`, id)
	return scanMessage(row)
}

// ListMessages returns messages matching the given filter.
func (s *Store) ListMessages(f ListFilter) ([]ReceivedMessage, error) {
	query := `
		SELECT id, event_id, message_id, device_id, phone_number, message,
		       sim_number, received_at, created_at, processed
		FROM received_messages WHERE 1=1`
	var args []any

	if f.Phone != "" {
		query += " AND phone_number = ?"
		args = append(args, f.Phone)
	}
	if f.Since != nil {
		query += " AND received_at >= ?"
		args = append(args, f.Since.UTC())
	}
	if f.Before != nil {
		query += " AND received_at < ?"
		args = append(args, f.Before.UTC())
	}
	if f.Processed != nil {
		query += " AND processed = ?"
		args = append(args, *f.Processed)
	}

	query += " ORDER BY received_at DESC"

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	} else {
		query += " LIMIT 100"
	}
	if f.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, f.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}
	defer rows.Close()

	var messages []ReceivedMessage
	for rows.Next() {
		var m ReceivedMessage
		var receivedAt, createdAt string
		if err := rows.Scan(&m.ID, &m.EventID, &m.MessageID, &m.DeviceID,
			&m.PhoneNumber, &m.Message, &m.SimNumber,
			&receivedAt, &createdAt, &m.Processed); err != nil {
			return nil, fmt.Errorf("scanning message: %w", err)
		}
		m.ReceivedAt, _ = time.Parse(time.RFC3339, receivedAt)
		if m.ReceivedAt.IsZero() {
			m.ReceivedAt, _ = time.Parse("2006-01-02 15:04:05+00:00", receivedAt)
		}
		if m.ReceivedAt.IsZero() {
			m.ReceivedAt, _ = time.Parse("2006-01-02T15:04:05Z", receivedAt)
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if m.CreatedAt.IsZero() {
			m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// MarkProcessed sets the processed flag on a message.
func (s *Store) MarkProcessed(id int64) error {
	res, err := s.db.Exec("UPDATE received_messages SET processed = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("marking processed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("message %d not found", id)
	}
	return nil
}

// MarkUnprocessed clears the processed flag on a message.
func (s *Store) MarkUnprocessed(id int64) error {
	res, err := s.db.Exec("UPDATE received_messages SET processed = 0 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("marking unprocessed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("message %d not found", id)
	}
	return nil
}

// Stats returns summary counts.
func (s *Store) Stats() (*Stats, error) {
	var st Stats
	err := s.db.QueryRow("SELECT COUNT(*) FROM received_messages").Scan(&st.Total)
	if err != nil {
		return nil, fmt.Errorf("counting total: %w", err)
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM received_messages WHERE processed = 0").Scan(&st.Unprocessed)
	if err != nil {
		return nil, fmt.Errorf("counting unprocessed: %w", err)
	}
	st.Processed = st.Total - st.Unprocessed
	return &st, nil
}

// scanner is implemented by *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanMessage(row scanner) (*ReceivedMessage, error) {
	var m ReceivedMessage
	var receivedAt, createdAt string
	if err := row.Scan(&m.ID, &m.EventID, &m.MessageID, &m.DeviceID,
		&m.PhoneNumber, &m.Message, &m.SimNumber,
		&receivedAt, &createdAt, &m.Processed); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scanning message: %w", err)
	}
	m.ReceivedAt, _ = time.Parse(time.RFC3339, receivedAt)
	if m.ReceivedAt.IsZero() {
		m.ReceivedAt, _ = time.Parse("2006-01-02 15:04:05+00:00", receivedAt)
	}
	if m.ReceivedAt.IsZero() {
		m.ReceivedAt, _ = time.Parse("2006-01-02T15:04:05Z", receivedAt)
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if m.CreatedAt.IsZero() {
		m.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	return &m, nil
}
