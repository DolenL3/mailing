CREATE TABLE IF NOT EXISTS mailing (
    id uuid NOT NULL PRIMARY KEY,
    text varchar(5000),
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    status integer DEFAULT 0
);

CREATE TABLE IF NOT EXISTS mailing_stats (
    mailing_id uuid REFERENCES mailing(id) ON DELETE CASCADE,
	matches integer,
    sent integer,
    fails integer,
	start_time TIMESTAMP,
    time_executing interval
);

CREATE TABLE IF NOT EXISTS mailing_filter (
	mailing_id uuid REFERENCES mailing(id) ON DELETE CASCADE,
	phone_operator integer,
	tag varchar(100),
	timezone varchar(100)
);

CREATE TABLE IF NOT EXISTS client (
	id serial PRIMARY KEY,
	phone_number bigint,
	phone_operator integer,
	tag varchar(100),
	timezone varchar(100)
);

CREATE TABLE IF NOT EXISTS message (
	id serial PRIMARY KEY,
	time_stamp TIMESTAMP,
	mailing_id uuid REFERENCES mailing(id) ON DELETE CASCADE,
	client_id integer REFERENCES client(id) ON DELETE CASCADE,
	status integer DEFAULT 0
);

INSERT INTO mailing
VALUES ('00000000-0000-0000-0000-000000000000', 'dynamic', '2001-09-11 08:46:00-00', '2001-09-11 10:28:00-00', 6);
