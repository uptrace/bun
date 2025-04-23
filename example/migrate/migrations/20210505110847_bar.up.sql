CREATE TABLE {{.Prefix}}test (
  id bigint PRIMARY KEY
);

--bun:split

ALTER TABLE {{.Prefix}}test ADD COLUMN name varchar(100);
