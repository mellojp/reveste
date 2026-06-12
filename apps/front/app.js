const API_BASE = "";

const state = {
  token: sessionStorage.getItem("reveste_token") || "",
  user: JSON.parse(sessionStorage.getItem("reveste_user") || "null"),
  ads: [],
  myAds: [],
  cart: { anuncios: [], total_centavos: 0 },
  filters: { q: "", categoria: "", estado_conservacao: "" },
};

const elements = {
  authModal: document.querySelector("#auth-modal"),
  sellModal: document.querySelector("#sell-modal"),
  cartModal: document.querySelector("#cart-modal"),
  profileModal: document.querySelector("#profile-modal"),
  sessionButton: document.querySelector("#session-button"),
  productGrid: document.querySelector("#product-grid"),
  catalogSummary: document.querySelector("#catalog-summary"),
  cartContent: document.querySelector("#cart-content"),
  cartCount: document.querySelector("#cart-count"),
  cartTotal: document.querySelector("#cart-total"),
  searchInput: document.querySelector("#search-input"),
  conditionFilter: document.querySelector("#condition-filter"),
  toastRegion: document.querySelector("#toast-region"),
  photoInput: document.querySelector("#photo-input"),
  photoPreview: document.querySelector("#photo-preview"),
  uploadStatus: document.querySelector("#upload-status"),
};

function money(cents) {
  return new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" }).format(cents / 100);
}

function escapeHTML(value = "") {
  return String(value).replace(/[&<>"']/g, (character) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#039;",
  })[character]);
}

function placeholderSVG() {
  return `<svg class="product-placeholder" viewBox="0 0 180 190" aria-hidden="true"><path d="M58 30 76 18h28l18 12 34 19-19 35-17-9v89H60V75l-17 9-19-35 34-19Z"/><path d="M76 18c0 18 28 18 28 0"/></svg>`;
}

function toast(message) {
  const openDialogs = [...document.querySelectorAll("dialog[open]")];
  const activeDialog = openDialogs.at(-1);
  let region = elements.toastRegion;

  if (activeDialog) {
    region = activeDialog.querySelector(".toast-region");
    if (!region) {
      region = document.createElement("div");
      region.className = "toast-region";
      region.setAttribute("aria-live", "polite");
      activeDialog.append(region);
    }
  }

  const notification = document.createElement("div");
  notification.className = "toast";
  notification.textContent = message;
  region.append(notification);
  setTimeout(() => notification.remove(), 4000);
}

class APIError extends Error {
  constructor(message, fields = {}) {
    super(message);
    this.name = "APIError";
    this.fields = fields;
  }
}

async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (options.body) headers["Content-Type"] = "application/json";
  if (state.token) headers.Authorization = `Bearer ${state.token}`;

  const response = await fetch(`${API_BASE}${path}`, { ...options, headers });
  const payload = response.status === 204 ? null : await response.json().catch(() => null);
  if (!response.ok) {
    if (response.status === 401 && state.token) clearSession();
    throw new APIError(
      payload?.mensagem || "Não foi possível concluir a operação.",
      payload?.campos || {},
    );
  }
  return payload;
}

function clearFormErrors(form) {
  form.querySelectorAll(".field-error").forEach((element) => element.remove());
  form.querySelectorAll("[aria-invalid=true]").forEach((field) => {
    field.removeAttribute("aria-invalid");
    field.removeAttribute("aria-describedby");
  });
}

function showFormErrors(form, fields = {}) {
  clearFormErrors(form);
  let firstInvalidField = null;

  Object.entries(fields).forEach(([path, message], index) => {
    const fieldName = path.split(".").at(-1);
    const field = form.elements.namedItem(fieldName);
    if (!(field instanceof HTMLElement)) return;

    const errorId = `${form.id}-${fieldName}-error-${index}`;
    const error = document.createElement("small");
    error.id = errorId;
    error.className = "field-error";
    error.textContent = message;
    field.setAttribute("aria-invalid", "true");
    field.setAttribute("aria-describedby", errorId);
    field.closest("label")?.append(error);
    firstInvalidField ||= field;
  });

  firstInvalidField?.focus();
}

