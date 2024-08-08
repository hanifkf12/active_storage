CREATE TABLE users (
                       id uuid PRIMARY KEY,
                       name VARCHAR(255),
                       email VARCHAR(255),
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE blobs (
                       id uuid PRIMARY KEY,
                       key VARCHAR(255) NOT NULL UNIQUE,
                       filename VARCHAR(255) NOT NULL,
                       content_type VARCHAR(255),
                       byte_size BIGINT NOT NULL,
                       checksum VARCHAR(64),
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE attachments (
                             id uuid PRIMARY KEY,
                             name VARCHAR(255) NOT NULL,
                             record_type VARCHAR(255) NOT NULL,
                             record_id uuid NOT NULL,
                             blob_id uuid NOT NULL,
                             created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                             FOREIGN KEY (blob_id) REFERENCES blobs(id) ON DELETE CASCADE
);