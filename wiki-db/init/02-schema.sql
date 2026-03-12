-- JOINs here
/*
CREATE VIEW page_snapshot AS 
SELECT snapshots.uuid AS snapshot, revision, page_id AS page
FROM snapshots
JOIN revisions ON snapshots.revision = revisions.uuid;
*/

CREATE VIEW page_category_slugs AS
SELECT pages.slug AS page, categories.slug AS category
FROM page_categories
JOIN pages ON pages.uuid = page_categories.page_id
JOIN categories ON categories.id = page_categories.category;

-- Helper view for hierarchical categories with full path information
CREATE VIEW page_category_full AS
SELECT 
    pc.page_id,
    c.id AS category_id,
    c.slug AS category_slug,
    c.name AS category_name,
    c.path,
    c.parent_id,
    CASE 
        WHEN c.parent_id IS NULL THEN c.slug
        ELSE (SELECT slug FROM categories WHERE id = c.parent_id) || '/' || c.slug
    END AS full_slug
FROM page_categories pc
JOIN categories c ON pc.category = c.id;