function updateSessionUI() {
  if (state.user) {
    const firstName = state.user.nome.split(" ")[0];
    elements.sessionButton.textContent = firstName;
    elements.sessionButton.dataset.action = "open-profile";
  } else {
    elements.sessionButton.textContent = "Entrar";
    elements.sessionButton.dataset.action = "open-login";
    state.cart = { anuncios: [], total_centavos: 0 };
    renderCart();
  }
}

function saveSession(session) {
  state.token = session.token;
  state.user = session.usuario;
  sessionStorage.setItem("reveste_token", state.token);
  sessionStorage.setItem("reveste_user", JSON.stringify(state.user));
  updateSessionUI();
  loadCart();
  loadAds();
}

function clearSession() {
  state.token = "";
  state.user = null;
  sessionStorage.removeItem("reveste_token");
  sessionStorage.removeItem("reveste_user");
  updateSessionUI();
  loadAds();
}

async function logout() {
  try {
    await request("/v1/sessoes/atual", { method: "DELETE" });
  } catch {
    // A sessao local deve ser removida mesmo se ja tiver expirado no servidor.
  }
  if (elements.profileModal.open) elements.profileModal.close();
  clearSession();
  toast("Sessão encerrada.");
}

function authRequired(callback) {
  if (!state.token) {
    openAuth("login");
    toast("Entre na sua conta para continuar.");
    return;
  }
  callback();
}

function openAuth(tab = "login") {
  switchAuthTab(tab);
  elements.authModal.showModal();
}

function switchAuthTab(tab) {
  document.querySelectorAll("[data-auth-tab]").forEach((button) => {
    button.classList.toggle("active", button.dataset.authTab === tab);
  });
  document.querySelector("#login-form").classList.toggle("hidden", tab !== "login");
  document.querySelector("#register-form").classList.toggle("hidden", tab !== "register");
  document.querySelector("#auth-title").textContent = tab === "login" ? "Entre para continuar" : "Crie sua conta";
  document.querySelector("#auth-description").textContent = tab === "login"
    ? "Acesse sua conta para vender peças e organizar sua sacola."
    : "Uma conta serve para comprar e vender na ReVeste.";
}

function renderProfile(user) {
  const address = user.endereco_principal;
  const initials = user.nome.split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
  document.querySelector("#profile-avatar").textContent = initials;
  document.querySelector("#profile-name").textContent = user.nome;
  document.querySelector("#profile-email").textContent = user.email;
  document.querySelector("#profile-details").innerHTML = `
    <div><dt>Telefone</dt><dd>${escapeHTML(user.telefone || "Não informado")}</dd></div>
    <div><dt>Localização</dt><dd>${escapeHTML(`${address.cidade}, ${address.estado}`)}</dd></div>
    <div><dt>Endereço</dt><dd>${escapeHTML(`${address.logradouro}, ${address.numero}`)}</dd></div>
    <div><dt>CEP</dt><dd>${escapeHTML(address.cep)}</dd></div>
  `;
}

function renderMyAds() {
  document.querySelector("#my-ads-summary").textContent =
    `${state.myAds.length} ${state.myAds.length === 1 ? "peça publicada" : "peças publicadas"}`;
  document.querySelector("#my-ads-list").innerHTML = state.myAds.length
    ? state.myAds.map((ad) => `
      <article class="my-ad">
        <div class="my-ad-image">${adImage(ad)}</div>
        <div>
          <h4>${escapeHTML(ad.titulo)}</h4>
          <p>${escapeHTML(ad.categoria)} · ${money(ad.preco_centavos)}</p>
        </div>
        <span class="status-badge">${escapeHTML(ad.status)}</span>
      </article>
    `).join("")
    : `<div class="empty-state"><h3>Nenhuma peça publicada</h3><p>Seu primeiro anúncio aparecerá aqui.</p></div>`;
}

async function openProfile() {
  elements.profileModal.showModal();
  try {
    const [user, adsResponse] = await Promise.all([
      request("/v1/me"),
      request("/v1/me/anuncios"),
    ]);
    state.user = user;
    state.myAds = adsResponse.dados || [];
    sessionStorage.setItem("reveste_user", JSON.stringify(user));
    renderProfile(user);
    renderMyAds();
  } catch (error) {
    elements.profileModal.close();
    toast(error.message);
  }
}

