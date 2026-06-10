CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE usuario (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome VARCHAR(150) NOT NULL,
    cpf VARCHAR(11) NOT NULL UNIQUE,
    email VARCHAR(254) NOT NULL,
    hash_senha TEXT NOT NULL,
    telefone VARCHAR(20),
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    excluido_em TIMESTAMPTZ,
    CONSTRAINT uq_usuario_email_normalizado UNIQUE (email),
    CONSTRAINT ck_usuario_cpf_digitos CHECK (cpf ~ '^[0-9]{11}$')
);

CREATE TABLE endereco (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_usuario UUID NOT NULL REFERENCES usuario(id),
    cep VARCHAR(8) NOT NULL,
    logradouro VARCHAR(200) NOT NULL,
    numero VARCHAR(20) NOT NULL,
    complemento VARCHAR(100),
    bairro VARCHAR(100) NOT NULL,
    cidade VARCHAR(100) NOT NULL,
    estado CHAR(2) NOT NULL,
    principal BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    excluido_em TIMESTAMPTZ,
    CONSTRAINT ck_endereco_cep CHECK (cep ~ '^[0-9]{8}$'),
    CONSTRAINT ck_endereco_estado CHECK (estado ~ '^[A-Z]{2}$')
);

CREATE UNIQUE INDEX uq_endereco_principal_usuario
    ON endereco (id_usuario)
    WHERE principal = TRUE AND excluido_em IS NULL;

