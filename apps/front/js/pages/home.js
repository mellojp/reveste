import { request } from "../core/api.js";
import { addToCart } from "../core/cart.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { state } from "../core/state.js";
import { categories } from "../core/utils.js";
import { emptyState, productCard } from "../components/products.js";
import { gridSkeleton, revealContent } from "../core/feedback.js";

export async function homePage(root) {
  root.innerHTML = `
    <section class="home-hero">
      <div class="home-hero-inner">
        <div class="hero-copy">
          <span class="eyebrow">Moda circular, escolhas únicas</span>
          <h1>Vista novas histórias.</h1>
          <p>Peças especiais merecem continuar circulando. Compre de outras pessoas, renove seu estilo e reduza o impacto no planeta.</p>
          <div class="hero-actions">
            <a class="button button-light button-large" href="/catalogo" data-link>Explorar peças</a>
            <a class="hero-link" href="/vender" data-link>Desapegar agora <span>↗</span></a>
          </div>
          <div class="hero-proof">
            <span><strong>Curadoria pessoal</strong> para escolhas únicas</span>
            <span><strong>Peças únicas</strong> de outras pessoas</span>
          </div>
        </div>
        <div class="hero-art" aria-hidden="true">
          <div class="hero-sun"></div>
          <div class="hero-arch hero-arch-one"></div>
          <div class="hero-arch hero-arch-two"></div>
          <div class="clothing-card card-shirt">
            <svg viewBox="0 0 180 190"><path d="M58 30 76 18h28l18 12 34 19-19 35-17-9v89H60V75l-17 9-19-35 34-19Z"/><path d="M76 18c0 18 28 18 28 0"/></svg>
          </div>
          <div class="clothing-card card-dress">
            <svg viewBox="0 0 180 220"><path d="M72 20h36l11 47-13 25 39 105H35L74 92 61 67l11-47Z"/><path d="M72 20c3 22 33 22 36 0"/></svg>
          </div>
          <span class="hero-spark spark-one">✦</span>
          <span class="hero-spark spark-two">✦</span>
        </div>
      </div>
    </section>

    <section class="home-benefits">
      <div><strong>01</strong><span>Encontre uma peça</span><p>Explore anúncios únicos e filtre pelo seu estilo.</p></div>
      <div><strong>02</strong><span>Guarde na sacola</span><p>Organize suas escolhas antes de decidir.</p></div>
      <div><strong>03</strong><span>Faça circular</span><p>Publique o que não usa mais em poucos minutos.</p></div>
    </section>

    <section class="page-section category-section home-categories">
      <div class="section-heading">
        <div><span class="eyebrow">Encontre seu estilo</span><h2>Explore por categoria</h2></div>
        <a class="text-link" href="/catalogo" data-link>Ver catálogo completo</a>
      </div>
      <div class="category-row">
        ${categories.map(([value, label], index) => `
          <a class="category-chip ${index === 0 ? "active" : ""}" href="/catalogo${value ? `?categoria=${value}` : ""}" data-link>${label}</a>`).join("")}
      </div>
    </section>

    <section class="page-section latest-section home-latest">
      <div class="section-heading">
        <div><span class="eyebrow">Recém-publicadas</span><h2>Novas histórias por aqui</h2></div>
        <a class="text-link" href="/catalogo" data-link>Ver todas</a>
      </div>
      <div class="product-grid" id="home-products">${gridSkeleton(4)}</div>
    </section>

    <section class="page-section circular-banner">
      <div><span class="eyebrow">Seu armário pode circular</span><h2>Desapegue do que não usa mais.</h2><p>Publique em poucos minutos e encontre alguém para continuar a história da peça.</p></div>
      <a class="button button-light button-large" href="/vender" data-link>Publicar uma peça</a>
    </section>
  `;

  const grid = root.querySelector("#home-products");
  try {
    const response = await request("/v1/anuncios?limite=8");
    const ads = response.dados || [];
    grid.innerHTML = ads.length
      ? ads.map((ad) => productCard(ad)).join("")
      : emptyState("O catálogo está começando", "Seja a primeira pessoa a publicar uma peça.", '<a class="button button-dark" href="/vender" data-link>Publicar peça</a>');
    revealContent(grid);
    bindCartButtons(grid);
  } catch (error) {
    grid.innerHTML = emptyState("Não foi possível carregar as peças", error.message);
  }
}

function bindCartButtons(root) {
  root.querySelectorAll("[data-add-cart]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!state.token) {
        navigate(`/entrar?retorno=${encodeURIComponent(window.location.pathname)}`);
        toast("Entre na sua conta para adicionar peças.");
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
        toast(error.message);
        button.classList.remove("is-loading");
        button.removeAttribute("aria-busy");
        button.disabled = false;
      }
    });
  });
}

homePage.title = "Início";
