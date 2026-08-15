-- messages — the chat transcript, persisted for the UI history view.
--
-- id              autoincrement; messages are replayed in id order, so a
--                 conversation reads chronologically
-- conversation_id references conversations(id)
-- role            'user' | 'assistant' (the CLI's tool chatter is NOT
--                 stored — only the final answers and the user prompts)
-- content         plain markdown text
-- created_at      UTC RFC3339 (same lexical-ordering contract as
--                 conversations.created_at)

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS messages_conv_idx ON messages(conversation_id, id);
