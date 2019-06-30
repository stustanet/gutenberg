-- gutenberg database
--

CREATE SEQUENCE job_id_seq;

CREATE TYPE duplex_t AS ENUM('simplex', 'short', 'long');

CREATE TYPE format_t AS ENUM('A5', 'A4', 'A3');

CREATE TABLE if not exists job (
        id integer UNIQUE PRIMARY KEY DEFAULT NEXTVAL('job_id_seq'),
        file_id char(9) NOT NULL,
        pin char(6) UNIQUE NOT NULL,
        ip_address varchar(50) NOT NULL,
        bw boolean NOT NULL,
        cyan DOUBLE PRECISION NOT NULL,
        magenta DOUBLE PRECISION NOT NULL,
        yellow DOUBLE PRECISION NOT NULL,
        key DOUBLE PRECISION NOT NULL,
        duplex duplex_t NOT NULL,
        format format_t NOT NULL,
        pages SMALLINT NOT NULL,
        sheets SMALLINT NOT NULL,
        price DOUBLE PRECISION NOT NULL,
        copies SMALLINT DEFAULT 1,
        date TIMESTAMP WITH TIME ZONE DEFAULT statement_timestamp(),
        error TEXT
);

CREATE SEQUENCE log_id_seq;

CREATE TABLE if not exists log (
        id INTEGER UNIQUE PRIMARY KEY DEFAULT NEXTVAL('log_id_seq'),
        internal BOOLEAN NOT NULL,
        bw BOOLEAN NOT NULL,
        cyan DOUBLE PRECISION,
        magenta DOUBLE PRECISION,
        yellow DOUBLE PRECISION,
        key DOUBLE PRECISION,
        duplex duplex_t NOT NULL,
        format format_t NOT NULL,
        pages SMALLINT NOT NULL,
        sheets SMALLINT NOT NULL,
        price DOUBLE PRECISION NOT NULL,
        copies SMALLINT DEFAULT 1,
        create_date TIMESTAMP WITH TIME ZONE NOT NULL,
        print_date TIMESTAMP WITH TIME ZONE DEFAULT statement_timestamp(),
        error TEXT
);
