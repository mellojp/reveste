const maximumImageBytes = 5 * 1024 * 1024;
const maximumImagePixels = 16_000_000;
const maximumImageDimension = 6_000;

export async function uploadPhoto(file) {
  const safeFile = await prepareImageForUpload(file);
  const responseAuthorization = await fetch("/v1/uploads/imagens/autorizacoes", {
    method: "POST",
    credentials: "same-origin",
    redirect: "error",
    referrerPolicy: "no-referrer",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      nome_arquivo: safeFile.name,
      tipo: safeFile.type,
      tamanho: safeFile.size,
    }),
  });
  const authorization = await responseAuthorization.json().catch(() => null);
  if (!responseAuthorization.ok) {
    throw new Error(authorization?.mensagem || "Não foi possível autorizar o envio da imagem.");
  }
  const tokenParts = authorization.token?.split("_") || [];
  const storeID = tokenParts[3];
  const uploadURL = new URL(authorization.url_upload);
  if (uploadURL.origin !== "https://vercel.com" ||
      uploadURL.pathname !== "/api/blob/" ||
      !storeID ||
      !authorization.pathname?.startsWith("anuncios/")) {
    throw new Error("A autorização de upload recebida é inválida.");
  }
  uploadURL.searchParams.set("pathname", authorization.pathname);
  let response;
  try {
    response = await fetch(uploadURL, {
      method: "PUT",
      body: safeFile,
      credentials: "omit",
      mode: "cors",
      redirect: "error",
      referrerPolicy: "no-referrer",
      headers: {
        Authorization: `Bearer ${authorization.token}`,
        "x-api-version": "12",
        "x-api-blob-request-id": `${storeID}:${Date.now()}:${crypto.randomUUID()}`,
        "x-api-blob-request-attempt": "0",
        "x-vercel-blob-store-id": storeID,
        "x-vercel-blob-access": "public",
        "x-content-type": safeFile.type,
      },
    });
  } catch {
    throw new Error("O upload foi recusado. Confirme a configuração do Blob store público.");
  }
  const blob = await response.json().catch(() => null);
  if (!response.ok || !isPublicImageURL(blob?.url)) {
    throw new Error(blob?.error?.message || `Não foi possível enviar ${safeFile.name}.`);
  }
  return blob.url;
}

export function isPublicImageURL(value) {
  try {
    const url = new URL(value);
    return url.protocol === "https:" &&
      url.username === "" &&
      url.password === "" &&
      url.search === "" &&
      url.hash === "" &&
      url.hostname.endsWith(".public.blob.vercel-storage.com");
  } catch {
    return false;
  }
}

export async function validateImageSignature(file) {
  const bytes = new Uint8Array(await file.slice(0, 12).arrayBuffer());
  const jpeg = bytes.length >= 3 &&
    bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff;
  const png = bytes.length >= 8 &&
    bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4e &&
    bytes[3] === 0x47 && bytes[4] === 0x0d && bytes[5] === 0x0a &&
    bytes[6] === 0x1a && bytes[7] === 0x0a;
  const webp = bytes.length >= 12 &&
    ascii(bytes.slice(0, 4)) === "RIFF" && ascii(bytes.slice(8, 12)) === "WEBP";
  return jpeg || png || webp;
}

async function prepareImageForUpload(file) {
  if (!["image/jpeg", "image/png", "image/webp"].includes(file.type) ||
      file.size <= 0 || file.size > maximumImageBytes ||
      !await validateImageSignature(file)) {
    throw new Error("A imagem selecionada não possui um formato válido.");
  }
  let bitmap;
  try {
    bitmap = await createImageBitmap(file, { imageOrientation: "from-image" });
  } catch {
    try {
      bitmap = await createImageBitmap(file);
    } catch {
      throw new Error("A imagem selecionada está corrompida ou não pôde ser lida.");
    }
  }
  try {
    if (bitmap.width <= 0 || bitmap.height <= 0 ||
        bitmap.width > maximumImageDimension ||
        bitmap.height > maximumImageDimension ||
        bitmap.width * bitmap.height > maximumImagePixels) {
      throw new Error("A imagem possui dimensões muito grandes.");
    }
    const canvas = document.createElement("canvas");
    canvas.width = bitmap.width;
    canvas.height = bitmap.height;
    const context = canvas.getContext("2d", { alpha: true });
    if (!context) {
      throw new Error("Não foi possível preparar a imagem para envio.");
    }
    context.drawImage(bitmap, 0, 0);
    let blob = await canvasBlob(canvas, 0.9);
    if (blob.size > maximumImageBytes) {
      blob = await canvasBlob(canvas, 0.72);
    }
    if (blob.size <= 0 || blob.size > maximumImageBytes) {
      throw new Error("A imagem continua muito grande após o processamento.");
    }
    const baseName = file.name.replace(/\.[^.]*$/, "").replace(/[^a-zA-Z0-9_-]/g, "_") ||
      "imagem";
    return new File([blob], `${baseName}.webp`, { type: "image/webp" });
  } finally {
    bitmap.close();
  }
}

function canvasBlob(canvas, quality) {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => blob ? resolve(blob) : reject(new Error("Não foi possível processar a imagem.")),
      "image/webp",
      quality,
    );
  });
}

function ascii(bytes) {
  return String.fromCharCode(...bytes);
}
