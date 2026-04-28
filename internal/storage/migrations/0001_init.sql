PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS telegram_users
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id    INTEGER  NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NULL
);

CREATE TABLE IF NOT EXISTS devices
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id    TEXT     NOT NULL UNIQUE,
    token_hash   TEXT     NOT NULL,
    plant_name   TEXT     NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME NULL
);

CREATE TABLE IF NOT EXISTS alert_types
(
    id   INTEGER PRIMARY KEY,
    type TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS device_subscriptions
(
    telegram_user_id INTEGER  NOT NULL,
    device_id        INTEGER  NOT NULL,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (telegram_user_id, device_id),

    FOREIGN KEY (telegram_user_id)
        REFERENCES telegram_users (id)
        ON DELETE CASCADE,

    FOREIGN KEY (device_id)
        REFERENCES devices (id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_device_subscriptions_device_id
    ON device_subscriptions (device_id);

CREATE TABLE IF NOT EXISTS measurements
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id       INTEGER  NOT NULL,

    soil_raw        REAL     NOT NULL DEFAULT 0,
    soil_voltage    REAL     NOT NULL DEFAULT 0,
    soil_percent    INTEGER  NOT NULL DEFAULT 0
        CHECK (soil_percent >= 0 AND soil_percent <= 100),

    light_raw       REAL     NOT NULL DEFAULT 0,
    light_voltage   REAL     NOT NULL DEFAULT 0,

    battery_voltage REAL     NULL,

    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id)
        REFERENCES devices (id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_measurements_device_created
    ON measurements (device_id, created_at DESC);

CREATE TABLE IF NOT EXISTS alerts
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id   INTEGER  NOT NULL,
    type_id     INTEGER  NOT NULL,
    status      TEXT     NOT NULL CHECK (status IN ('ACTIVE', 'RESOLVED')),
    message     TEXT     NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME NULL,

    FOREIGN KEY (device_id)
        REFERENCES devices (id)
        ON DELETE CASCADE,

    FOREIGN KEY (type_id)
        REFERENCES alert_types (id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_alerts_device_type_status
    ON alerts (device_id, type_id, status);