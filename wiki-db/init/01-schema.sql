CREATE EXTENSION IF NOT EXISTS ltree;

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
    slug                TEXT NOT NULL,
    name                TEXT NOT NULL,
    archive_date        DATE,
    deleted_at          TIMESTAMP,
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

-- Hierarchical Categories Table
CREATE TABLE categories (
    id              SERIAL PRIMARY KEY,
    slug            TEXT UNIQUE NOT NULL,
    name            TEXT UNIQUE NOT NULL,
    parent_id       INTEGER REFERENCES categories(id) ON DELETE CASCADE,
    path            LTREE NOT NULL DEFAULT 'root',
    CONSTRAINT chk_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$')
);

-- Category indexes for hierarchical queries
CREATE INDEX idx_categories_path ON categories USING GIST(path);
CREATE INDEX idx_categories_parent ON categories(parent_id);

-- Circular reference prevention trigger
CREATE OR REPLACE FUNCTION check_category_circular_reference()
RETURNS TRIGGER AS $$
BEGIN
    -- Prevent a category from being its own parent
    IF NEW.parent_id = NEW.id THEN
        RAISE EXCEPTION 'Category cannot be its own parent';
    END IF;
    
    -- Prevent circular references via path check
    IF NEW.parent_id IS NOT NULL THEN
        IF EXISTS (
            SELECT 1 FROM categories 
            WHERE id = NEW.parent_id 
            AND NEW.path @> path
        ) THEN
            RAISE EXCEPTION 'Circular reference detected: parent is already a descendant';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_check_category_circular
BEFORE INSERT OR UPDATE ON categories
FOR EACH ROW EXECUTE FUNCTION check_category_circular_reference();

CREATE TABLE page_categories (
    page_id         UUID REFERENCES pages(uuid) ON DELETE CASCADE NOT NULL,
    category        INTEGER REFERENCES categories(id) ON DELETE CASCADE NOT NULL,
    PRIMARY KEY (page_id, category)
);

