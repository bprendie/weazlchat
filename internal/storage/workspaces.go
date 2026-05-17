package storage

import (
	"errors"
	"strings"
)

func (s *Store) SaveWorkspace(name, sessionID, snapshot string, throughMessageID int64) (int64, error) {
	if !s.unlocked {
		return 0, errors.New("database is locked")
	}
	blob, err := s.encrypt(snapshot)
	if err != nil {
		return 0, err
	}
	result, err := s.db.Exec(
		`insert into workspace_saves (name, session_id, snapshot, through_message_id) values (?, ?, ?, ?)`,
		name, sessionID, blob, throughMessageID,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) RenameWorkspace(id int64, name string) error {
	if !s.unlocked {
		return errors.New("database is locked")
	}
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return errors.New("workspace name is required")
	}
	result, err := s.db.Exec(`update workspace_saves set name = ? where id = ?`, name, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("workspace save not found")
	}
	return nil
}

func (s *Store) UpdateWorkspace(id int64, sessionID, snapshot string, throughMessageID int64) error {
	if !s.unlocked {
		return errors.New("database is locked")
	}
	blob, err := s.encrypt(snapshot)
	if err != nil {
		return err
	}
	result, err := s.db.Exec(
		`update workspace_saves set session_id = ?, snapshot = ?, through_message_id = ? where id = ?`,
		sessionID, blob, throughMessageID, id,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("workspace save not found")
	}
	return nil
}

func (s *Store) DeleteWorkspace(id int64) error {
	if !s.unlocked {
		return errors.New("database is locked")
	}
	result, err := s.db.Exec(`delete from workspace_saves where id = ?`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("workspace save not found")
	}
	return nil
}

func (s *Store) WorkspaceSaves(limit int) ([]WorkspaceSave, error) {
	rows, err := s.db.Query(`select id, name, session_id, through_message_id, created_at from workspace_saves order by created_at desc limit ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var saves []WorkspaceSave
	for rows.Next() {
		var save WorkspaceSave
		if err := rows.Scan(&save.ID, &save.Name, &save.SessionID, &save.ThroughMessageID, &save.CreatedAt); err != nil {
			return nil, err
		}
		saves = append(saves, save)
	}
	return saves, rows.Err()
}
