--  HISUCON2018 Portal

DROP DATABASE IF EXISTS hisucon2018_portal;
CREATE DATABASE IF NOT EXISTS hisucon2018_portal DEFAULT CHARACTER SET utf8mb4;
USE hisucon2018_portal;

DROP TABLE IF EXISTS bench;

CREATE TABLE bench (
    id              int(11)         NOT NULL AUTO_INCREMENT,
    team            VARCHAR(64)     NOT NULL,
    ipaddress       VARCHAR(64)     NOT NULL,
    result          json            NOT NULL,
    created_at      datetime(6)     NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
