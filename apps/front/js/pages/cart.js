import { removeFromCart } from "../core/cart.js";
import { toast } from "../core/notifications.js";
import { loadCart } from "../core/session.js";
import { adImage, escapeHTML, money } from "../core/utils.js";
import { emptyState } from "../components/products.js";
import { pageSkeleton, setButtonLoading } from "../core/feedback.js";

export async function cartPage(root) {
  root.innerHTML = pageSkeleton("Carregando sacola");
  try {
    const cart = await loadCart();
    renderCart(root, cart);
  } catch (error) {
    root.innerHTML = `<section class="cart-page page-section compact-section">${emptyState("Não foi possível carregar sua sacola", error.message)}</section>`;
  }
}

function renderCart(root, cart) {
  const ads = cart.anuncios || [];
  const availableAds = ads.filter((ad) => ad.status === "disponivel");
  const unavailableCount = ads.length - availableAds.length;
  root.innerHTML = `
    <section class="cart-page page-section compact-section">
      <div class="page-intro"><span class="eyebrow">Sua seleção</span><h1>Minha sacola</h1><p>${ads.length} ${ads.length === 1 ? "peça adicionada" : "peças adicionadas"} para decidir. A disponibilidade é confirmada somente no checkout.</p></div>
      ${unavailableCount ? `<div class="inline-alert" role="status">${unavailableCount} ${unavailableCount === 1 ? "peça ficou indisponível e não entra" : "peças ficaram indisponíveis e não entram"} no total.</div>` : ""}
      ${ads.length ? `
        <div class="cart-layout">
          <div class="cart-list">
            ${ads.map((ad) => `
              <article class="cart-item">
                <a class="cart-thumb" href="/anuncios/${encodeURIComponent(ad.id)}" data-link>${adImage(ad)}</a>
                <div class="cart-item-info">
                  <span>${escapeHTML(ad.categoria.replaceAll("_", " "))} · Tam. ${escapeHTML(ad.tamanho)}</span>
                  <h2><a href="/anuncios/${encodeURIComponent(ad.id)}" data-link>${escapeHTML(ad.titulo)}</a></h2>
                  <p>${escapeHTML(ad.estado_conservacao.replaceAll("_", " "))}</p>
                  ${ad.status !== "disponivel" ? `<span class="unavailable-label">Indisponível</span>` : ""}
                  <strong>${money(ad.preco_centavos)}</strong>
                </div>
                <button class="remove-item" type="button" data-remove-cart="${escapeHTML(ad.id)}">Remover</button>
              </article>`).join("")}
          </div>
          <aside class="cart-summary">
            <span class="eyebrow">Resumo</span><h2>Seu pedido</h2>
            <div><span>Itens disponíveis (${availableAds.length})</span><strong>${money(cart.total_centavos)}</strong></div>
            <div><span>Frete</span><span>Calculado no checkout</span></div>
            <div class="cart-total"><span>Total dos itens</span><strong>${money(cart.total_centavos)}</strong></div>
            <button class="button button-dark button-large" type="button" disabled>Checkout em breve</button>
            <p>O fluxo de compra será liberado após a implementação de pedidos e pagamento na API.</p>
          </aside>
        </div>` :
        emptyState("Sua sacola está vazia", "Explore o catálogo e adicione as peças que combinam com você.", '<a class="button button-dark" href="/catalogo" data-link>Explorar catálogo</a>')}
    </section>`;

  root.querySelectorAll("[data-remove-cart]").forEach((button) => {
    button.addEventListener("click", async () => {
      setButtonLoading(button, true, "Removendo...");
      try {
        const updated = await removeFromCart(button.dataset.removeCart);
        renderCart(root, updated);
      } catch (error) {
        setButtonLoading(button, false);
        toast(error.message);
      }
    });
  });
}

cartPage.title = "Minha sacola";
