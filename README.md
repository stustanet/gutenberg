# Gutenberg

Gutenberg is a printing service system consisting of an upload web service and an administration web service.

## systemd socket activation

Gutenberg relies on systemd socket activation for automatic restarts on failures and also to be able to listen on port 80 with an unprivileged user (www-print by default).
The www-admin socket and service are started by ipsec after the tunnel was established, see `scripts/www-admin_change.sh` and the `ipsec.conf`.

Copy the files in `systemd` to the systemd unit config folder (on Debian: `/etc/systemd/system/`) and modify the config files as needed.

Afterwards, run
```sh
systemctl enable www-admin.socket www-print-http.socket www-print-https.socket
```

And to start the services, run:
```sh
systemctl start www-admin.socket www-admin.service www-print-http.socket www-print-https.socket www-print.service
```

## SQL Schema
Gutenberg uses Postgres as a database. The following schema is required:

```sql
CREATE SEQUENCE job_id_seq;

CREATE TYPE duplex_t AS ENUM('simplex', 'short', 'long');
 
CREATE TABLE job (
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
        pages SMALLINT NOT NULL,
        sheets SMALLINT NOT NULL,
        price DOUBLE PRECISION NOT NULL,
        copies SMALLINT DEFAULT 1,
        date TIMESTAMP WITH TIME ZONE DEFAULT statement_timestamp(),
        error TEXT
);

CREATE SEQUENCE log_id_seq;

CREATE TABLE log (
        id INTEGER UNIQUE PRIMARY KEY DEFAULT NEXTVAL('log_id_seq'),
        internal BOOLEAN NOT NULL,
        bw BOOLEAN NOT NULL,
        cyan DOUBLE PRECISION,
        magenta DOUBLE PRECISION,
        yellow DOUBLE PRECISION,
        key DOUBLE PRECISION,
        duplex duplex_t NOT NULL,
        pages SMALLINT NOT NULL,
        sheets SMALLINT NOT NULL,
        price DOUBLE PRECISION NOT NULL,
        copies SMALLINT DEFAULT 1,
        create_date TIMESTAMP WITH TIME ZONE NOT NULL,
        print_date TIMESTAMP WITH TIME ZONE DEFAULT statement_timestamp(),
        error TEXT
);
```
