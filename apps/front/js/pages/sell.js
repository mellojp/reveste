import { request } from "../core/api.js";
import { clearFormErrors, showFormErrors } from "../core/forms.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { categories, conditions } from "../core/utils.js";

const allowedTypes = new Set(["image/jpeg", "image/png", "image/webp"]);

export async function sellPage(root) {
  root.innerHTML = `
    <section class="form-page page-section compact-section">
      <div class="page-intro">
        <span class="eyebrow">Novo anúncio</span>
        <h1>Conte a história da sua peça.</h1>
        <p>Boas fotos e informações claras ajudam a peça a encontrar a próxima pessoa.</p>
      </div>
      <form class="editor-form" id="sell-form">
        <section class="form-panel">
          <div class="panel-number">01</div>
          <div class="panel-content">
            <h2>Fotos</h2><p>Envie de 2 a 5 imagens. A primeira será a capa do anúncio.</p>
            <label class="photo-drop">
              <input id="photo-input" name="fotos" type="file" accept="image/jpeg,image/png,image/webp" multiple required>
              <span class="upload-icon">+</span><strong>Escolher imagens</strong><small>JPEG, PNG ou WebP · até 5 MB cada</small>
            </label>
            <div class="photo-preview" id="photo-preview"></div>
            <p class="upload-status hidden" id="upload-status"></p>
          </div>
        </section>
        <section class="form-panel">
          <div class="panel-number">02</div>
          <div class="panel-content">
            <h2>Sobre a peça</h2>
            <div class="form-grid">
              <label class="span-2">Título<input name="titulo" maxlength="120" placeholder="Ex.: Blazer de linho bege" required></label>
              <label class="span-2">Descrição<textarea name="descricao" rows="5" placeholder="Conte sobre caimento, detalhes e marcas de uso." required></textarea></label>
              <label>Categoria<select name="categoria" required>${categories.slice(1).map(([value, label]) => `<option value="${value}">${label}</option>`).join("")}</select></label>
              <label>Tamanho<input name="tamanho" maxlength="20" placeholder="Ex.: M, 38, único" required></label>
              <label>Cor<input name="cor" maxlength="60" required></label>
              <label>Conservação<select name="estado_conservacao" required>${conditions.slice(1).map(([value, label]) => `<option value="${value}">${label}</option>`).join("")}</select></label>
            </div>
          </div>
        </section>
        <section class="form-panel">
          <div class="panel-number">03</div>
          <div class="panel-content">
            <h2>Preço</h2><p>Informe o valor da peça em reais.</p>
            <label class="price-field"><span>R$</span><input name="preco" inputmode="decimal" placeholder="0,00" required></label>
          </div>
        </section>
        <div class="form-submit">
          <a class="text-link" href="/meus-anuncios" data-link>Cancelar</a>
          <button class="button button-dark button-large" type="submit">Publicar anúncio</button>
        </div>
      </form>
    </section>
  `;

  const form = root.querySelector("#sell-form");
  const input = root.querySelector("#photo-input");
  input.addEventListener("change", () => renderPhotoPreview(input.files, root.querySelector("#photo-preview")));
  form.addEventListener("submit", (event) => submitAd(event, root));
}

function renderPhotoPreview(fileList, preview) {
  preview.innerHTML = "";
  [...fileList].forEach((file, index) => {
    const item = document.createElement("div");
    item.className = "photo-preview-item";
    const image = document.createElement("img");
    image.alt = `Prévia da foto ${index + 1}`;
    image.src = URL.createObjectURL(file);
    image.addEventListener("load", () => URL.revokeObjectURL(image.src), { once: true });
    item.innerHTML = `<span>${index + 1}</span>`;
    item.prepend(image);
    preview.append(item);
  });
}

function validatePhotos(files) {
  if (files.length < 2 || files.length > 5) return "Selecione entre 2 e 5 imagens.";
  if (files.some((file) => !allowedTypes.has(file.type))) return "Envie apenas imagens JPEG, PNG ou WebP.";
  if (files.some((file) => file.size <= 0 || file.size > 5 * 1024 * 1024)) return "Cada imagem deve ter no máximo 5 MB.";
  return "";
}

async function submitAd(event, root) {
  event.preventDefault();
  const form = event.currentTarget;
  const values = Object.fromEntries(new FormData(form));
  const files = [...root.querySelector("#photo-input").files];
  const status = root.querySelector("#upload-status");
  const button = form.querySelector("button[type=submit]");
  clearFormErrors(form);
  const photoError = validatePhotos(files);
  if (photoError) {
    showFormErrors(form, { fotos: photoError });
    toast(photoError);
    return;
  }
  const normalizedPrice = values.preco.replace(/\./g, "").replace(",", ".");
  const price = Math.round(Number(normalizedPrice) * 100);
  if (!Number.isFinite(price) || price <= 0) {
    showFormErrors(form, { preco: "Informe um preço válido." });
    return;
  }
  button.disabled = true;
  status.classList.remove("hidden");
  try {
    const urls = [];
    for (const [index, file] of files.entries()) {
      status.textContent = `Enviando foto ${index + 1} de ${files.length}...`;
      urls.push(await uploadPhoto(file));
    }
    status.textContent = "Fotos enviadas. Finalizando anúncio...";
    await request("/v1/anuncios", {
      method: "POST",
      body: JSON.stringify({
        titulo: values.titulo,
        descricao: values.descricao,
        categoria: values.categoria,
        tamanho: values.tamanho,
        cor: values.cor,
        estado_conservacao: values.estado_conservacao,
        preco_centavos: price,
        urls_fotos: urls,
      }),
    });
    toast("Anúncio publicado.");
    navigate("/meus-anuncios");
  } catch (error) {
    status.classList.add("hidden");
    showFormErrors(form, error.fields);
    toast(error.message);
  } finally {
    button.disabled = false;
  }
}

async function uploadPhoto(file) {
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

sellPage.title = "Vender";
