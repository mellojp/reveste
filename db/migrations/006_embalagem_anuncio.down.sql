ALTER TABLE anuncio DROP CONSTRAINT IF EXISTS ck_anuncio_embalagem;
ALTER TABLE anuncio
    DROP COLUMN IF EXISTS peso_g,
    DROP COLUMN IF EXISTS altura_cm,
    DROP COLUMN IF EXISTS largura_cm,
    DROP COLUMN IF EXISTS comprimento_cm;
