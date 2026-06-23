package cep

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"reveste/apps/api/internal/common"
)

func novoViaCEPDeTeste(servidor *httptest.Server) *ViaCEP {
	v := NovoViaCEP()
	v.cliente = servidor.Client()
	v.urlBase = servidor.URL + "/"
	return v
}

func TestViaCEPSucesso(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"cep":"01310-100","logradouro":"Avenida Paulista","complemento":"de 612 a 1510","bairro":"Bela Vista","localidade":"São Paulo","uf":"SP"}`))
	}))
	defer servidor.Close()

	endereco, err := novoViaCEPDeTeste(servidor).ConsultarCEP(context.Background(), "01310100")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if endereco.Logradouro != "Avenida Paulista" || endereco.Bairro != "Bela Vista" {
		t.Fatalf("logradouro/bairro inesperados: %+v", endereco)
	}
	if endereco.Cidade != "São Paulo" || endereco.Estado != "SP" {
		t.Fatalf("cidade/estado inesperados: %+v", endereco)
	}
	if endereco.CEP != "01310100" {
		t.Fatalf("cep deveria conter apenas digitos, veio %q", endereco.CEP)
	}
}

func TestViaCEPNaoEncontradoBooleano(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"erro": true}`))
	}))
	defer servidor.Close()

	_, err := novoViaCEPDeTeste(servidor).ConsultarCEP(context.Background(), "00000000")
	if !errors.Is(err, common.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, veio %v", err)
	}
}

func TestViaCEPNaoEncontradoStringErro(t *testing.T) {
	// Algumas versoes do ViaCEP devolvem "erro" como string.
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"erro": "true"}`))
	}))
	defer servidor.Close()

	_, err := novoViaCEPDeTeste(servidor).ConsultarCEP(context.Background(), "00000000")
	if !errors.Is(err, common.ErrNaoEncontrado) {
		t.Fatalf("esperava ErrNaoEncontrado, veio %v", err)
	}
}

func TestViaCEPProvedorForaDoAr(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer servidor.Close()

	_, err := novoViaCEPDeTeste(servidor).ConsultarCEP(context.Background(), "01310100")
	if !errors.Is(err, common.ErrConsultaCEPIndisponivel) {
		t.Fatalf("esperava ErrConsultaCEPIndisponivel, veio %v", err)
	}
}
