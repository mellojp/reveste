UPDATE anuncio
SET categoria = CASE
    WHEN LOWER(categoria) IN ('vestido', 'vestidos') THEN 'vestidos'
    WHEN LOWER(categoria) IN ('camisa', 'camisas', 'camiseta', 'camisetas', 'blusa', 'blusas') THEN 'camisetas'
    WHEN LOWER(categoria) IN ('calca', 'calcas', 'calça', 'calças') THEN 'calcas'
    WHEN LOWER(categoria) IN ('saia', 'saias', 'short', 'shorts', 'bermuda', 'bermudas') THEN 'saias_e_shorts'
    WHEN LOWER(categoria) IN ('casaco', 'casacos', 'jaqueta', 'jaquetas') THEN 'casacos'
    WHEN LOWER(categoria) IN ('acessorio', 'acessorios', 'acessório', 'acessórios') THEN 'acessorios'
    WHEN LOWER(categoria) IN ('calcado', 'calcados', 'calçado', 'calçados', 'sapato', 'sapatos') THEN 'calcados'
    ELSE 'outros'
END;

ALTER TABLE anuncio
ADD CONSTRAINT ck_anuncio_categoria CHECK (
    categoria IN (
        'vestidos', 'camisetas', 'calcas', 'saias_e_shorts',
        'casacos', 'acessorios', 'calcados', 'outros'
    )
);
