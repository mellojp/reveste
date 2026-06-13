import { request } from "../core/api.js";
import { addToCart } from "../core/cart.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { state } from "../core/state.js";
import { categories, conditions, escapeHTML } from "../core/utils.js";
import { emptyState, productCard } from "../components/products.js";

export async function catalogPage(root) {
  const current = new URLSearchParams(window.location.search);
  root.innerHTML = `
    <section class="catalog-header page-section compact-section">
      <span class="eyebrow">Catálogo ReVeste</span>
      <h1>Encontre a próxima peça favorita.</h1>
      <p>Itens únicos, publicados por pessoas como você.</p>
    </section>
    <section class="catalog-layout page-section">
      <aside class="filter-panel">
        <div class="filter-title"><h2>Filtros</h2><button class="link-button" type="button" id="clear-filters">Limpar</button></div>
        <form id="catalog-filters">
          <label>Buscar
            <input type="search" name="q" value="${escapeHTML(current.get("q") || "")}" placeholder="Ex.: vestido azul">
          </label>
          <label>Categoria
            <select name="categoria">${categories.map(([value, label]) => `<option value="${value}" ${current.get("categoria") === value ? "selected" : ""}>${label}</option>`).join("")}</select>
          </label>
          <label>Conservação
            <select name="estado_conservacao">${conditions.map(([value, label]) => `<option value="${value}" ${current.get("estado_conservacao") === value ? "selected" : ""}>${label}</option>`).join("")}</select>
          </label>
          <button class="button button-dark" type="submit">Aplicar filtros</button>
        </form>
      </aside>
      <div class="catalog-results">
        <div class="results-heading"><p id="catalog-summary">Buscando peças...</p></div>
        <div class="product-grid" id="catalog-products"><div class="page-loading">Carregando...</div></div>
      </div>
    </section>
  `;

  const form = root.querySelector("#catalog-filters");
  form.addEventListener("submit", (event) => {
    event.preventDefault();
    const params = new URLSearchParams();
    for (const [key, value] of new FormData(form)) {
      if (value) params.set(key, value);
    }
    navigate(`/catalogo${params.size ? `?${params}` : ""}`, { replace: true });
  });
  root.querySelector("#clear-filters").addEventListener("click", () => navigate("/catalogo", { replace: true }));
  await loadCatalog(root, current);
}

async function loadCatalog(root, filters) {
  const summary = root.querySelector("#catalog-summary");
  const grid = root.querySelector("#catalog-products");
  const params = new URLSearchParams({ limite: "48" });
  for (const key of ["q", "categoria", "estado_conservacao"]) {
    if (filters.get(key)) params.set(key, filters.get(key));
  }
  try {
    const response = await request(`/v1/anuncios?${params}`);
    state.catalog = response.dados || [];
    summary.textContent = `${state.catalog.length} ${state.catalog.length === 1 ? "peça encontrada" : "peças encontradas"}`;
    grid.innerHTML = state.catalog.length
      ? state.catalog.map((ad) => productCard(ad)).join("")
      : emptyState("Nenhuma peça encontrada", "Tente remover ou alterar algum filtro.", '<button class="button button-dark" id="empty-clear">Limpar filtros</button>');
    root.querySelector("#empty-clear")?.addEventListener("click", () => navigate("/catalogo", { replace: true }));
    bindCartButtons(grid);
  } catch (error) {
    summary.textContent = "Catálogo indisponível";
    grid.innerHTML = emptyState("Não foi possível carregar o catálogo", error.message);
  }
}

function bindCartButtons(root) {
  root.querySelectorAll("[data-add-cart]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (!state.token) {
        navigate(`/entrar?retorno=${encodeURIComponent(`${window.location.pathname}${window.location.search}`)}`);
        toast("Entre na sua conta para adicionar peças.");
        return;
      }
      button.disabled = true;
      try {
        await addToCart(button.dataset.addCart);
      } catch (error) {
        toast(error.message);
      } finally {
        button.disabled = false;
      }
    });
  });
}

catalogPage.title = "Catálogo";
