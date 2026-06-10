package common

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

type GeradorIDCriptografico struct{}

func (GeradorIDCriptografico) Novo() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic("nao foi possivel gerar identificador seguro: " + err.Error())
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	texto := hex.EncodeToString(bytes)
	return texto[0:8] + "-" + texto[8:12] + "-" + texto[12:16] + "-" + texto[16:20] + "-" + texto[20:32]
}

type RelogioSistema struct{}

func (RelogioSistema) Agora() time.Time {
	return time.Now().UTC()
}

type ProcessadorPBKDF2 struct {
	Iteracoes int
}

func (p ProcessadorPBKDF2) Gerar(senha string) (string, error) {
	if len(senha) < 8 {
		return "", errors.New("senha deve possuir ao menos 8 caracteres")
	}
	iteracoes := p.Iteracoes
	if iteracoes <= 0 {
		iteracoes = 210_000
	}
	sal := make([]byte, 16)
	if _, err := rand.Read(sal); err != nil {
		return "", err
	}
	hash := pbkdf2SHA256([]byte(senha), sal, iteracoes, 32)
	return "pbkdf2_sha256$" + strconv.Itoa(iteracoes) + "$" +
		base64.RawStdEncoding.EncodeToString(sal) + "$" +
		base64.RawStdEncoding.EncodeToString(hash), nil
}

func (p ProcessadorPBKDF2) Comparar(hashArmazenado, senha string) bool {
	partes := strings.Split(hashArmazenado, "$")
	if len(partes) != 4 || partes[0] != "pbkdf2_sha256" {
		return false
	}
	iteracoes, err := strconv.Atoi(partes[1])
	if err != nil || iteracoes < 100_000 {
		return false
	}
	sal, err := base64.RawStdEncoding.DecodeString(partes[2])
	if err != nil {
		return false
	}
	esperado, err := base64.RawStdEncoding.DecodeString(partes[3])
	if err != nil {
		return false
	}
	obtido := pbkdf2SHA256([]byte(senha), sal, iteracoes, len(esperado))
	return subtle.ConstantTimeCompare(esperado, obtido) == 1
}

func pbkdf2SHA256(senha, sal []byte, iteracoes, tamanho int) []byte {
	tamanhoHash := sha256.Size
	blocos := (tamanho + tamanhoHash - 1) / tamanhoHash
	resultado := make([]byte, 0, blocos*tamanhoHash)

	for bloco := 1; bloco <= blocos; bloco++ {
		mac := hmac.New(sha256.New, senha)
		mac.Write(sal)
		mac.Write([]byte{byte(bloco >> 24), byte(bloco >> 16), byte(bloco >> 8), byte(bloco)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iteracoes; i++ {
			mac = hmac.New(sha256.New, senha)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		resultado = append(resultado, t...)
	}
	return resultado[:tamanho]
}
