import { request } from "./api.js";

export async function uploadPhoto(file) {
  const authorization = await request("/v1/uploads/imagens/autorizacoes", {
    method: "POST",
    body: JSON.stringify({ nome_arquivo: file.name, tipo: file.type, tamanho: file.size }),
  });
  const storeID = authorization.token.split("_")[3];
  const uploadURL = new URL(authorization.url_upload);
  uploadURL.searchParams.set("pathname", authorization.pathname);
  let response;
  try {
    response = await fetch(uploadURL, {
      method: "PUT",
      body: file,
      headers: {
        Authorization: `Bearer ${authorization.token}`,
        "x-api-version": "12",
        "x-api-blob-request-id": `${storeID}:${Date.now()}:${crypto.randomUUID()}`,
        "x-api-blob-request-attempt": "0",
        "x-vercel-blob-store-id": storeID,
        "x-vercel-blob-access": "public",
        "x-content-type": file.type,
      },
    });
  } catch {
    throw new Error("O upload foi recusado. Confirme a configuração do Blob store público.");
  }
  const blob = await response.json().catch(() => null);
  if (!response.ok || !blob?.url) {
    throw new Error(blob?.error?.message || `Não foi possível enviar ${file.name}.`);
  }
  return blob.url;
}
