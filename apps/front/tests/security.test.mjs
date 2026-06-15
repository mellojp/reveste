import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { isPublicImageURL, validateImageSignature } from "../js/uploads.js";

test("upload rejeita conteudo disfarcado de imagem", async () => {
  const html = new Blob(["<script>alert(1)</script>"], { type: "image/jpeg" });
  const jpeg = new Blob([new Uint8Array([0xff, 0xd8, 0xff, 0xdb])], {
    type: "image/jpeg",
  });

  assert.equal(await validateImageSignature(html), false);
  assert.equal(await validateImageSignature(jpeg), true);
});

test("imagens publicas aceitam apenas o Blob publico esperado", () => {
  assert.equal(
    isPublicImageURL("https://reveste.public.blob.vercel-storage.com/anuncios/foto.jpg"),
    true,
  );
  assert.equal(isPublicImageURL("https://atacante.example/rastreio.png"), false);
  assert.equal(isPublicImageURL("javascript:alert(1)"), false);
  assert.equal(
    isPublicImageURL("https://reveste.public.blob.vercel-storage.com/foto.jpg?token=segredo"),
    false,
  );
});

test("runtime SSR nao monta HTML dinamico com innerHTML", async () => {
  const source = await readFile(new URL("../js/web.js", import.meta.url), "utf8");
  assert.doesNotMatch(source, /\.innerHTML\s*=/);
  assert.doesNotMatch(source, /insertAdjacentHTML/);
  assert.doesNotMatch(source, /\beval\s*\(/);
});
