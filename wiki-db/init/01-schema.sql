CREATE TABLE pages (
    uuid                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                TEXT NOT NULL UNIQUE,
    name                TEXT NOT NULL,
    last_revision_id    UUID, -- FK added later
    archive_date        DATE,
    deleted_at          TIMESTAMP
);

CREATE TABLE revisions (
    uuid                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id             UUID REFERENCES pages(uuid) ON DELETE CASCADE NOT NULL,
    date_time           TIMESTAMP DEFAULT now() NOT NULL,
    author              TEXT NOT NULL,
    CONSTRAINT uq_page_timestamp UNIQUE (page_id, date_time)
);

ALTER TABLE pages
ADD CONSTRAINT fk_pages_last_revision
FOREIGN KEY (last_revision_id) REFERENCES revisions(uuid) ON DELETE SET NULL;

CREATE TABLE snapshots (
    uuid                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page                UUID REFERENCES pages(uuid) ON DELETE CASCADE NOT NULL,
    revision            UUID REFERENCES revisions(uuid) ON DELETE CASCADE NOT NULL
);

CREATE OR REPLACE FUNCTION update_last_revision()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE pages SET last_revision_id = NEW.uuid WHERE uuid = NEW.page_id;
    RETURN NEW;
END; $$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_last_revision
AFTER INSERT ON revisions
FOR EACH ROW EXECUTE FUNCTION update_last_revision();

CREATE TABLE categories (
    id              SERIAL PRIMARY KEY,
    slug            TEXT UNIQUE NOT NULL,
    name            TEXT UNIQUE NOT NULL
);

CREATE TABLE page_categories (
    page_id         UUID REFERENCES pages(uuid) ON DELETE CASCADE NOT NULL,
    category        INTEGER REFERENCES categories(id) ON DELETE CASCADE NOT NULL,
    PRIMARY KEY (page_id, category)
);

CREATE TABLE metadata_logs (
    rev_id          UUID REFERENCES revisions(uuid) ON DELETE CASCADE PRIMARY KEY,
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    archive_date    DATE,
    deleted_at      TIMESTAMP
);