function adImage(ad, className = "") {
  const url = ad.fotos?.[0]?.url;
  return isPublicImageURL(url)
    ? `<img class="${className}" src="${escapeHTML(url)}" alt="" loading="lazy" onerror="this.remove()">`
    : placeholderSVG();
}

function isPublicImageURL(value) {
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
}

function renderPhotoPreview() {
  const files = [...elements.photoInput.files];
  elements.photoPreview.innerHTML = "";
  files.forEach((file, index) => {
    const item = document.createElement("div");
    item.className = "photo-preview-item";
    const image = document.createElement("img");
    image.alt = `Prévia da foto ${index + 1}`;
    image.src = URL.createObjectURL(file);
    image.addEventListener("load", () => URL.revokeObjectURL(image.src), { once: true });
    const order = document.createElement("span");
    order.textContent = `${index + 1}`;
    item.append(image, order);
    elements.photoPreview.append(item);
  });
}

function validatePhotoFiles(files) {
  if (files.length < 2 || files.length > 5) {
    return "Selecione entre 2 e 5 imagens.";
  }
  const allowedTypes = new Set(["image/jpeg", "image/png", "image/webp"]);
  if (files.some((file) => !allowedTypes.has(file.type))) {
    return "Envie apenas imagens JPEG, PNG ou WebP.";
  }
  if (files.some((file) => file.size <= 0 || file.size > 5 * 1024 * 1024)) {
    return "Cada imagem deve ter no máximo 5 MB.";
  }
  return "";
}

async function uploadPhoto(file, index, total) {
  elements.uploadStatus.classList.remove("hidden");
  elements.uploadStatus.textContent = `Enviando foto ${index + 1} de ${total}...`;
  const authorization = await request("/v1/uploads/imagens/autorizacoes", {
    method: "POST",
    body: JSON.stringify({
      nome_arquivo: file.name,
      tipo: file.type,
      tamanho: file.size,
    }),
  });
  const storeId = authorization.token.split("_")[3];
  const uploadURL = new URL(authorization.url_upload);
  uploadURL.searchParams.set("pathname", authorization.pathname);
  let response;
  try {
    response = await fetch(uploadURL, {
      method: "PUT",
      body: file,
      headers: {
        Authorization: `Bearer ${authorization.token}`,
        "x-api-version": "12",
        "x-api-blob-request-id": `${storeId}:${Date.now()}:${crypto.randomUUID()}`,
        "x-api-blob-request-attempt": "0",
        "x-vercel-blob-store-id": storeId,
        "x-vercel-blob-access": "public",
        "x-content-type": file.type,
      },
    });
  } catch {
    throw new Error(
      "O upload foi recusado. Confirme que o Blob store da Vercel foi criado com acesso público.",
    );
  }
  const blob = await response.json().catch(() => null);
  if (!response.ok || !blob?.url) {
    const providerMessage = blob?.error?.message || "";
    if (providerMessage.includes("private store")) {
      throw new Error("O Blob store da Vercel precisa ter acesso público.");
    }
    throw new Error(providerMessage || `Não foi possível enviar ${file.name}.`);
  }
  return blob.url;
}

async function uploadPhotos(files) {
  const urls = [];
  for (const [index, file] of files.entries()) {
    urls.push(await uploadPhoto(file, index, files.length));
  }
  elements.uploadStatus.textContent = "Fotos enviadas. Finalizando anúncio...";
  return urls;
}

function renderAds() {
  elements.catalogSummary.textContent = `${state.ads.length} ${state.ads.length === 1 ? "peça encontrada" : "peças encontradas"}`;
  if (!state.ads.length) {
    elements.productGrid.innerHTML = `
      <div class="empty-state">
        <h3>Nenhuma peça por aqui ainda</h3>
        <p>Ajuste os filtros ou seja a primeira pessoa a publicar nesta seleção.</p>
      </div>`;
    return;
  }

  elements.productGrid.innerHTML = state.ads.map((ad) => `
    <article class="product-card">
      <div class="product-image">
        ${adImage(ad)}
        <span class="condition-tag">${escapeHTML(ad.estado_conservacao.replaceAll("_", " "))}</span>
        <button class="add-cart" data-add-cart="${escapeHTML(ad.id)}" aria-label="Adicionar ${escapeHTML(ad.titulo)} à sacola">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 8h12l1 13H5L6 8Zm3 2V6a3 3 0 0 1 6 0v4"/></svg>
        </button>
      </div>
      <div class="product-info">
        <span class="product-meta">${escapeHTML(ad.categoria)} · Tam. ${escapeHTML(ad.tamanho)}</span>
        <h3>${escapeHTML(ad.titulo)}</h3>
        <strong>${money(ad.preco_centavos)}</strong>
      </div>
    </article>
  `).join("");
}

