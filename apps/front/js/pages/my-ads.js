import { request } from "../core/api.js";
import { toast } from "../core/notifications.js";
import { adImage, escapeHTML, money } from "../core/utils.js";
import { emptyState } from "../components/products.js";

export async function myAdsPage(root) {
  root.innerHTML = `
    <section class="dashboard-page page-section compact-section">
      <div class="section-heading">
        <div><span class="eyebrow">Área de venda</span><h1>Meus anúncios</h1><p id="ads-summary">Carregando suas peças...</p></div>
        <a class="button button-dark" href="/vender" data-link>Publicar nova peça</a>
      </div>
      <div class="seller-stats">
        <div><span>Publicados</span><strong id="published-count">—</strong></div>
        <div><span>Disponíveis</span><strong id="available-count">—</strong></div>
        <div><span>Pendências de envio</span><strong>0</strong></div>
      </div>
      <div class="my-ads-grid" id="my-ads-list"></div>
    </section>`;
  try {
    const response = await request("/v1/me/anuncios");
    const ads = response.dados || [];
    root.querySelector("#ads-summary").textContent = `${ads.length} ${ads.length === 1 ? "peça publicada" : "peças publicadas"}`;
    root.querySelector("#published-count").textContent = ads.length;
    root.querySelector("#available-count").textContent = ads.filter((ad) => ad.status === "disponivel").length;
    root.querySelector("#my-ads-list").innerHTML = ads.length
      ? ads.map((ad) => `
          <a class="my-ad-card" href="/anuncios/${encodeURIComponent(ad.id)}" data-link>
            <div class="my-ad-image">${adImage(ad)}</div>
            <div class="my-ad-content">
              <span class="status-badge">${escapeHTML(ad.status.replaceAll("_", " "))}</span>
              <h2>${escapeHTML(ad.titulo)}</h2>
              <p>${escapeHTML(ad.categoria.replaceAll("_", " "))} · Tam. ${escapeHTML(ad.tamanho)}</p>
              <strong>${money(ad.preco_centavos)}</strong>
              <small>Publicado em ${new Date(ad.criado_em).toLocaleDateString("pt-BR")}</small>
            </div>
          </a>`).join("")
      : emptyState("Nenhuma peça publicada", "Seu primeiro anúncio aparecerá aqui.", '<a class="button button-dark" href="/vender" data-link>Publicar primeira peça</a>');
  } catch (error) {
    toast(error.message);
    root.querySelector("#my-ads-list").innerHTML = emptyState("Não foi possível carregar seus anúncios", error.message);
  }
}

myAdsPage.title = "Meus anúncios";
