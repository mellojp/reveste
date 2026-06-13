import { request } from "../core/api.js";
import { addToCart } from "../core/cart.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { state } from "../core/state.js";
import { escapeHTML, isPublicImageURL, money, placeholderSVG } from "../core/utils.js";
import { emptyState } from "../components/products.js";
import { pageSkeleton, revealContent, setButtonLoading } from "../core/feedback.js";

export async function adDetailPage(root, { idAnuncio }) {
  root.innerHTML = pageSkeleton("Carregando anúncio");
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
  const seller = ad.vendedor;
  const sellerInitials = seller?.nome?.split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase() || "R";
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
          <span class="seller-avatar">${isOwnAd ? "Você" : escapeHTML(sellerInitials)}</span>
          <div>
            <span class="eyebrow">${isOwnAd ? "Seu anúncio" : "Vendido por"}</span>
            <h3>${escapeHTML(isOwnAd ? "Esta peça é sua" : seller?.nome || "Vendedor ReVeste")}</h3>
            <p>${isOwnAd ? "Acompanhe ou edite a peça pelo seu painel." : `${escapeHTML(seller?.cidade || "")}${seller?.estado ? `, ${escapeHTML(seller.estado)}` : ""} · membro desde ${seller?.membro_desde ? new Date(seller.membro_desde).getFullYear() : "2026"}`}</p>
            ${isOwnAd ? '<a class="text-link" href="/meus-anuncios" data-link>Gerenciar anúncio</a>' : `<a class="text-link" href="/vendedores/${encodeURIComponent(ad.id_vendedor)}" data-link>Ver perfil e outras peças</a>`}
          </div>
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
    setButtonLoading(button, true, "Adicionando...");
    try {
      await addToCart(ad.id);
      setButtonLoading(button, false);
      button.textContent = "Peça adicionada";
      button.disabled = true;
      button.classList.add("is-success");
    } catch (error) {
      toast(error.message);
      setButtonLoading(button, false);
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
      const mainPhoto = root.querySelector("#ad-main-photo");
      mainPhoto.innerHTML = photoMarkup(photos[Number(button.dataset.photo)], title);
      revealContent(mainPhoto);
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
