import { request } from "../core/api.js";
import { clearFormErrors, showFormErrors } from "../core/forms.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { state } from "../core/state.js";
import { uploadPhoto } from "../core/uploads.js";
import { categories, conditions, escapeHTML } from "../core/utils.js";
import { emptyState } from "../components/products.js";
import { pageSkeleton, setButtonLoading } from "../core/feedback.js";

const allowedTypes = new Set(["image/jpeg", "image/png", "image/webp"]);

export async function editAdPage(root, { idAnuncio }) {
  root.innerHTML = pageSkeleton("Carregando anúncio");
  try {
    const ad = await request(`/v1/anuncios/${encodeURIComponent(idAnuncio)}`);
    renderEditor(root, ad);
  } catch (error) {
    root.innerHTML = `<section class="page-section compact-section">${emptyState("Não foi possível editar este anúncio", error.message, '<a class="button button-dark" href="/meus-anuncios" data-link>Voltar ao painel</a>')}</section>`;
  }
}

function renderEditor(root, ad) {
  if (ad.id_vendedor !== state.user?.id) {
    root.innerHTML = `<section class="page-section compact-section">${emptyState("Você não pode editar este anúncio", "Somente o proprietário pode alterar esta peça.", '<a class="button button-dark" href="/meus-anuncios" data-link>Voltar ao painel</a>')}</section>`;
    return;
  }
  if (ad.status !== "disponivel") {
    root.innerHTML = `<section class="page-section compact-section">${emptyState("Este anúncio não pode ser editado", "Somente anúncios disponíveis podem ser alterados.", '<a class="button button-dark" href="/meus-anuncios" data-link>Voltar ao painel</a>')}</section>`;
    return;
  }
  let photos = [...(ad.fotos || [])]
    .sort((first, second) => first.ordem - second.ordem)
    .map((photo) => ({ url: photo.url, preview: photo.url }));
  root.innerHTML = `
    <section class="form-page page-section compact-section">
      <nav class="breadcrumbs"><a href="/meus-anuncios" data-link>Meus anúncios</a><span>/</span><span>Editar</span></nav>
      <div class="page-intro"><span class="eyebrow">Editar anúncio</span><h1>Atualize os detalhes da peça.</h1><p>As alterações ficam visíveis no catálogo assim que forem salvas.</p></div>
      <form class="editor-form" id="edit-ad-form">
        <section class="form-panel"><div class="panel-number">01</div><div class="panel-content">
          <h2>Fotos</h2><p>Mantenha entre 2 e 5 imagens. Arraste a ordem pelos controles.</p>
          <label class="photo-drop compact-drop"><input id="edit-photo-input" type="file" accept="image/jpeg,image/png,image/webp" multiple><span class="upload-icon">+</span><strong>Adicionar imagens</strong></label>
          <div class="photo-preview" id="edit-photo-preview"></div>
          <p class="upload-status hidden" id="edit-upload-status"></p>
        </div></section>
        <section class="form-panel"><div class="panel-number">02</div><div class="panel-content">
          <h2>Informações</h2>
          <div class="form-grid">
            <label class="span-2">Título<input name="titulo" maxlength="120" value="${escapeHTML(ad.titulo)}" required></label>
            <label class="span-2">Descrição<textarea name="descricao" rows="5" required>${escapeHTML(ad.descricao)}</textarea></label>
            <label>Categoria<select name="categoria" required>${categories.slice(1).map(([value, label]) => `<option value="${value}" ${ad.categoria === value ? "selected" : ""}>${label}</option>`).join("")}</select></label>
            <label>Tamanho<input name="tamanho" maxlength="20" value="${escapeHTML(ad.tamanho)}" required></label>
            <label>Cor<input name="cor" maxlength="60" value="${escapeHTML(ad.cor)}" required></label>
            <label>Conservação<select name="estado_conservacao" required>${conditions.slice(1).map(([value, label]) => `<option value="${value}" ${ad.estado_conservacao === value ? "selected" : ""}>${label}</option>`).join("")}</select></label>
          </div>
        </div></section>
        <section class="form-panel"><div class="panel-number">03</div><div class="panel-content">
          <h2>Preço</h2><label class="price-field"><span>R$</span><input name="preco" inputmode="decimal" value="${(ad.preco_centavos / 100).toFixed(2).replace(".", ",")}" required></label>
        </div></section>
        <div class="form-submit"><a class="text-link" href="/meus-anuncios" data-link>Cancelar</a><button class="button button-dark button-large" type="submit">Salvar alterações</button></div>
      </form>
    </section>`;

  const preview = root.querySelector("#edit-photo-preview");
  const renderPhotos = () => {
    preview.innerHTML = photos.map((photo, index) => `
      <div class="photo-preview-item">
        <img src="${escapeHTML(photo.preview)}" alt="Foto ${index + 1}">
        <span>${index === 0 ? "Capa" : index + 1}</span>
        <div class="photo-preview-controls">
          <button type="button" data-photo-left="${index}" ${index === 0 ? "disabled" : ""}>←</button>
          <button type="button" data-photo-right="${index}" ${index === photos.length - 1 ? "disabled" : ""}>→</button>
          <button type="button" data-photo-remove="${index}">×</button>
        </div>
      </div>`).join("");
  };
  renderPhotos();
  preview.addEventListener("click", (event) => {
    const remove = event.target.closest("[data-photo-remove]");
    const left = event.target.closest("[data-photo-left]");
    const right = event.target.closest("[data-photo-right]");
    if (remove) photos.splice(Number(remove.dataset.photoRemove), 1);
    if (left) {
      const index = Number(left.dataset.photoLeft);
      [photos[index - 1], photos[index]] = [photos[index], photos[index - 1]];
    }
    if (right) {
      const index = Number(right.dataset.photoRight);
      [photos[index + 1], photos[index]] = [photos[index], photos[index + 1]];
    }
    if (remove || left || right) renderPhotos();
  });
  root.querySelector("#edit-photo-input").addEventListener("change", (event) => {
    const files = [...event.target.files];
    for (const file of files) {
      if (photos.length >= 5) break;
      photos.push({ file, preview: URL.createObjectURL(file) });
    }
    renderPhotos();
    event.target.value = "";
  });
  root.querySelector("#edit-ad-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const values = Object.fromEntries(new FormData(form));
    const button = form.querySelector("button[type=submit]");
    const status = root.querySelector("#edit-upload-status");
    clearFormErrors(form);
    const error = validatePhotos(photos);
    if (error) {
      toast(error);
      return;
    }
    const price = priceToCents(values.preco);
    if (!price) {
      showFormErrors(form, { preco: "Informe um preço válido." });
      return;
    }
    setButtonLoading(button, true, "Salvando...");
    status.classList.remove("hidden");
    try {
      const urls = [];
      for (const [index, photo] of photos.entries()) {
        status.textContent = `Preparando foto ${index + 1} de ${photos.length}...`;
        urls.push(photo.url || await uploadPhoto(photo.file));
      }
      await request(`/v1/me/anuncios/${encodeURIComponent(ad.id)}`, {
        method: "PATCH",
        body: JSON.stringify({
          titulo: values.titulo, descricao: values.descricao,
          categoria: values.categoria, tamanho: values.tamanho, cor: values.cor,
          estado_conservacao: values.estado_conservacao,
          preco_centavos: price, urls_fotos: urls,
        }),
      });
      toast("Anúncio atualizado.");
      navigate(`/anuncios/${encodeURIComponent(ad.id)}`);
    } catch (requestError) {
      showFormErrors(form, requestError.fields);
      toast(requestError.message);
      status.classList.add("hidden");
    } finally {
      setButtonLoading(button, false);
    }
  });
}

function validatePhotos(photos) {
  if (photos.length < 2 || photos.length > 5) return "Mantenha entre 2 e 5 imagens.";
  const invalid = photos.some(({ file }) => file && (
    !allowedTypes.has(file.type) || file.size <= 0 || file.size > 5 * 1024 * 1024
  ));
  return invalid ? "Use imagens JPEG, PNG ou WebP com até 5 MB." : "";
}

function priceToCents(value) {
  const normalized = String(value).replace(/\./g, "").replace(",", ".");
  const number = Number(normalized);
  return Number.isFinite(number) && number > 0 ? Math.round(number * 100) : 0;
}

editAdPage.title = "Editar anúncio";
