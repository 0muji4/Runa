-- 0003_diary.down.sql
-- Reverts 0003_diary.up.sql. Dropping the table removes its indexes too.

DROP TABLE IF EXISTS diary_entries;
