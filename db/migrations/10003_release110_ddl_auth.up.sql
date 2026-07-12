ALTER TABLE sample_comment ADD COLUMN user_id INT;
-- we don't strictly enforce foreign key constraint to users table here since the users table is managed by the Java service and might be in a different schema/db in a real microservice setup, but they are in the same DB here.
-- For simplicity, we just add the column.
UPDATE sample_comment SET user_id = 1;
ALTER TABLE sample_comment ALTER COLUMN user_id SET NOT NULL;
