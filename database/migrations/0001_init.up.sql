CREATE TABLE IF NOT EXISTS locations (
    id BYTEA PRIMARY KEY NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    parent_location_id BYTEA REFERENCES locations (id)
);

CREATE TABLE IF NOT EXISTS products (
        id BYTEA PRIMARY KEY NOT NULL,
        name VARCHAR(128) NOT NULL,
        parent_product_id BYTEA REFERENCES products (id)
    );

CREATE TABLE IF NOT EXISTS samples (
    id BYTEA PRIMARY KEY NOT NULL,
    location_id BYTEA REFERENCES locations (id),
    product_id BYTEA REFERENCES products (id),
    time_registered TIMESTAMP,
    last_update TIMESTAMP,
    state TEXT CHECK (
        state IN (
            'in_use',
            'broken',
            'available',
            'archived',
            'unassigned'
        )
    ) DEFAULT 'unassigned' NOT NULL
);

CREATE TABLE IF NOT EXISTS sample_mods (
    id BYTEA PRIMARY KEY NOT NULL,
    sample_id BYTEA NOT NULL REFERENCES samples (id),
    name VARCHAR(128) NOT NULL,
    time_added TIMESTAMP NOT NULL,
    time_removed TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sample_notes (
    id BYTEA PRIMARY KEY NOT NULL,
    sample_id BYTEA NOT NULL REFERENCES samples (id),
    contents TEXT NOT NULL,
    time_made TIMESTAMP NOT NULL
);
