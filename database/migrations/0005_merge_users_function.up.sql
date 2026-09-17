-- Repoints every FK column referencing users(id) from legacy_id to target_id,
-- then deletes the legacy user. Catalog-driven so future tables referencing
-- users(id) are handled automatically without code changes.
CREATE OR REPLACE FUNCTION merge_users(target_id BYTEA, legacy_id BYTEA) RETURNS void AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT
            tc.table_schema AS tbl_schema,
            tc.table_name AS tbl,
            kcu.column_name AS col
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
            ON tc.constraint_catalog = kcu.constraint_catalog
           AND tc.constraint_schema = kcu.constraint_schema
           AND tc.constraint_name = kcu.constraint_name
        JOIN information_schema.constraint_column_usage ccu
            ON tc.constraint_catalog = ccu.constraint_catalog
           AND tc.constraint_schema = ccu.constraint_schema
           AND tc.constraint_name = ccu.constraint_name
        WHERE tc.constraint_type = 'FOREIGN KEY'
          AND ccu.table_schema = 'public'
          AND ccu.table_name = 'users'
          AND ccu.column_name = 'id'
    LOOP
        EXECUTE format('UPDATE %I.%I SET %I = $1 WHERE %I = $2', rec.tbl_schema, rec.tbl, rec.col, rec.col)
            USING target_id, legacy_id;
    END LOOP;

    DELETE FROM public.users WHERE id = legacy_id;
END;
$$ LANGUAGE plpgsql;
