ALTER TABLE notificacao
    ADD COLUMN id_pedido UUID REFERENCES pedido(id);

CREATE INDEX ix_notificacao_pedido ON notificacao (id_pedido);