async function loadAds() {
  elements.catalogSummary.textContent = "Carregando peças...";
  const params = new URLSearchParams({ limite: "24" });
  Object.entries(state.filters).forEach(([key, value]) => {
    if (value) params.set(key, value);
  });
  try {
    const response = await request(`/v1/anuncios?${params}`);
    state.ads = response.dados || [];
    renderAds();
  } catch (error) {
    state.ads = [];
    renderAds();
    elements.catalogSummary.textContent = "Catálogo indisponível";
    toast(error.message);
  }
}

function renderCart() {
  const ads = state.cart.anuncios || [];
  elements.cartCount.textContent = ads.length;
  elements.cartTotal.textContent = money(state.cart.total_centavos || 0);
  elements.cartContent.innerHTML = ads.length ? ads.map((ad) => `
    <article class="cart-item">
      <div class="cart-thumb">${adImage(ad)}</div>
      <div>
        <p>${escapeHTML(ad.categoria)} · ${escapeHTML(ad.tamanho)}</p>
        <h3>${escapeHTML(ad.titulo)}</h3>
        <strong>${money(ad.preco_centavos)}</strong>
      </div>
      <button class="remove-item" data-remove-cart="${escapeHTML(ad.id)}" aria-label="Remover item">×</button>
    </article>
  `).join("") : `
    <div class="empty-state">
      <h3>Sua sacola está vazia</h3>
      <p>Explore o catálogo e salve as peças que combinam com você.</p>
    </div>`;
}

async function loadCart() {
  if (!state.token) return;
  try {
    state.cart = await request("/v1/carrinho");
    renderCart();
  } catch (error) {
    toast(error.message);
  }
}

async function addToCart(adId) {
  try {
    state.cart = await request("/v1/carrinho/itens", {
      method: "POST",
      body: JSON.stringify({ id_anuncio: adId }),
    });
    renderCart();
    toast("Peça adicionada à sacola.");
  } catch (error) {
    toast(error.message);
  }
}

async function removeFromCart(adId) {
  try {
    state.cart = await request(`/v1/carrinho/itens/${encodeURIComponent(adId)}`, { method: "DELETE" });
    renderCart();
  } catch (error) {
    toast(error.message);
  }
}

document.addEventListener("click", (event) => {
  const action = event.target.closest("[data-action]")?.dataset.action;
  if (action === "open-login") openAuth("login");
  if (action === "open-profile") authRequired(openProfile);
  if (action === "logout") logout();
  if (action === "profile-sell") {
    elements.profileModal.close();
    elements.sellModal.showModal();
  }
  if (action === "open-sell") authRequired(() => elements.sellModal.showModal());
  if (action === "open-cart") authRequired(() => {
    loadCart();
    elements.cartModal.showModal();
  });
  if (action === "close-modal") event.target.closest("dialog").close();
  if (action === "explore") document.querySelector("#catalog").scrollIntoView();
  if (action === "clear-filters") {
    state.filters = { q: "", categoria: "", estado_conservacao: "" };
    elements.searchInput.value = "";
    elements.conditionFilter.value = "";
    document.querySelectorAll(".category-chip").forEach((chip) => chip.classList.toggle("active", !chip.dataset.category));
    loadAds();
  }

  const authTab = event.target.closest("[data-auth-tab]")?.dataset.authTab;
  if (authTab) switchAuthTab(authTab);

  const category = event.target.closest("[data-category]");
  if (category) {
    document.querySelectorAll(".category-chip").forEach((chip) => chip.classList.remove("active"));
    category.classList.add("active");
    state.filters.categoria = category.dataset.category;
    loadAds();
  }

  const addButton = event.target.closest("[data-add-cart]");
  if (addButton) authRequired(() => addToCart(addButton.dataset.addCart));

  const removeButton = event.target.closest("[data-remove-cart]");
  if (removeButton) removeFromCart(removeButton.dataset.removeCart);
});

