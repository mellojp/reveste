# Estrutura da aplicacao

O fluxo principal da API e:

```text
http -> casosdeuso -> dominio -> database
```

Os casos de uso estao separados por funcionalidade:

- `casosdeuso/cadastros`: cadastro, autenticacao e sessoes;
- `casosdeuso/anuncios`: publicacao e consulta de anuncios;
- `casosdeuso/compras`: carrinho e futuros fluxos de checkout e pedidos.
