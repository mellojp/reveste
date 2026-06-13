import { request } from "../core/api.js";
import { addToCart } from "../core/cart.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { state } from "../core/state.js";
import { categories, conditions, escapeHTML } from "../core/utils.js";
import { emptyState, productCard } from "../components/products.js";
import { gridSkeleton, revealContent, setButtonLoading } from "../core/feedback.js";

export async function catalogPage(root) {
  const current = new URLSearchParams(window.location.search);
  let offset = 0;
  const pageSize = 24;
  root.innerHTML = `
    <section class="catalog-header page-section compact-section">
      <span class="eyebrow">Catálogo ReVeste</span>
      <h1>Encontre a próxima peça favorita.</h1>
      <p>Itens únicos, publicados por pessoas como você.</p>
    </section>
    <section class="catalog-layout page-section">
      <button class="filter-mobile-toggle button button-dark" type="button" id="filter-toggle" aria-expanded="false">Filtros</button>
      <aside class="filter-panel" id="filter-panel">
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
          <label>Tamanho
            <input name="tamanho" value="${escapeHTML(current.get("tamanho") || "")}" placeholder="Ex.: M ou 38">
          </label>
          <div class="price-filter">
            <label>Preço mínimo<input name="preco_min" inputmode="decimal" value="${escapeHTML(current.get("preco_min") || "")}" placeholder="R$ 0"></label>
            <label>Preço máximo<input name="preco_max" inputmode="decimal" value="${escapeHTML(current.get("preco_max") || "")}" placeholder="R$ 500"></label>
          </div>
          <button class="button button-dark" type="submit">Aplicar filtros</button>
        </form>
      </aside>
      <div class="catalog-results">
        <div class="results-heading"><p id="catalog-summary">Buscando peças...</p><span>Mais recentes primeiro</span></div>
        <div class="product-grid" id="catalog-products">${gridSkeleton(8)}</div>
        <div class="catalog-more"><button class="button button-outline hidden" type="button" id="load-more">Carregar mais</button></div>
      </div>
    </section>
  `;

  const form = root.querySelector("#catalog-filters");
  form.addEventListener("submit", (event) => {
    event.preventDefault();
    setButtonLoading(form.querySelector("button[type=submit]"), true, "Aplicando...");
    const params = new URLSearchParams();
    for (const [key, value] of new FormData(form)) {
      if (value) params.set(key, value);
    }
    navigate(`/catalogo${params.size ? `?${params}` : ""}`, {
      replace: true,
      preserveScroll: true,
    });
  });
  root.querySelector("#clear-filters").addEventListener("click", () => navigate("/catalogo", {
    replace: true,
    preserveScroll: true,
  }));
  root.querySelector("#filter-toggle").addEventListener("click", (event) => {
    const panel = root.querySelector("#filter-panel");
    const open = panel.classList.toggle("is-open");
    event.currentTarget.setAttribute("aria-expanded", String(open));
  });
  const moreButton = root.querySelector("#load-more");
  moreButton.addEventListener("click", async () => {
    offset += pageSize;
    await loadCatalog(root, current, { offset, pageSize, append: true });
  });
  await loadCatalog(root, current, { offset, pageSize });
}

async function loadCatalog(root, filters, { offset = 0, pageSize = 24, append = false } = {}) {
  const summary = root.querySelector("#catalog-summary");
  const grid = root.querySelector("#catalog-products");
  const moreButton = root.querySelector("#load-more");
  const params = new URLSearchParams({ limite: String(pageSize), deslocamento: String(offset) });
  for (const key of ["q", "categoria", "estado_conservacao", "tamanho"]) {
    if (filters.get(key)) params.set(key, filters.get(key));
  }
  const minimum = priceToCents(filters.get("preco_min"));
  const maximum = priceToCents(filters.get("preco_max"));
  if (minimum) params.set("preco_min_centavos", String(minimum));
  if (maximum) params.set("preco_max_centavos", String(maximum));
  const filterButton = root.querySelector("#catalog-filters button[type=submit]");
  if (!append) grid.innerHTML = gridSkeleton(8);
  setButtonLoading(moreButton, true, "Carregando...");
  try {
    const response = await request(`/v1/anuncios?${params}`);
    const page = response.dados || [];
    state.catalog = append ? [...state.catalog, ...page] : page;
    summary.textContent = `${state.catalog.length} ${state.catalog.length === 1 ? "peça encontrada" : "peças encontradas"}`;
    grid.innerHTML = state.catalog.length
      ? state.catalog.map((ad) => productCard(ad)).join("")
      : emptyState("Nenhuma peça encontrada", "Tente remover ou alterar algum filtro.", '<button class="button button-dark" id="empty-clear">Limpar filtros</button>');
    revealContent(grid);
    moreButton.classList.toggle("hidden", page.length < pageSize);
    root.querySelector("#empty-clear")?.addEventListener("click", () => navigate("/catalogo", {
      replace: true,
      preserveScroll: true,
    }));
    bindCartButtons(grid);
  } catch (error) {
    summary.textContent = "Catálogo indisponível";
    grid.innerHTML = emptyState("Não foi possível carregar o catálogo", error.message);
  } finally {
    setButtonLoading(moreButton, false);
    setButtonLoading(filterButton, false);
  }
}

function priceToCents(value) {
  if (!value) return 0;
  const normalized = String(value).replace(/\./g, "").replace(",", ".");
  const number = Number(normalized);
  return Number.isFinite(number) && number > 0 ? Math.round(number * 100) : 0;
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
      button.classList.add("is-loading");
      button.setAttribute("aria-busy", "true");
      try {
        await addToCart(button.dataset.addCart);
        button.classList.remove("is-loading");
        button.removeAttribute("aria-busy");
        button.classList.add("is-added");
        button.disabled = true;
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

catalogPage.title = "Catálogo";
