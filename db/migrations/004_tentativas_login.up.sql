CREATE TABLE tentativa_login (
    id BIGSERIAL PRIMARY KEY,
    chave VARCHAR(64) NOT NULL,
    criada_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_tentativa_login_chave ON tentativa_login (chave, criada_em);
