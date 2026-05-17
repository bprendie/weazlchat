package storage

import (
	"database/sql"
	"errors"
)

func (s *Store) SaveContextCheckpoint(sessionID string, throughMessageID int64, summary string) error {
	if !s.unlocked {
		return errors.New("database is locked")
	}
	blob, err := s.encrypt(summary)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`insert into context_checkpoints (session_id, through_message_id, summary) values (?, ?, ?)`,
		sessionID, throughMessageID, blob,
	)
	return err
}

func (s *Store) LatestContextCheckpoint(sessionID string) (ContextCheckpoint, bool, error) {
	if !s.unlocked {
		return ContextCheckpoint{}, false, errors.New("database is locked")
	}
	row := s.db.QueryRow(`select id, session_id, through_message_id, summary, created_at from context_checkpoints where session_id = ? order by id desc limit 1`, sessionID)
	var cp ContextCheckpoint
	var enc string
	if err := row.Scan(&cp.ID, &cp.SessionID, &cp.ThroughMessageID, &enc, &cp.CreatedAt); errors.Is(err, sql.ErrNoRows) {
		return ContextCheckpoint{}, false, nil
	} else if err != nil {
		return ContextCheckpoint{}, false, err
	}
	summary, err := s.decrypt(enc)
	if err != nil {
		return ContextCheckpoint{}, false, err
	}
	cp.Summary = summary
	return cp, true, nil
}
