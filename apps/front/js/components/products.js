import { adImage, escapeHTML, money } from "../core/utils.js";
import { state } from "../core/state.js";

export function productCard(ad, { allowCart = true } = {}) {
  const detailURL = `/anuncios/${encodeURIComponent(ad.id)}`;
  const inCart = state.cart.anuncios?.some((item) => item.id === ad.id);
  return `
    <article class="product-card">
      <div class="product-image">
        <a class="product-image-link" href="${detailURL}" data-link aria-label="Ver detalhes de ${escapeHTML(ad.titulo)}">
          ${adImage(ad)}
          <span class="condition-tag">${escapeHTML(ad.estado_conservacao.replaceAll("_", " "))}</span>
        </a>
        ${allowCart ? `
          <button class="add-cart ${inCart ? "is-added" : ""}" data-add-cart="${escapeHTML(ad.id)}" aria-label="${inCart ? "Peça já está na sacola" : `Adicionar ${escapeHTML(ad.titulo)} à sacola`}" ${inCart ? "disabled" : ""}>
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 8h12l1 13H5L6 8Zm3 2V6a3 3 0 0 1 6 0v4"/></svg>
          </button>` : ""}
      </div>
      <div class="product-info">
        <a href="${detailURL}" data-link>
          <span class="product-meta">${escapeHTML(ad.categoria.replaceAll("_", " "))} · Tam. ${escapeHTML(ad.tamanho)}</span>
          <h3>${escapeHTML(ad.titulo)}</h3>
          <strong>${money(ad.preco_centavos)}</strong>
        </a>
      </div>
    </article>
  `;
}

export function emptyState(title, description, action = "") {
  return `<div class="empty-state"><span class="empty-state-mark" aria-hidden="true">R</span><h3>${title}</h3><p>${description}</p>${action}</div>`;
}
