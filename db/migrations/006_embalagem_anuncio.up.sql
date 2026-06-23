-- Peso e dimensoes da peca embalada, usados na cotacao de frete.
-- Os defaults fazem o backfill das linhas existentes com valores plausiveis e validos;
-- novos anuncios sempre informam os quatro campos pelo formulario.
ALTER TABLE anuncio
    ADD COLUMN peso_g INTEGER NOT NULL DEFAULT 300,
    ADD COLUMN altura_cm INTEGER NOT NULL DEFAULT 5,
    ADD COLUMN largura_cm INTEGER NOT NULL DEFAULT 20,
    ADD COLUMN comprimento_cm INTEGER NOT NULL DEFAULT 30;

-- Limites alinhados aos minimos/maximos de encomendas dos Correios (PAC/SEDEX),
-- evitando que a cotacao do agregador seja recusada por embalagem fora do padrao.
ALTER TABLE anuncio
    ADD CONSTRAINT ck_anuncio_embalagem CHECK (
        peso_g BETWEEN 1 AND 30000
        AND altura_cm BETWEEN 2 AND 105
        AND largura_cm BETWEEN 11 AND 105
        AND comprimento_cm BETWEEN 16 AND 105
    );
