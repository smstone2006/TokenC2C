-- Preserve custom operator branding; upgrade only this fork's legacy names.
UPDATE settings SET value = 'TokenC2C', updated_at = NOW()
WHERE key = 'site_name' AND value IN ('SubC2C', 'Sub2API');

UPDATE settings SET value = '共享账户 · 按量使用 · 双边计量', updated_at = NOW()
WHERE key = 'site_subtitle'
  AND value IN ('共享 Token 账户池', 'AI API Gateway Platform');
