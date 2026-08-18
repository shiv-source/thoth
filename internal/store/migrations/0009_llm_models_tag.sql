-- description → tag. 0008 shipped with the column named description while
-- the feature was in development; the final name is tag (the UI offers it
-- as a dropdown with free entry, rendered as a colored chip). Applied
-- migrations are immutable, so the rename is its own migration rather than
-- an edit to 0008.

ALTER TABLE llm_models RENAME COLUMN description TO tag;
