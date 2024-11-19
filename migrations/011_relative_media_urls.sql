-- Photos uploaded to object storage used to store an absolute URL
-- (e.g. http://localhost:9000/kfc-media/photos/…), which breaks as soon as
-- the platform is served from another host. Rewrite them to the
-- same-origin /media path that every web app proxies to the bucket.
-- Only rows with an object key are touched; externally hosted URLs stay.
UPDATE menu_items
SET image_url = '/media/' || image_key
WHERE image_key IS NOT NULL AND image_key <> '' AND image_url <> '/media/' || image_key;

UPDATE menu_categories
SET image_url = '/media/' || image_key
WHERE image_key IS NOT NULL AND image_key <> '' AND image_url <> '/media/' || image_key;
