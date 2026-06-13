import { request } from "../core/api.js";
import { addToCart } from "../core/cart.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { state } from "../core/state.js";
import { escapeHTML } from "../core/utils.js";
import { emptyState, productCard } from "../components/products.js";
import { pageSkeleton } from "../core/feedback.js";

export async function sellerProfilePage(root, { idVendedor }) {
  root.innerHTML = pageSkeleton("Carregando vendedor");
  try {
    const response = await request(`/v1/vendedores/${encodeURIComponent(idVendedor)}`);
    const seller = response.vendedor;
    const ads = response.anuncios || [];
    const initials = seller.nome.split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
    document.title = `ReVeste | ${seller.nome}`;
    root.innerHTML = `
      <section class="seller-profile-page page-section compact-section">
        <header class="public-profile-header">
          <div class="profile-avatar">${escapeHTML(initials)}</div>
          <div><span class="eyebrow">Perfil do vendedor</span><h1>${escapeHTML(seller.nome)}</h1><p>${escapeHTML(seller.cidade)}, ${escapeHTML(seller.estado)} · Na ReVeste desde ${new Date(seller.membro_desde).toLocaleDateString("pt-BR", { month: "long", year: "numeric" })}</p></div>
          <div class="public-profile-stat"><strong>${ads.length}</strong><span>${ads.length === 1 ? "peça disponível" : "peças disponíveis"}</span></div>
        </header>
        <div class="section-heading"><div><span class="eyebrow">Arara de ${escapeHTML(seller.nome.split(" ")[0])}</span><h2>Peças disponíveis</h2></div></div>
        <div class="product-grid" id="seller-products">${ads.length ? ads.map((ad) => productCard(ad, { allowCart: state.user?.id !== seller.id })).join("") : emptyState("Nenhuma peça disponível", "Este vendedor não possui anúncios ativos no momento.")}</div>
      </section>`;
    root.querySelectorAll("[data-add-cart]").forEach((button) => {
      button.addEventListener("click", async () => {
        if (!state.token) {
          toast("Entre na sua conta para adicionar peças.");
          navigate(`/entrar?retorno=${encodeURIComponent(window.location.pathname)}`);
          return;
        }
        button.disabled = true;
        button.classList.add("is-loading");
        button.setAttribute("aria-busy", "true");
        try {
          await addToCart(button.dataset.addCart);
          button.classList.remove("is-loading");
          button.removeAttribute("aria-busy");
          button.classList.add("is-added");
          button.setAttribute("aria-label", "Peça já está na sacola");
        } catch (error) {
          button.classList.remove("is-loading");
          button.removeAttribute("aria-busy");
          button.disabled = false;
          toast(error.message);
        }
      });
    });
  } catch (error) {
    root.innerHTML = `<section class="page-section compact-section">${emptyState("Vendedor não encontrado", error.message, '<a class="button button-dark" href="/catalogo" data-link>Voltar ao catálogo</a>')}</section>`;
  }
}

sellerProfilePage.title = "Perfil do vendedor";
