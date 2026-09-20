-- ============================================================================
-- Migration (manual / one-off) — work_modality em applications
-- ============================================================================
-- Contexto:
--   Antes de c3493a0, a modalidade de trabalho era gravada DENTRO do campo
--   `location`:
--     • Front antigo: select com os valores EXATOS "Remote" / "Hybrid" /
--       "On-site" (gravados no campo location);
--     • MCP antigo: texto livre ("Local da vaga (cidade, remoto, híbrido,
--       etc.)") — por isso existem variações informais no banco.
--   O commit c3493a0 criou a coluna `work_modality` (via AutoMigrate), que fica
--   NULL nas linhas antigas, e adicionou validação (oneof + IsValid) + tag
--   `check` no model GORM.
--
--   Este script:
--     1) Faz o backfill de `work_modality` a partir de `location`:
--        normaliza (case-insensitive) os valores que se aplicam — ex.:
--        "On-Site" → ONSITE, "Home Office" → REMOTE, "Híbrido" → HYBRID —
--        e DEIXA VAZIO (NULL) os que não dá para normalizar (cidade,
--        texto livre, etc.).
--     2) Cria de fato a check constraint no banco (o AutoMigrate do GORM
--        não adiciona check em tabelas já existentes — só no CREATE TABLE).
--     3) Remove a coluna legada `location` (o backend novo não a usa).
--
-- Modo de uso no Railway (rodar logo APÓS o deploy do backend novo):
--   railway run --service hirely-api psql "$DATABASE_URL" -f migrations/001_backfill_work_modality.sql
--   (ou cole o conteúdo no painel SQL do Railway)
-- ============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- 0) Sanity checks (opcionais, rode ANTES do passo 1 se quiser revisar):
--    0a) Deve retornar 0 linhas antes da constraint (passo 2). Se retornar,
--        ajuste esses valores antes de rodar o script.
--    0b) Dry-run do backfill: mostra quantas linhas legadas cada modalidade
--        vai receber (e quantas ficam NULL porque não casaram com nada).
-- ---------------------------------------------------------------------------
-- SELECT id, work_modality FROM applications
-- WHERE work_modality IS NOT NULL
--   AND work_modality NOT IN ('REMOTE', 'HYBRID', 'ONSITE');

-- SELECT CASE
--     WHEN lower(trim(location)) ~ 'hibrido|híbrido|hybrid|semi[[:space:]-]*presencial'                        THEN 'HYBRID'
--     WHEN lower(trim(location)) ~ 'remot|home[[:space:]-]*office|wfh|work[[:space:]-]*from[[:space:]-]*home|teletrabalh' THEN 'REMOTE'
--     WHEN lower(trim(location)) ~ 'presencial|onsite|on[[:space:]-]*site|in[[:space:]-]*office|escrit'        THEN 'ONSITE'
--     ELSE NULL
--   END AS mapped_modality,
--   count(*)
-- FROM applications
-- WHERE work_modality IS NULL
--   AND trim(location) <> ''
-- GROUP BY 1;

-- ---------------------------------------------------------------------------
-- 1) Backfill — normaliza `work_modality` a partir do `location` legado.
--    • Case-insensitive (lower + regex): aceita "On-Site", "ONSITE", "remoto",
--      "Home Office", "Híbrido", "Escritório", "WFH", etc.
--    • Prioridade quando a string cita mais de uma modalidade:
--      HYBRID > REMOTE > ONSITE (ex.: "Híbrido (2 dias remoto)" → HYBRID).
--    • O que NÃO casa (cidade, texto livre, vazio) fica NULL — "deixa vazio".
--      NULL, e não '' — string vazia violaria a check constraint do passo 2.
--    • Só toca linhas legadas (work_modality IS NULL); linhas novas já têm o
--      valor correto.
--    • O DO block guarda o UPDATE: se a coluna já foi dropada num run anterior,
--      o script continua sem erro (idempotente).
-- ---------------------------------------------------------------------------
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'applications' AND column_name = 'location'
  ) THEN
    UPDATE applications
    SET work_modality = CASE
        WHEN lower(trim(location)) ~ 'hibrido|híbrido|hybrid|semi[[:space:]-]*presencial'                        THEN 'HYBRID'
        WHEN lower(trim(location)) ~ 'remot|home[[:space:]-]*office|wfh|work[[:space:]-]*from[[:space:]-]*home|teletrabalh' THEN 'REMOTE'
        WHEN lower(trim(location)) ~ 'presencial|onsite|on[[:space:]-]*site|in[[:space:]-]*office|escrit'        THEN 'ONSITE'
    END
    WHERE work_modality IS NULL
      AND trim(location) <> ''
      AND lower(trim(location)) ~ 'hibrido|híbrido|hybrid|semi[[:space:]-]*presencial|remot|home[[:space:]-]*office|wfh|work[[:space:]-]*from[[:space:]-]*home|teletrabalh|presencial|onsite|on[[:space:]-]*site|in[[:space:]-]*office|escrit';
  END IF;
END $$;

-- ---------------------------------------------------------------------------
-- 2) Check constraint real (DROP IF EXISTS + ADD torna o script idempotente).
--    Permite NULL (linhas sem modalidade informada).
-- ---------------------------------------------------------------------------
ALTER TABLE applications DROP CONSTRAINT IF EXISTS chk_applications_work_modality;
ALTER TABLE applications
  ADD CONSTRAINT chk_applications_work_modality
  CHECK (work_modality IS NULL OR work_modality IN ('REMOTE', 'HYBRID', 'ONSITE'));

-- ---------------------------------------------------------------------------
-- 3) Remover a coluna legada `location` (destrutivo — os dados úteis já
--    foram migrados no passo 1 e o backend novo não a referencia).
--    ⚠ Se algum valor não normalizável de `location` (ex.: cidade real) for
--    importante, copie para `notes`/nova coluna ANTES de rodar este passo.
-- ---------------------------------------------------------------------------
ALTER TABLE applications DROP COLUMN IF EXISTS location;

-- ---------------------------------------------------------------------------
-- 4) Verificação final.
-- ---------------------------------------------------------------------------
SELECT
  count(*) FILTER (WHERE work_modality IS NOT NULL) AS with_modality,
  count(*) FILTER (WHERE work_modality IS NULL)     AS without_modality,
  count(*)                                          AS total
FROM applications;

SELECT conname, pg_get_constraintdef(oid)
FROM pg_constraint
WHERE conrelid = 'applications'::regclass
  AND conname = 'chk_applications_work_modality';

COMMIT;