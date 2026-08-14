-- Rewrite conversation ids to valid RFC 4122 v4 UUIDs.
-- Legacy ids (newID before the google/uuid switch) were UUID-shaped but
-- carried random version/variant nibbles, and the claude CLI rejects
-- --session-id values that are not valid UUIDs. New ids are built in SQL
-- from randomblob(16) with the version nibble forced to 4 and the variant
-- to 8-b. The temp map keeps conversations and messages consistent.

CREATE TEMP TABLE id_map AS
SELECT id AS old_id,
       substr(h,1,8)||'-'||substr(h,9,4)||'-4'||substr(h,14,3)||'-'||
       substr('89ab', 1 + abs(random() % 4), 1)||substr(h,18,3)||'-'||substr(h,21,12) AS new_id
FROM (SELECT id, lower(hex(randomblob(16))) AS h FROM conversations
      WHERE lower(substr(id,20,1)) NOT IN ('8','9','a','b')
         OR lower(substr(id,15,1)) NOT IN ('1','2','3','4','5','6','7','8'));

UPDATE conversations SET id = (SELECT new_id FROM id_map WHERE old_id = conversations.id)
WHERE id IN (SELECT old_id FROM id_map);

UPDATE messages SET conversation_id = (SELECT new_id FROM id_map WHERE old_id = conversation_id)
WHERE conversation_id IN (SELECT old_id FROM id_map);

DROP TABLE id_map;
