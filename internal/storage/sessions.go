package storage

func (s *Store) CreateSession(id, title, provider, model string) error {
	_, err := s.db.Exec(
		`insert into sessions (id, title, provider, model, updated_at) values (?, ?, ?, ?, current_timestamp)`,
		id, title, provider, model,
	)
	return err
}

func (s *Store) TouchSession(id, title string) error {
	_, err := s.db.Exec(`update sessions set title = ?, updated_at = current_timestamp where id = ?`, title, id)
	return err
}

func (s *Store) AddSessionTokens(id string, inputTokens, outputTokens int) error {
	_, err := s.db.Exec(
		`update sessions
		 set input_tokens = input_tokens + ?,
		     output_tokens = output_tokens + ?,
		     updated_at = current_timestamp
		 where id = ?`,
		inputTokens, outputTokens, id,
	)
	return err
}

func (s *Store) LatestSession() (Session, bool, error) {
	rows, err := s.db.Query(`select id, title, provider, model, input_tokens, output_tokens, created_at, updated_at from sessions order by updated_at desc limit 1`)
	if err != nil {
		return Session{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Session{}, false, rows.Err()
	}
	sess, err := scanSession(rows)
	return sess, err == nil, err
}

func (s *Store) ListSessions(limit int) ([]Session, error) {
	rows, err := s.db.Query(`select id, title, provider, model, input_tokens, output_tokens, created_at, updated_at from sessions order by updated_at desc limit ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *Store) Session(id string) (Session, bool, error) {
	rows, err := s.db.Query(`select id, title, provider, model, input_tokens, output_tokens, created_at, updated_at from sessions where id = ?`, id)
	if err != nil {
		return Session{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Session{}, false, rows.Err()
	}
	sess, err := scanSession(rows)
	return sess, err == nil, err
}

func (s *Store) DeleteSession(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`delete from messages where session_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`delete from workspace_saves where session_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`delete from sessions where id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func scanSession(rows interface {
	Scan(dest ...any) error
}) (Session, error) {
	var sess Session
	return sess, rows.Scan(&sess.ID, &sess.Title, &sess.Provider, &sess.Model, &sess.InputTokens, &sess.OutputTokens, &sess.CreatedAt, &sess.UpdatedAt)
}
