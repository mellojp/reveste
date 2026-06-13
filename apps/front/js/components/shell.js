import { state } from "../core/state.js";

export function renderShell() {
  renderHeader();
  document.querySelector("#site-footer").innerHTML = `
    <div class="footer-main">
      <div class="footer-intro">
        <a class="brand footer-brand" href="/" data-link aria-label="Página inicial ReVeste">
          <img src="/assets/logo-light.svg" alt="ReVeste">
        </a>
        <p>Moda de pessoa para pessoa. Peças especiais continuam circulando e ganhando novas histórias.</p>
      </div>
      <nav class="footer-links" aria-label="Links do catálogo">
        <strong>Descobrir</strong>
        <a href="/catalogo" data-link>Explorar peças</a>
        <a href="/catalogo?categoria=vestidos" data-link>Vestidos</a>
        <a href="/catalogo?categoria=casacos" data-link>Casacos</a>
      </nav>
      <nav class="footer-links" aria-label="Links para vendedores">
        <strong>Vender</strong>
        <a href="/vender" data-link>Publicar peça</a>
        <a href="/meus-anuncios" data-link>Meus anúncios</a>
        <a href="/perfil" data-link>Minha conta</a>
      </nav>
      <div class="footer-message">
        <span class="eyebrow">Escolhas que circulam</span>
        <p>Menos peças paradas. Mais estilo em movimento.</p>
      </div>
    </div>
    <div class="footer-bottom">
      <p>© 2026 ReVeste</p>
      <p>Projeto MVP · Moda circular</p>
    </div>
  `;
}

export function renderHeader() {
  const header = document.querySelector("#site-header");
  const firstName = state.user?.nome?.split(" ")[0];
  const currentPath = window.location.pathname;
  const current = (path) => currentPath === path || (path !== "/" && currentPath.startsWith(path));
  header.className = "site-header";
  header.innerHTML = `
    <a class="brand" href="/" data-link aria-label="Página inicial ReVeste">
      <img src="/assets/logo.svg" alt="ReVeste">
    </a>
    <nav class="main-nav" aria-label="Navegação principal">
      <a href="/catalogo" data-link ${current("/catalogo") || current("/anuncios/") ? 'aria-current="page"' : ""}>Explorar</a>
      <a href="/vender" data-link ${current("/vender") ? 'aria-current="page"' : ""}>Vender</a>
      ${state.token ? `<a href="/meus-anuncios" data-link ${current("/meus-anuncios") ? 'aria-current="page"' : ""}>Meus anúncios</a>` : ""}
    </nav>
    <div class="header-actions">
      <a class="icon-button cart-button" href="/carrinho" data-link aria-label="Abrir sacola">
        <svg aria-hidden="true" viewBox="0 0 24 24"><path d="M6 8h12l1 13H5L6 8Zm3 2V6a3 3 0 0 1 6 0v4"/></svg>
        <span class="badge">${state.cart.anuncios?.length || 0}</span>
      </a>
      <a class="button button-dark" href="${state.token ? "/perfil" : "/entrar"}" data-link>
        <span>${firstName || "Entrar"}</span>
      </a>
    </div>
  `;
}
