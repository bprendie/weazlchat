package storage

func (s *Store) Migrate() error {
	stmts := []string{
		`create table if not exists vault (
			id integer primary key check (id = 1),
			password_hash text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists sessions (
			id text primary key,
			title text not null,
			provider text not null,
			model text not null,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists messages (
			id integer primary key autoincrement,
			session_id text not null references sessions(id) on delete cascade,
			role text not null,
			content text not null,
			tool_calls text,
			tool_call_id text,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists workspace_saves (
			id integer primary key autoincrement,
			name text not null,
			session_id text not null references sessions(id) on delete cascade,
			snapshot text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create table if not exists memories (
			key text primary key,
			value text not null,
			tags text,
			created_at datetime not null default current_timestamp,
			updated_at datetime not null default current_timestamp
		)`,
		`create table if not exists context_checkpoints (
			id integer primary key autoincrement,
			session_id text not null references sessions(id) on delete cascade,
			through_message_id integer not null,
			summary text not null,
			created_at datetime not null default current_timestamp
		)`,
		`create index if not exists idx_messages_session on messages(session_id, id)`,
		`create index if not exists idx_sessions_updated on sessions(updated_at desc)`,
		`create index if not exists idx_memories_updated on memories(updated_at desc)`,
		`create index if not exists idx_context_checkpoints_session on context_checkpoints(session_id, id desc)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return s.ensureColumns()
}

func (s *Store) ensureColumns() error {
	columns := []struct {
		table string
		name  string
		spec  string
	}{
		{"sessions", "input_tokens", "integer not null default 0"},
		{"sessions", "output_tokens", "integer not null default 0"},
		{"messages", "tool_calls", "text"},
		{"messages", "tool_call_id", "text"},
		{"workspace_saves", "through_message_id", "integer not null default 0"},
	}
	for _, column := range columns {
		if err := s.ensureColumn(column.table, column.name, column.spec); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureColumn(table, name, spec string) error {
	rows, err := s.db.Query(`pragma table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var columnName, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if columnName == name {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(`alter table ` + table + ` add column ` + name + ` ` + spec)
	return err
}
