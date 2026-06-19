DROP INDEX IF EXISTS ix_notificacao_pedido;

ALTER TABLE notificacao
    DROP COLUMN IF EXISTS id_pedido;