CREATE TABLE sessao (
    token_hash CHAR(64) PRIMARY KEY,
    id_usuario UUID NOT NULL REFERENCES usuario(id) ON DELETE CASCADE,
    expira_em TIMESTAMPTZ NOT NULL,
    criada_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_sessao_usuario ON sessao (id_usuario);
CREATE INDEX ix_sessao_expiracao ON sessao (expira_em);

CREATE TABLE dados_bancarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_usuario UUID NOT NULL UNIQUE REFERENCES usuario(id),
    provedor VARCHAR(50) NOT NULL,
    identificador_externo TEXT NOT NULL UNIQUE,
    habilitada BOOLEAN NOT NULL DEFAULT FALSE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE perfil_vendedor (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_usuario UUID NOT NULL UNIQUE REFERENCES usuario(id),
    itens_nao_enviados INTEGER NOT NULL DEFAULT 0 CHECK (itens_nao_enviados >= 0),
    bloqueado BOOLEAN NOT NULL DEFAULT FALSE,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE anuncio (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_perfil_vendedor UUID NOT NULL REFERENCES perfil_vendedor(id),
    titulo VARCHAR(120) NOT NULL,
    descricao TEXT NOT NULL,
    categoria VARCHAR(60) NOT NULL,
    tamanho VARCHAR(20) NOT NULL,
    cor VARCHAR(50) NOT NULL,
    estado_conservacao VARCHAR(20) NOT NULL,
    preco_centavos BIGINT NOT NULL CHECK (preco_centavos > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'disponivel',
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    excluido_em TIMESTAMPTZ,
    CONSTRAINT ck_anuncio_estado_conservacao CHECK (
        estado_conservacao IN ('novo', 'seminovo', 'usado', 'muito_usado', 'desgastado')
    ),
    CONSTRAINT ck_anuncio_status CHECK (
        status IN ('disponivel', 'reservado', 'vendido', 'suspenso', 'excluido')
    )
);

CREATE INDEX ix_anuncio_catalogo ON anuncio (status, criado_em DESC);
CREATE INDEX ix_anuncio_categoria ON anuncio (categoria, status);
CREATE INDEX ix_anuncio_tamanho ON anuncio (tamanho, status);
CREATE INDEX ix_anuncio_conservacao ON anuncio (estado_conservacao, status);
CREATE INDEX ix_anuncio_preco ON anuncio (preco_centavos, status);
CREATE INDEX ix_anuncio_perfil_vendedor ON anuncio (id_perfil_vendedor, status);

CREATE TABLE foto_anuncio (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_anuncio UUID NOT NULL REFERENCES anuncio(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    ordem SMALLINT NOT NULL CHECK (ordem BETWEEN 0 AND 4),
    legenda VARCHAR(200),
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_foto_anuncio_ordem UNIQUE (id_anuncio, ordem)
);

CREATE TABLE carrinho (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_usuario UUID NOT NULL UNIQUE REFERENCES usuario(id),
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE carrinho_anuncio (
    id_carrinho UUID NOT NULL REFERENCES carrinho(id) ON DELETE CASCADE,
    id_anuncio UUID NOT NULL REFERENCES anuncio(id) ON DELETE CASCADE,
    adicionado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id_carrinho, id_anuncio)
);

CREATE TABLE compra (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_comprador UUID NOT NULL REFERENCES usuario(id),
    status VARCHAR(30) NOT NULL DEFAULT 'aguardando_pagamento',
    valor_itens_centavos BIGINT NOT NULL CHECK (valor_itens_centavos >= 0),
    valor_fretes_centavos BIGINT NOT NULL CHECK (valor_fretes_centavos >= 0),
    valor_taxa_servico_centavos BIGINT NOT NULL CHECK (valor_taxa_servico_centavos >= 0),
    valor_total_centavos BIGINT NOT NULL CHECK (valor_total_centavos >= 0),
    chave_idempotencia VARCHAR(100) NOT NULL UNIQUE,
    expira_em TIMESTAMPTZ NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_compra_status CHECK (
        status IN ('aguardando_pagamento', 'aprovada', 'recusada', 'expirada', 'cancelada')
    ),
    CONSTRAINT ck_compra_total CHECK (
        valor_total_centavos =
        valor_itens_centavos + valor_fretes_centavos + valor_taxa_servico_centavos
    )
);

CREATE INDEX ix_compra_comprador ON compra (id_comprador, criado_em DESC);
CREATE INDEX ix_compra_status_expiracao ON compra (status, expira_em);

CREATE TABLE pedido (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_compra UUID NOT NULL REFERENCES compra(id),
    id_comprador UUID NOT NULL REFERENCES usuario(id),
    id_vendedor UUID NOT NULL REFERENCES usuario(id),
    status VARCHAR(30) NOT NULL DEFAULT 'criado',
    valor_itens_centavos BIGINT NOT NULL CHECK (valor_itens_centavos >= 0),
    valor_frete_centavos BIGINT NOT NULL CHECK (valor_frete_centavos >= 0),
    valor_taxa_servico_centavos BIGINT NOT NULL CHECK (valor_taxa_servico_centavos >= 0),
    valor_liquido_vendedor_centavos BIGINT NOT NULL CHECK (valor_liquido_vendedor_centavos >= 0),
    nome_destinatario VARCHAR(150) NOT NULL,
    cep_destino VARCHAR(8) NOT NULL,
    logradouro_destino VARCHAR(200) NOT NULL,
    numero_destino VARCHAR(20) NOT NULL,
    complemento_destino VARCHAR(100),
    bairro_destino VARCHAR(100) NOT NULL,
    cidade_destino VARCHAR(100) NOT NULL,
    estado_destino CHAR(2) NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finalizado_em TIMESTAMPTZ,
    CONSTRAINT uq_pedido_compra_vendedor UNIQUE (id_compra, id_vendedor),
    CONSTRAINT ck_pedido_status CHECK (
        status IN (
            'criado', 'aguardando_pagamento', 'cancelado', 'expirado',
            'aguardando_envio', 'aguardando_entrega', 'finalizado'
        )
    ),
    CONSTRAINT ck_pedido_cep_destino CHECK (cep_destino ~ '^[0-9]{8}$')
);

CREATE INDEX ix_pedido_comprador ON pedido (id_comprador, status, criado_em DESC);
CREATE INDEX ix_pedido_vendedor ON pedido (id_vendedor, status, criado_em DESC);

CREATE TABLE item_pedido (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_pedido UUID NOT NULL REFERENCES pedido(id),
    id_anuncio UUID NOT NULL REFERENCES anuncio(id),
    status VARCHAR(25) NOT NULL DEFAULT 'aguardando_envio',
    titulo VARCHAR(120) NOT NULL,
    categoria VARCHAR(60) NOT NULL,
    tamanho VARCHAR(20) NOT NULL,
    cor VARCHAR(50) NOT NULL,
    estado_conservacao VARCHAR(20) NOT NULL,
    valor_unitario_centavos BIGINT NOT NULL CHECK (valor_unitario_centavos > 0),
    taxa_servico_centavos BIGINT NOT NULL CHECK (taxa_servico_centavos >= 0),
    prazo_envio_em TIMESTAMPTZ NOT NULL,
    enviado_em TIMESTAMPTZ,
    recebido_em TIMESTAMPTZ,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_item_pedido_anuncio UNIQUE (id_pedido, id_anuncio),
    CONSTRAINT ck_item_pedido_status CHECK (
        status IN ('aguardando_envio', 'enviado', 'nao_enviado', 'recebido', 'suspenso')
    )
);

CREATE INDEX ix_item_pedido_prazo ON item_pedido (status, prazo_envio_em);

CREATE TABLE pagamento (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_compra UUID NOT NULL UNIQUE REFERENCES compra(id),
    provedor VARCHAR(50) NOT NULL,
    identificador_externo TEXT UNIQUE,
    status VARCHAR(30) NOT NULL DEFAULT 'pendente',
    valor_centavos BIGINT NOT NULL CHECK (valor_centavos > 0),
    chave_idempotencia VARCHAR(100) NOT NULL UNIQUE,
    pago_em TIMESTAMPTZ,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_pagamento_status CHECK (
        status IN ('pendente', 'aprovado', 'recusado', 'reembolsado_parcial', 'reembolsado')
    )
);

CREATE TABLE reembolso (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_pagamento UUID NOT NULL REFERENCES pagamento(id),
    id_item_pedido UUID NOT NULL REFERENCES item_pedido(id),
    identificador_externo TEXT UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pendente',
    valor_centavos BIGINT NOT NULL CHECK (valor_centavos > 0),
    motivo VARCHAR(100) NOT NULL,
    chave_idempotencia VARCHAR(100) NOT NULL UNIQUE,
    processado_em TIMESTAMPTZ,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_reembolso_status CHECK (
        status IN ('pendente', 'processado', 'falhou')
    )
);

CREATE TABLE entrega (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_pedido UUID NOT NULL UNIQUE REFERENCES pedido(id),
    provedor VARCHAR(50),
    codigo_rastreio VARCHAR(100),
    status VARCHAR(25) NOT NULL DEFAULT 'aguardando_postagem',
    valor_frete_centavos BIGINT NOT NULL CHECK (valor_frete_centavos >= 0),
    postado_em TIMESTAMPTZ,
    entregue_em TIMESTAMPTZ,
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_entrega_status CHECK (
        status IN ('aguardando_postagem', 'postado', 'em_transito', 'entregue', 'falhou')
    )
);

CREATE TABLE conversa (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_pedido UUID NOT NULL UNIQUE REFERENCES pedido(id),
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mensagem (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_conversa UUID NOT NULL REFERENCES conversa(id) ON DELETE CASCADE,
    id_usuario_remetente UUID NOT NULL REFERENCES usuario(id),
    conteudo TEXT NOT NULL,
    lida_em TIMESTAMPTZ,
    criada_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_mensagem_conteudo CHECK (length(trim(conteudo)) BETWEEN 1 AND 4000)
);

CREATE INDEX ix_mensagem_conversa ON mensagem (id_conversa, criada_em DESC);

CREATE TABLE notificacao (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_usuario UUID NOT NULL REFERENCES usuario(id),
    tipo VARCHAR(50) NOT NULL,
    conteudo TEXT NOT NULL,
    lida_em TIMESTAMPTZ,
    criada_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_notificacao_usuario ON notificacao (id_usuario, criada_em DESC);

CREATE TABLE avaliacao (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_pedido UUID NOT NULL UNIQUE REFERENCES pedido(id),
    id_usuario_autor UUID NOT NULL REFERENCES usuario(id),
    id_usuario_avaliado UUID NOT NULL REFERENCES usuario(id),
    nota SMALLINT NOT NULL CHECK (nota BETWEEN 1 AND 5),
    comentario TEXT,
    criada_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_avaliacao_usuarios_distintos CHECK (id_usuario_autor <> id_usuario_avaliado)
);

CREATE INDEX ix_avaliacao_usuario_avaliado ON avaliacao (id_usuario_avaliado, criada_em DESC);

CREATE TABLE evento_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agregado_tipo VARCHAR(50) NOT NULL,
    agregado_id UUID NOT NULL,
    evento_tipo VARCHAR(100) NOT NULL,
    dados JSONB NOT NULL,
    tentativas INTEGER NOT NULL DEFAULT 0 CHECK (tentativas >= 0),
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processado_em TIMESTAMPTZ,
    proxima_tentativa_em TIMESTAMPTZ
);

CREATE INDEX ix_evento_outbox_pendente
    ON evento_outbox (criado_em)
    WHERE processado_em IS NULL;
