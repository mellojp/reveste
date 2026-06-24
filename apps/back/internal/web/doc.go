// Package web implementa o adaptador HTTP orientado a documentos HTML.
//
// Fluxo de leitura:
//
//	requisicao GET -> consultas_paginas -> controlador de caso de uso
//	    -> contextoDocumento -> respostas_html -> template
//
// Fluxo de escrita:
//
//	formulario POST -> comandos_formularios -> controlador de caso de uso
//	    -> documento com validacao ou redirecionamento
//
// Sessao, limite de autenticacao e apresentacao ficam isolados para que o
// adaptador nao replique regras pertencentes aos casos de uso ou ao dominio.
package web
