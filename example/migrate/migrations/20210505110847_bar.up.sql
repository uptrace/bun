CREATE TABLE test (
  id bigint PRIMARY KEY
);

--bun:split

ALTER TABLE test ADD COLUMN name varchar(100);
