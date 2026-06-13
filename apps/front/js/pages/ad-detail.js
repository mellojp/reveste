import { request } from "../core/api.js";
import { addToCart } from "../core/cart.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { state } from "../core/state.js";
import { escapeHTML, isPublicImageURL, money, placeholderSVG } from "../core/utils.js";
import { emptyState } from "../components/products.js";

export async function adDetailPage(root, { idAnuncio }) {
  root.innerHTML = `<section class="ad-detail-page page-section"><div class="page-loading">Carregando anúncio...</div></section>`;
  const cachedAd = findCachedAd(idAnuncio);
  try {
    const ad = await request(`/v1/anuncios/${encodeURIComponent(idAnuncio)}`);
    document.title = `ReVeste | ${ad.titulo}`;
    renderDetail(root, ad);
  } catch (error) {
    if (cachedAd) {
      document.title = `ReVeste | ${cachedAd.titulo}`;
      renderDetail(root, cachedAd);
      toast("Exibindo os dados já carregados. Não foi possível atualizar o anúncio.");
      return;
    }
    const notFound = error.status === 404;
    root.innerHTML = `
      <section class="ad-detail-page page-section">
        ${emptyState(
          notFound ? "Anúncio não encontrado" : "Não foi possível carregar o anúncio",
          notFound ? "A peça pode ter sido removida ou o endereço está incorreto." : error.message,
          '<a class="button button-dark" href="/catalogo" data-link>Voltar ao catálogo</a>',
        )}
      </section>`;
  }
}

function findCachedAd(idAnuncio) {
  return [
    ...(state.catalog || []),
    ...(state.cart.anuncios || []),
  ].find((ad) => ad.id === idAnuncio);
}

function renderDetail(root, ad) {
  const photos = [...(ad.fotos || [])].sort((first, second) => first.ordem - second.ordem);
  const isOwnAd = state.user?.id === ad.id_vendedor;
  const isAvailable = ad.status === "disponivel";
  root.innerHTML = `
    <section class="ad-detail-page page-section">
      <nav class="breadcrumbs" aria-label="Navegação estrutural">
        <a href="/catalogo" data-link>Catálogo</a><span>/</span>
        <a href="/catalogo?categoria=${encodeURIComponent(ad.categoria)}" data-link>${escapeHTML(label(ad.categoria))}</a><span>/</span>
        <span>${escapeHTML(ad.titulo)}</span>
      </nav>
      <div class="ad-detail-layout">
        <div class="ad-gallery">
          <div class="ad-main-photo" id="ad-main-photo">${photoMarkup(photos[0], ad.titulo)}</div>
          ${photos.length > 1 ? `
            <div class="ad-thumbnails" aria-label="Fotos do anúncio">
              ${photos.map((photo, index) => `
                <button class="ad-thumbnail ${index === 0 ? "active" : ""}" type="button" data-photo="${index}" aria-label="Ver foto ${index + 1}">
                  ${photoMarkup(photo, "")}
                </button>`).join("")}
            </div>` : ""}
        </div>
        <aside class="ad-purchase-panel">
          <div class="ad-status-row">
            <span class="status-badge">${escapeHTML(label(ad.status))}</span>
            <span>Publicado em ${new Date(ad.criado_em).toLocaleDateString("pt-BR")}</span>
          </div>
          <span class="ad-category">${escapeHTML(label(ad.categoria))}</span>
          <h1>${escapeHTML(ad.titulo)}</h1>
          <strong class="ad-price">${money(ad.preco_centavos)}</strong>
          <dl class="ad-attributes">
            <div><dt>Tamanho</dt><dd>${escapeHTML(ad.tamanho)}</dd></div>
            <div><dt>Cor</dt><dd>${escapeHTML(label(ad.cor))}</dd></div>
            <div><dt>Conservação</dt><dd>${escapeHTML(label(ad.estado_conservacao))}</dd></div>
          </dl>
          ${purchaseAction(ad, isOwnAd, isAvailable)}
          <div class="ad-assurances">
            <p><strong>Peça única</strong><span>Este anúncio representa uma única unidade.</span></p>
            <p><strong>Compra em desenvolvimento</strong><span>Checkout e pagamento serão liberados na próxima fase.</span></p>
          </div>
        </aside>
      </div>
      <div class="ad-description-layout">
        <section class="ad-description">
          <span class="eyebrow">Sobre a peça</span>
          <h2>Detalhes do anúncio</h2>
          <p>${escapeHTML(ad.descricao)}</p>
        </section>
        <aside class="seller-card">
          <span class="seller-avatar">${isOwnAd ? "Você" : "R"}</span>
          <div><span class="eyebrow">${isOwnAd ? "Seu anúncio" : "Publicado na ReVeste"}</span><h3>${isOwnAd ? "Esta peça é sua" : "Venda de pessoa para pessoa"}</h3><p>${isOwnAd ? "Acompanhe o status pelo seu painel." : "Os dados públicos do vendedor serão adicionados com o perfil público."}</p></div>
        </aside>
      </div>
    </section>`;

  bindGallery(root, photos, ad.titulo);
  root.querySelector("[data-detail-add-cart]")?.addEventListener("click", async (event) => {
    if (!state.token) {
      toast("Entre na sua conta para adicionar esta peça.");
      navigate(`/entrar?retorno=${encodeURIComponent(window.location.pathname)}`);
      return;
    }
    const button = event.currentTarget;
    button.disabled = true;
    try {
      await addToCart(ad.id);
      button.textContent = "Peça adicionada";
    } catch (error) {
      toast(error.message);
      button.disabled = false;
    }
  });
}

function purchaseAction(ad, isOwnAd, isAvailable) {
  if (isOwnAd) {
    return `<a class="button button-dark button-large ad-primary-action" href="/meus-anuncios" data-link>Ver no meu painel</a>`;
  }
  if (!isAvailable) {
    return `<button class="button button-dark button-large ad-primary-action" type="button" disabled>Peça ${escapeHTML(label(ad.status))}</button>`;
  }
  const alreadyInCart = state.cart.anuncios?.some((item) => item.id === ad.id);
  return `<button class="button button-dark button-large ad-primary-action" type="button" data-detail-add-cart ${alreadyInCart ? "disabled" : ""}>${alreadyInCart ? "Já está na sacola" : "Adicionar à sacola"}</button>`;
}

function bindGallery(root, photos, title) {
  root.querySelectorAll("[data-photo]").forEach((button) => {
    button.addEventListener("click", () => {
      root.querySelector("#ad-main-photo").innerHTML = photoMarkup(photos[Number(button.dataset.photo)], title);
      root.querySelectorAll("[data-photo]").forEach((item) => item.classList.toggle("active", item === button));
    });
  });
}

function photoMarkup(photo, alt) {
  return photo && isPublicImageURL(photo.url)
    ? `<img src="${escapeHTML(photo.url)}" alt="${escapeHTML(alt)}">`
    : placeholderSVG();
}

function label(value = "") {
  return String(value).replaceAll("_", " ").replace(/^\w/, (letter) => letter.toUpperCase());
}

adDetailPage.title = "Detalhes do anúncio";
