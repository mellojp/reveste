import { request } from "../core/api.js";
import { toast } from "../core/notifications.js";
import { adImage, escapeHTML, money } from "../core/utils.js";
import { emptyState } from "../components/products.js";
import { gridSkeleton, revealContent, setButtonLoading } from "../core/feedback.js";

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
      <div class="my-ads-grid" id="my-ads-list">${gridSkeleton(4)}</div>
    </section>`;
  try {
    const response = await request("/v1/me/anuncios");
    renderAds(root, response.dados || []);
  } catch (error) {
    toast(error.message);
    root.querySelector("#my-ads-list").innerHTML = emptyState("Não foi possível carregar seus anúncios", error.message);
  }
}

function renderAds(root, ads) {
  root.querySelector("#ads-summary").textContent = `${ads.length} ${ads.length === 1 ? "peça publicada" : "peças publicadas"}`;
  root.querySelector("#published-count").textContent = ads.length;
  root.querySelector("#available-count").textContent = ads.filter((ad) => ad.status === "disponivel").length;
  const list = root.querySelector("#my-ads-list");
  list.innerHTML = ads.length
    ? ads.map((ad) => `
        <article class="my-ad-card" data-ad-card="${escapeHTML(ad.id)}">
          <a class="my-ad-image" href="/anuncios/${encodeURIComponent(ad.id)}" data-link>${adImage(ad)}</a>
          <div class="my-ad-content">
            <span class="status-badge">${escapeHTML(ad.status.replaceAll("_", " "))}</span>
            <h2><a href="/anuncios/${encodeURIComponent(ad.id)}" data-link>${escapeHTML(ad.titulo)}</a></h2>
            <p>${escapeHTML(ad.categoria.replaceAll("_", " "))} · Tam. ${escapeHTML(ad.tamanho)}</p>
            <strong>${money(ad.preco_centavos)}</strong>
            <small>Publicado em ${new Date(ad.criado_em).toLocaleDateString("pt-BR")}</small>
            <div class="my-ad-actions">
              <a class="text-link" href="/anuncios/${encodeURIComponent(ad.id)}" data-link>Visualizar</a>
              ${ad.status === "disponivel" ? `<a class="text-link" href="/meus-anuncios/${encodeURIComponent(ad.id)}/editar" data-link>Editar</a><button class="danger-link" type="button" data-delete-ad="${escapeHTML(ad.id)}">Excluir</button>` : ""}
            </div>
          </div>
        </article>`).join("")
    : emptyState("Nenhuma peça publicada", "Seu primeiro anúncio aparecerá aqui.", '<a class="button button-dark" href="/vender" data-link>Publicar primeira peça</a>');
  revealContent(list);

  list.querySelectorAll("[data-delete-ad]").forEach((button) => {
    button.addEventListener("click", async () => {
      if (button.dataset.confirming !== "true") {
        button.dataset.confirming = "true";
        button.textContent = "Confirmar exclusão";
        setTimeout(() => {
          if (button.isConnected) {
            button.dataset.confirming = "false";
            button.textContent = "Excluir";
          }
        }, 5000);
        return;
      }
      setButtonLoading(button, true, "Excluindo...");
      try {
        await request(`/v1/me/anuncios/${encodeURIComponent(button.dataset.deleteAd)}`, { method: "DELETE" });
        toast("Anúncio excluído.");
        renderAds(root, ads.filter((ad) => ad.id !== button.dataset.deleteAd));
      } catch (error) {
        setButtonLoading(button, false);
        toast(error.message);
      }
    });
  });
}

myAdsPage.title = "Meus anúncios";