document.querySelector("#header-search").addEventListener("submit", (event) => {
  event.preventDefault();
  state.filters.q = elements.searchInput.value.trim();
  loadAds();
  document.querySelector("#catalog").scrollIntoView();
});

elements.conditionFilter.addEventListener("change", () => {
  state.filters.estado_conservacao = elements.conditionFilter.value;
  loadAds();
});

elements.photoInput.addEventListener("change", renderPhotoPreview);

document.querySelector("#login-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const formElement = event.currentTarget;
  const form = new FormData(formElement);
  const button = formElement.querySelector("button[type=submit]");
  clearFormErrors(formElement);
  button.disabled = true;
  try {
    const session = await request("/v1/sessoes", {
      method: "POST",
      body: JSON.stringify(Object.fromEntries(form)),
    });
    saveSession(session);
    elements.authModal.close();
    formElement.reset();
    toast(`Olá, ${session.usuario.nome.split(" ")[0]}.`);
  } catch (error) {
    showFormErrors(formElement, error.fields);
    toast(error.message);
  } finally {
    button.disabled = false;
  }
});

document.querySelector("#register-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const formElement = event.currentTarget;
  const form = Object.fromEntries(new FormData(formElement));
  const button = formElement.querySelector("button[type=submit]");
  clearFormErrors(formElement);
  button.disabled = true;
  const payload = {
    nome: form.nome,
    cpf: form.cpf,
    email: form.email,
    senha: form.senha,
    telefone: form.telefone,
    endereco: {
      cep: form.cep,
      logradouro: form.logradouro,
      numero: form.numero,
      complemento: "",
      bairro: form.bairro,
      cidade: form.cidade,
      estado: form.estado,
    },
  };
  try {
    await request("/v1/usuarios", { method: "POST", body: JSON.stringify(payload) });
    formElement.reset();
    switchAuthTab("login");
    document.querySelector("#login-form [name=identificador]").value = payload.email;
    toast("Conta criada. Agora entre com sua senha.");
  } catch (error) {
    showFormErrors(formElement, error.fields);
    toast(error.message);
  } finally {
    button.disabled = false;
  }
});

document.querySelector("#sell-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const formElement = event.currentTarget;
  const form = Object.fromEntries(new FormData(formElement));
  clearFormErrors(formElement);
  const files = [...elements.photoInput.files];
  const photoError = validatePhotoFiles(files);
  if (photoError) {
    showFormErrors(formElement, { fotos: photoError });
    toast(photoError);
    return;
  }
  const normalizedPrice = form.preco.replace(/\./g, "").replace(",", ".");
  const payload = {
    titulo: form.titulo,
    descricao: form.descricao,
    categoria: form.categoria,
    tamanho: form.tamanho,
    cor: form.cor,
    estado_conservacao: form.estado_conservacao,
    preco_centavos: Math.round(Number(normalizedPrice) * 100),
    urls_fotos: [],
  };
  if (!Number.isFinite(payload.preco_centavos) || payload.preco_centavos <= 0) {
    toast("Informe um preço válido.");
    return;
  }

  const button = formElement.querySelector("button[type=submit]");
  button.disabled = true;
  try {
    payload.urls_fotos = await uploadPhotos(files);
    await request("/v1/anuncios", { method: "POST", body: JSON.stringify(payload) });
    elements.sellModal.close();
    formElement.reset();
    elements.photoPreview.innerHTML = "";
    elements.uploadStatus.classList.add("hidden");
    await loadAds();
    document.querySelector("#catalog").scrollIntoView();
    toast("Anúncio publicado.");
  } catch (error) {
    elements.uploadStatus.classList.add("hidden");
    showFormErrors(formElement, error.fields);
    toast(error.message);
  } finally {
    button.disabled = false;
    if (!elements.sellModal.open) elements.uploadStatus.classList.add("hidden");
  }
});

updateSessionUI();
loadAds();
if (state.token) loadCart();
