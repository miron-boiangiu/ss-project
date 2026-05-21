CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    email       TEXT PRIMARY KEY,
    password    TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'user'
);

CREATE TABLE devices (
    id            TEXT PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    device_id     TEXT UNIQUE NOT NULL,
    device_name   TEXT NOT NULL DEFAULT '',
    device_status TEXT NOT NULL DEFAULT 'active',
    ip_address    TEXT NOT NULL DEFAULT '',
    port          TEXT NOT NULL DEFAULT '',
    last_seen     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE photos (
    id                        TEXT PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    timestamp                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    image_type                TEXT NOT NULL DEFAULT '',
    device_id                 TEXT NOT NULL DEFAULT '',
    text                      TEXT NOT NULL DEFAULT '',
    unitate_medicala          TEXT NOT NULL DEFAULT '',
    adresa_unitate_medicala   TEXT NOT NULL DEFAULT '',
    telefon_unitate_medicala  TEXT NOT NULL DEFAULT '',
    numar_fisa                TEXT NOT NULL DEFAULT '',
    societate_unitate         TEXT NOT NULL DEFAULT '',
    adresa_angajator          TEXT NOT NULL DEFAULT '',
    telefon_angajator         TEXT NOT NULL DEFAULT '',
    nume                      TEXT NOT NULL DEFAULT '',
    prenume                   TEXT NOT NULL DEFAULT '',
    cnp                       TEXT NOT NULL DEFAULT '',
    profesie_functie          TEXT NOT NULL DEFAULT '',
    loc_de_munca              TEXT NOT NULL DEFAULT '',
    tip_control               TEXT NOT NULL DEFAULT '',
    control_angajare          BOOLEAN NOT NULL DEFAULT FALSE,
    control_periodic          BOOLEAN NOT NULL DEFAULT FALSE,
    control_adaptare          BOOLEAN NOT NULL DEFAULT FALSE,
    control_reluare           BOOLEAN NOT NULL DEFAULT FALSE,
    control_supraveghere      BOOLEAN NOT NULL DEFAULT FALSE,
    control_alte              BOOLEAN NOT NULL DEFAULT FALSE,
    aviz_medical              TEXT NOT NULL DEFAULT '',
    recomandari               TEXT NOT NULL DEFAULT '',
    data                      TEXT NOT NULL DEFAULT '',
    data_urm_examinari        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_photos_timestamp ON photos(timestamp DESC);
CREATE INDEX idx_photos_device_id ON photos(device_id);
CREATE INDEX idx_devices_device_id ON devices(device_id);
