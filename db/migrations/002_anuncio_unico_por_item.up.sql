ALTER TABLE item_pedido
    DROP CONSTRAINT uq_item_pedido_anuncio;

ALTER TABLE item_pedido
    ADD CONSTRAINT uq_item_pedido_anuncio UNIQUE (id_anuncio);
