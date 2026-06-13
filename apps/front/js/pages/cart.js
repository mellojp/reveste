import { removeFromCart } from "../core/cart.js";
import { toast } from "../core/notifications.js";
import { loadCart } from "../core/session.js";
import { adImage, escapeHTML, money } from "../core/utils.js";
import { emptyState } from "../components/products.js";

export async function cartPage(root) {
  root.innerHTML = `<section class="cart-page page-section compact-section"><div class="page-loading">Carregando sacola...</div></section>`;
  try {
    const cart = await loadCart();
    renderCart(root, cart);
  } catch (error) {
    root.innerHTML = `<section class="cart-page page-section compact-section">${emptyState("Não foi possível carregar sua sacola", error.message)}</section>`;
  }
}

function renderCart(root, cart) {
  const ads = cart.anuncios || [];
  root.innerHTML = `
    <section class="cart-page page-section compact-section">
      <div class="page-intro"><span class="eyebrow">Sua seleção</span><h1>Minha sacola</h1><p>${ads.length} ${ads.length === 1 ? "peça reservada" : "peças reservadas"} para decidir.</p></div>
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
                  <strong>${money(ad.preco_centavos)}</strong>
                </div>
                <button class="remove-item" type="button" data-remove-cart="${escapeHTML(ad.id)}">Remover</button>
              </article>`).join("")}
          </div>
          <aside class="cart-summary">
            <span class="eyebrow">Resumo</span><h2>Seu pedido</h2>
            <div><span>Itens (${ads.length})</span><strong>${money(cart.total_centavos)}</strong></div>
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
      button.disabled = true;
      try {
        const updated = await removeFromCart(button.dataset.removeCart);
        renderCart(root, updated);
      } catch (error) {
        button.disabled = false;
        toast(error.message);
      }
    });
  });
}

cartPage.title = "Minha sacola";
