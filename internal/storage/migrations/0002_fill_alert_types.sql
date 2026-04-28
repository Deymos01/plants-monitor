PRAGMA foreign_keys = ON;

INSERT INTO alert_types (id, type)
VALUES
    (0, 'SOIL_LOW'),
    (1, 'LIGHT_LOW'),
    (2, 'BATTERY_LOW')
ON CONFLICT(id) DO UPDATE SET
    type = excluded.type;