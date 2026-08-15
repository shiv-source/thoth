-- One Claude CLI session per conversation. New conversations seed their own
-- id (CreateConversation); the backfill below gives legacy conversations
-- their own id as the session (their on-disk CLI sessions are keyed by it),
-- so revisiting old history still resumes.

ALTER TABLE conversations ADD COLUMN claude_session_id TEXT NOT NULL DEFAULT '';

UPDATE conversations SET claude_session_id = id WHERE claude_session_id = '';
