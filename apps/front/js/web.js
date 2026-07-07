import { uploadPhoto } from "./uploads.js";

const allowedTypes = new Set(["image/jpeg", "image/png", "image/webp"]);
const editors = new WeakMap();
const registrationForms = new WeakSet();
const brazilianStates = new Set([
  "AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG",
  "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
]);
let authSwitchInProgress = false;

document.addEventListener("click", (event) => {
  const authSwitch = event.target.closest("[data-auth-switch]");
  if (!authSwitch || !document.querySelector(".auth-page")) return;
  event.preventDefault();
  event.stopPropagation();
  event.stopImmediatePropagation();
  switchAuthForm(authSwitch.href, authSwitch.dataset.authSwitch);
}, true);

document.addEventListener("click", (event) => {
  const menuToggle = event.target.closest("[data-menu-toggle]");
  if (menuToggle) {
    toggleMainMenu(menuToggle);
    return;
  }

  const openMenu = document.querySelector(".main-nav.is-open");
  if (openMenu) {
    const clickedInsideMenu = event.target.closest(".main-nav");
    const clickedHeaderControl = event.target.closest(".site-header");
    if (event.target.closest(".main-nav a") || (!clickedInsideMenu && !clickedHeaderControl)) {
      closeMainMenu();
    }
  }

  const filter = event.target.closest("#filter-toggle");
  if (filter) {
    const panel = document.querySelector("#filter-panel");
    const open = panel?.classList.toggle("is-open") || false;
    filter.setAttribute("aria-expanded", String(open));
    return;
  }

  const thumbnail = event.target.closest("[data-gallery-photo]");
  if (thumbnail) {
    const url = thumbnail.dataset.galleryPhoto;
    if (!isPublicImageURL(url)) return;
    const image = document.createElement("img");
    image.src = url;
    image.alt = thumbnail.dataset.galleryAlt || "";
    image.referrerPolicy = "no-referrer";
    document.querySelector("#ad-main-photo")?.replaceChildren(image);
    document.querySelectorAll("[data-gallery-photo]").forEach((item) => {
      item.classList.toggle("active", item === thumbnail);
    });
    return;
  }

  const deletion = event.target.closest("[data-confirm-delete]");
  if (deletion && deletion.dataset.confirming !== "true") {
    event.preventDefault();
    deletion.dataset.originalLabel ||= deletion.textContent;
    deletion.dataset.confirming = "true";
    deletion.textContent = deletion.dataset.confirmLabel || "Confirmar exclusão";
    window.setTimeout(() => {
      if (deletion.isConnected) {
        deletion.dataset.confirming = "false";
        deletion.textContent = deletion.dataset.originalLabel;
      }
    }, 5000);
  }
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") closeMainMenu();

  const chatInput = event.target.closest("[data-chat-compose] textarea");
  if (chatInput && event.key === "Enter" && !event.shiftKey && !event.isComposing) {
    event.preventDefault();
    if (chatInput.value.trim()) chatInput.form.requestSubmit();
  }
});

document.addEventListener("submit", async (event) => {
  const registrationForm = event.target.closest("[data-register-form]");
  if (registrationForm) {
    validateRegistrationForm(registrationForm);
    if (!registrationForm.checkValidity()) {
      event.preventDefault();
      focusFirstInvalidField(registrationForm);
      return;
    }
    registrationForm.classList.add("is-submitting");
    registrationForm.querySelector("button[type=submit]")?.setAttribute("aria-busy", "true");
    return;
  }

  const form = event.target.closest("[data-ad-form]");
  if (!form || form.dataset.uploadReady === "true") return;
  event.preventDefault();
  event.stopImmediatePropagation();

  const editor = initializeEditor(form);
  const error = validatePhotos(editor.photos);
  if (error) {
    showUploadStatus(form, error, true);
    return;
  }

  const button = form.querySelector("button[type=submit]");
  button.disabled = true;
  button.classList.add("is-loading");
  try {
    const urls = [];
    for (const [index, photo] of editor.photos.entries()) {
      showUploadStatus(form, `Preparando foto ${index + 1} de ${editor.photos.length}...`);
      urls.push(photo.url || await uploadPhoto(photo.file));
    }
    form.querySelectorAll('input[name="urls_fotos"]').forEach((input) => input.remove());
    for (const url of urls) {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = "urls_fotos";
      input.value = url;
      form.append(input);
    }
    form.dataset.uploadReady = "true";
    // Quando todas as fotos já têm URL (ex.: edição reaproveitando as existentes), o laço
    // acima não aguarda nada e todo o handler corre de forma síncrona dentro do despacho do
    // submit atual. Chamar requestSubmit() aqui seria reentrante e o navegador o ignora em
    // silêncio (botão trava, nenhuma requisição). Cede um tick para submeter já fora do
    // despacho original — no fluxo de criação o await do upload já garantia isso.
    await new Promise((resolve) => setTimeout(resolve, 0));
    form.requestSubmit();
  } catch (error) {
    showUploadStatus(form, error.message || "Não foi possível enviar as fotos.", true);
    button.disabled = false;
    button.classList.remove("is-loading");
  }
}, true);

document.addEventListener("change", (event) => {
  const input = event.target.closest("[data-photo-input]");
  if (!input) return;
  const form = input.closest("[data-ad-form]");
  const editor = initializeEditor(form);
  for (const file of input.files) {
    if (editor.photos.length >= 5) break;
    editor.photos.push({ file, preview: URL.createObjectURL(file) });
  }
  input.value = "";
  renderPhotos(editor);
});

document.addEventListener("click", async (event) => {
  const copiar = event.target.closest("[data-copy]");
  if (copiar) {
    try {
      await navigator.clipboard.writeText(copiar.dataset.copy);
      const rotulo = copiar.dataset.copyLabel || copiar.textContent;
      copiar.textContent = "Copiado!";
      copiar.disabled = true;
      setTimeout(() => { copiar.textContent = rotulo; copiar.disabled = false; }, 1800);
    } catch {
      /* clipboard indisponível: o usuário ainda pode selecionar e copiar manualmente */
    }
    return;
  }
});

document.addEventListener("click", (event) => {
  const control = event.target.closest("[data-photo-action]");
  if (!control) return;
  const form = control.closest("[data-ad-form]");
  const editor = initializeEditor(form);
  const item = control.closest(".photo-preview-item");
  const index = [...editor.preview.children].indexOf(item);
  if (index < 0) return;
  const action = control.dataset.photoAction;
  if (action === "remove") {
    const [removed] = editor.photos.splice(index, 1);
    if (removed?.file) URL.revokeObjectURL(removed.preview);
  } else if (action === "left" && index > 0) {
    [editor.photos[index - 1], editor.photos[index]] = [editor.photos[index], editor.photos[index - 1]];
  } else if (action === "right" && index < editor.photos.length - 1) {
    [editor.photos[index + 1], editor.photos[index]] = [editor.photos[index], editor.photos[index + 1]];
  }
  renderPhotos(editor);
});

document.addEventListener("input", (event) => {
  const input = event.target.closest("[data-register-form] input");
  if (!input) return;
  applyRegistrationMask(input);
  clearClientError(input);
  if (input.matches("[data-password]")) {
    updatePasswordStrength(input);
    const confirmation = input.form.querySelector("[data-password-confirmation]");
    if (confirmation?.value) validateRegistrationInput(confirmation);
  }
  updateRegistrationProgress(input.form, input.closest("[data-registration-section]"));
});

document.addEventListener("focusout", (event) => {
  const input = event.target.closest("[data-register-form] input");
  if (!input) return;
  validateRegistrationInput(input);
  updateRegistrationProgress(input.form, input.closest("[data-registration-section]"));
});

// Autofill de endereço a partir do CEP. Funciona em qualquer formulário com um campo
// name="cep" (cadastro, perfil e endereços), consultando o backend (/v1/cep) para manter
// a CSP restrita e reaproveitar a validação do domínio.
const cepConsultado = new WeakMap();

document.addEventListener("input", (event) => {
  const cepInput = event.target.closest('input[name="cep"]');
  if (cepInput) autofillEnderecoPorCEP(cepInput);
});

async function autofillEnderecoPorCEP(input) {
  const form = input.form;
  if (!form) return;
  const digits = input.value.replace(/\D/g, "");
  if (digits.length !== 8) return;
  if (cepConsultado.get(input) === digits) return;
  cepConsultado.set(input, digits);

  input.setAttribute("aria-busy", "true");
  try {
    const response = await fetch(`/v1/cep/${digits}`, { headers: { Accept: "application/json" } });
    if (!response.ok) {
      // CEP inexistente ou provedor fora do ar: o usuário preenche manualmente.
      cepConsultado.delete(input);
      return;
    }
    const endereco = await response.json();
    preencherCampoEndereco(form, "logradouro", endereco.logradouro);
    preencherCampoEndereco(form, "bairro", endereco.bairro);
    preencherCampoEndereco(form, "cidade", endereco.cidade);
    preencherCampoEndereco(form, "estado", endereco.estado);
    const numero = form.elements.numero;
    if (numero && !numero.value.trim()) numero.focus({ preventScroll: true });
  } catch {
    cepConsultado.delete(input);
  } finally {
    input.removeAttribute("aria-busy");
  }
}

function preencherCampoEndereco(form, nome, valor) {
  if (!valor) return;
  const campo = form.elements[nome];
  if (!campo) return;
  campo.value = valor;
  clearClientError(campo);
}

document.addEventListener("focusin", (event) => {
  const section = event.target.closest("[data-register-form] [data-registration-section]");
  if (section) updateRegistrationProgress(section.closest("form"), section);
});

document.addEventListener("click", (event) => {
  const toggle = event.target.closest("[data-password-toggle]");
  if (!toggle) return;
  const input = document.getElementById(toggle.getAttribute("aria-controls"));
  if (!input) return;
  const showing = input.type === "text";
  input.type = showing ? "password" : "text";
  toggle.textContent = showing ? "Mostrar" : "Ocultar";
  toggle.setAttribute("aria-label", showing ? "Mostrar senha" : "Ocultar senha");
  input.focus({ preventScroll: true });
});

document.addEventListener("htmx:load", (event) => {
  initializeEditors(event.detail.elt);
  initializeRegistrationForms(event.detail.elt);
});
document.addEventListener("htmx:responseError", () => {
  document.querySelectorAll("[data-register-form].is-submitting").forEach(resetRegistrationSubmission);
});
// O aviso e exibido apos a pagina assentar, ja fora da transicao de swap, para nao ser
// capturado pelo snapshot da view transition (o que fazia o toast piscar atras do conteudo).
document.addEventListener("htmx:afterSettle", exibirAvisoDaURL);
// Ao voltar/avancar, o htmx restaura um snapshot que pode conter um toast antigo. Removemos
// para o aviso nao ficar preso na tela; um aviso novo so aparece via parametro de URL.
document.addEventListener("htmx:historyRestore", () => {
  document.querySelectorAll(".toast").forEach((toast) => toast.remove());
});
document.addEventListener("DOMContentLoaded", () => {
  initializeEditors(document);
  initializeRegistrationForms(document);
  exibirAvisoDaURL();
  rolarChatSeNecessario();
});

// Reseta o scroll ao topo em navegacoes de pagina inteira. Cobre tanto links boosted
// quanto redirecionamentos via HX-Location (publicar anuncio, sair), que o htmx nao
// rola sozinho. Voltar/avancar (popstate) preserva a posicao restaurada pelo htmx.
let restaurandoHistorico = false;
window.addEventListener("popstate", () => {
  restaurandoHistorico = true;
  window.setTimeout(() => { restaurandoHistorico = false; }, 600);
});
document.addEventListener("htmx:afterSwap", (event) => {
  if (event.detail?.target?.tagName !== "BODY" || restaurandoHistorico) return;
  window.scrollTo({ top: 0, left: 0, behavior: "auto" });
});

// Barra de progresso de navegacao. Toda requisicao HTMX (navegacao boosted e fragmentos)
// acende uma barra no topo, dando feedback imediato enquanto o backend responde. A barra e
// construida via DOM API (sem innerHTML) e ancorada no <html>, que nao e trocado no swap do
// body, de modo que persiste entre as navegacoes. Um contador trata requisicoes simultaneas.
const barraNavegacao = (() => {
  let barra = null;
  let pendentes = 0;
  let atrasoInicio = null;
  let limpeza = null;

  function elemento() {
    if (barra && barra.isConnected) return barra;
    barra = document.createElement("div");
    barra.className = "nav-progress";
    barra.setAttribute("aria-hidden", "true");
    document.documentElement.appendChild(barra);
    return barra;
  }
  function iniciar() {
    pendentes += 1;
    if (pendentes > 1) return;
    clearTimeout(atrasoInicio);
    clearTimeout(limpeza);
    // Atraso curto evita o "flash" da barra quando a resposta volta quase instantaneamente.
    atrasoInicio = window.setTimeout(() => {
      const el = elemento();
      el.classList.remove("is-active", "is-done");
      void el.offsetWidth; // reinicia a animacao CSS do zero a cada nova navegacao
      el.classList.add("is-active");
      document.documentElement.classList.add("navegando");
    }, 90);
  }
  function terminar() {
    pendentes = Math.max(0, pendentes - 1);
    if (pendentes > 0) return;
    clearTimeout(atrasoInicio);
    document.documentElement.classList.remove("navegando");
    const el = elemento();
    if (!el.classList.contains("is-active")) return; // resposta instantanea: barra nem apareceu
    el.classList.add("is-done"); // completa ate 100% e some; sobrepoe a animacao da fase ativa
    limpeza = window.setTimeout(() => el.classList.remove("is-active", "is-done"), 450);
  }
  return { iniciar, terminar };
})();
// Ignora requisicoes de polling (ex.: o chat, com hx-trigger "every 5s") para a barra nao
// piscar em segundo plano; ela so reflete acoes de navegacao/envio iniciadas pelo usuario.
function ehPolling(elt) {
  return (elt?.getAttribute?.("hx-trigger") || "").includes("every");
}
document.addEventListener("htmx:beforeRequest", (event) => {
  if (!ehPolling(event.detail?.elt)) barraNavegacao.iniciar();
});
document.addEventListener("htmx:afterRequest", (event) => {
  if (!ehPolling(event.detail?.elt)) barraNavegacao.terminar();
});

// Chat: mantem a conversa rolada ate a ultima mensagem. So rola automaticamente quando o
// usuario ja estava no fim, para o polling nao interromper a leitura do historico.
let chatPresoNoFim = true;
function chatNoFim(elemento) {
  return elemento.scrollHeight - elemento.scrollTop - elemento.clientHeight < 80;
}
function rolarChatSeNecessario() {
  const thread = document.getElementById("chat-thread");
  if (thread && chatPresoNoFim) thread.scrollTop = thread.scrollHeight;
}
document.addEventListener("htmx:beforeSwap", (event) => {
  const alvo = event.detail?.target;
  if (alvo?.id === "chat-thread") chatPresoNoFim = chatNoFim(alvo);
  else if (alvo?.tagName === "BODY") chatPresoNoFim = true;
});
document.addEventListener("htmx:afterSettle", rolarChatSeNecessario);
document.addEventListener("htmx:afterRequest", (event) => {
  const form = event.detail?.elt?.closest?.("[data-chat-compose]");
  if (!form || !event.detail.successful) return;
  form.reset();
  form.querySelector("textarea")?.focus({ preventScroll: true });
});

window.addEventListener("popstate", () => {
  if (!document.querySelector(".auth-page")) return;
  if (!["/entrar", "/cadastro"].includes(window.location.pathname)) return;
  const direction = window.location.pathname === "/cadastro" ? "cadastro" : "entrar";
  switchAuthForm(window.location.href, direction, { push: false });
});

function toggleMainMenu(button) {
  const nav = document.getElementById(button.getAttribute("aria-controls"));
  if (!nav) return;
  const open = !nav.classList.contains("is-open");
  nav.classList.toggle("is-open", open);
  button.classList.toggle("is-open", open);
  button.setAttribute("aria-expanded", String(open));
  button.setAttribute("aria-label", open ? "Fechar navegação" : "Abrir navegação");
}

function closeMainMenu() {
  const button = document.querySelector("[data-menu-toggle]");
  const nav = document.getElementById(button?.getAttribute("aria-controls"));
  if (!button || !nav) return;
  nav.classList.remove("is-open");
  button.classList.remove("is-open");
  button.setAttribute("aria-expanded", "false");
  button.setAttribute("aria-label", "Abrir navegação");
}

async function switchAuthForm(url, direction, options = {}) {
  if (authSwitchInProgress) return;
  const content = document.querySelector(".auth-content");
  const current = content?.querySelector(".auth-form-panel");
  if (!content || !current) {
    window.location.href = url;
    return;
  }

  authSwitchInProgress = true;
  const toCadastro = direction === "cadastro";
  const leavingClass = toCadastro ? "is-leaving-left" : "is-leaving-right";
  const enteringClass = toCadastro ? "is-entering-right" : "is-entering-left";

  try {
    const response = await fetch(url, {
      headers: { "Accept": "text/html" },
      credentials: "same-origin",
    });
    if (!response.ok) throw new Error("Falha ao carregar formulário.");
    const html = await response.text();
    const nextDocument = new DOMParser().parseFromString(html, "text/html");
    const nextPanel = nextDocument.querySelector(".auth-form-panel");
    if (!nextPanel) throw new Error("Formulário não encontrado.");

    const scrollTask = scrollAuthPageToTop({ animate: true });
    const currentHeight = content.getBoundingClientRect().height;
    content.style.height = `${currentHeight}px`;
    content.classList.add("is-resizing");
    current.classList.add(leavingClass);
    await Promise.all([scrollTask, waitForAnimation(current)]);

    nextPanel.classList.add(enteringClass);
    content.replaceChildren(nextPanel);
    window.htmx?.process(nextPanel);
    initializeEditors(nextPanel);
    initializeRegistrationForms(nextPanel);
    document.title = nextDocument.title || document.title;
    if (options.push !== false) history.pushState({ auth: true }, "", url);

    const nextHeight = measureAuthContentHeight(content, nextPanel);
    requestAnimationFrame(() => {
      content.style.height = `${nextHeight}px`;
    });

    await waitForAnimation(nextPanel);
    nextPanel.classList.remove(enteringClass);
    await waitForTransition(content);
  } catch {
    window.location.href = url;
  } finally {
    current.classList.remove(leavingClass);
    content.classList.remove("is-resizing");
    content.style.height = "";
    authSwitchInProgress = false;
  }
}

function waitForAnimation(element) {
  return new Promise((resolve) => {
    if (prefersReducedMotion() || window.getComputedStyle(element).animationName === "none") {
      requestAnimationFrame(resolve);
      return;
    }
    let done = false;
    const finish = () => {
      if (done) return;
      done = true;
      element.removeEventListener("animationend", finish);
      resolve();
    };
    element.addEventListener("animationend", finish, { once: true });
    window.setTimeout(finish, 450);
  });
}

function scrollAuthPageToTop(options = {}) {
  const page = document.querySelector(".auth-page");
  const header = document.querySelector(".site-header");
  if (!page) return Promise.resolve();
  const headerHeight = header?.getBoundingClientRect().height || 0;
  const top = Math.max(0, page.getBoundingClientRect().top + window.scrollY - headerHeight);
  if (options.animate && !prefersReducedMotion()) {
    return animateScrollTo(top, 380);
  }
  window.scrollTo({ top, behavior: "auto" });
  return Promise.resolve();
}

function animateScrollTo(targetTop, duration) {
  return new Promise((resolve) => {
    const html = document.documentElement;
    const originalStyle = html.style.scrollBehavior;
    html.style.scrollBehavior = "auto";

    const startTop = window.scrollY;
    const distance = targetTop - startTop;
    if (Math.abs(distance) < 2) {
      html.style.scrollBehavior = originalStyle;
      resolve();
      return;
    }
    const startedAt = performance.now();
    // Aproximação de cubic-bezier(0.3, 0, 0.2, 1) para suavidade extra
    const ease = (p) => p < 0.5
      ? 4 * p * p * p
      : 1 - Math.pow(-2 * p + 2, 3) / 2;

    const step = (now) => {
      const progress = Math.min(1, (now - startedAt) / duration);
      window.scrollTo(0, startTop + distance * ease(progress));
      if (progress < 1) {
        requestAnimationFrame(step);
      } else {
        html.style.scrollBehavior = originalStyle;
        resolve();
      }
    };

    requestAnimationFrame(step);
  });
}

function measureAuthContentHeight(content, panel) {
  const styles = window.getComputedStyle(content);
  const padding = parseFloat(styles.paddingTop) + parseFloat(styles.paddingBottom);
  const headerHeight = document.querySelector(".site-header")?.getBoundingClientRect().height || 0;
  const viewportHeight = Math.max(0, window.innerHeight - headerHeight);
  return Math.max(panel.scrollHeight + padding, viewportHeight);
}

function waitForTransition(element) {
  return new Promise((resolve) => {
    const transitionDuration = window.getComputedStyle(element).transitionDuration;
    if (prefersReducedMotion() || transitionDuration === "0s" || transitionDuration === "0ms") {
      requestAnimationFrame(resolve);
      return;
    }
    let done = false;
    const finish = (event) => {
      if (event && event.target !== element) return;
      if (done) return;
      done = true;
      element.removeEventListener("transitionend", finish);
      resolve();
    };
    element.addEventListener("transitionend", finish);
    window.setTimeout(finish, 500);
  });
}

function prefersReducedMotion() {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

function initializeRegistrationForms(root) {
  if (root.matches?.("[data-register-form]")) initializeRegistrationForm(root);
  root.querySelectorAll?.("[data-register-form]").forEach(initializeRegistrationForm);
}

function initializeEditors(root) {
  if (root.matches?.("[data-ad-form]")) initializeEditor(root);
  root.querySelectorAll?.("[data-ad-form]").forEach(initializeEditor);
}

function initializeRegistrationForm(form) {
  if (registrationForms.has(form)) return;
  registrationForms.add(form);
  form.noValidate = true;
  form.querySelectorAll("input").forEach((input) => {
    applyRegistrationMask(input);
    if (input.matches("[data-password]")) updatePasswordStrength(input);
  });
  const invalid = form.querySelector('[aria-invalid="true"]');
  const activeSection = invalid?.closest("[data-registration-section]") ||
    form.querySelector("[data-registration-section]");
  updateRegistrationProgress(form, activeSection);
}

function applyRegistrationMask(input) {
  if (input.dataset.stateCode !== undefined) {
    input.value = input.value.replace(/[^a-z]/gi, "").slice(0, 2).toUpperCase();
    return;
  }
  const digits = input.value.replace(/\D/g, "");
  if (input.dataset.mask === "cpf") {
    input.value = digits.slice(0, 11)
      .replace(/^(\d{3})(\d)/, "$1.$2")
      .replace(/^(\d{3})\.(\d{3})(\d)/, "$1.$2.$3")
      .replace(/\.(\d{3})(\d)/, ".$1-$2");
  } else if (input.dataset.mask === "cep") {
    input.value = digits.slice(0, 8).replace(/^(\d{5})(\d)/, "$1-$2");
  } else if (input.dataset.mask === "phone") {
    const value = digits.slice(0, 11);
    input.value = value.length > 10
      ? value.replace(/^(\d{2})(\d{5})(\d{0,4})/, "($1) $2-$3")
      : value.replace(/^(\d{2})(\d{4})(\d{0,4})/, "($1) $2-$3");
  }
}

function validateRegistrationForm(form) {
  form.querySelectorAll("input").forEach(validateRegistrationInput);
}

function validateRegistrationInput(input) {
  clearClientError(input);
  let message = "";
  const value = input.value.trim();
  const digits = value.replace(/\D/g, "");
  if (input.validity.valueMissing) {
    message = "Preencha este campo.";
  } else if (input.validity.typeMismatch) {
    message = "Informe um e-mail válido.";
  } else if (input.validity.tooShort) {
    message = `Use pelo menos ${input.minLength} caracteres.`;
  } else if (input.name === "cpf" && value && !isValidCPF(digits)) {
    message = "Informe um CPF válido.";
  } else if (input.name === "telefone" && value && ![10, 11].includes(digits.length)) {
    message = "Informe um telefone com DDD.";
  } else if (input.name === "cep" && value && digits.length !== 8) {
    message = "O CEP deve conter 8 dígitos.";
  } else if (input.name === "estado" && value && !brazilianStates.has(value.toUpperCase())) {
    message = "Informe uma sigla de estado válida.";
  } else if (input.matches("[data-password]") && value && value.length < 8) {
    message = "A senha deve conter pelo menos 8 caracteres.";
  } else if (input.matches("[data-password-confirmation]") && value !== input.form.elements.senha.value) {
    message = "As senhas informadas não coincidem.";
  }
  setClientError(input, message);
  return message === "";
}

function setClientError(input, message) {
  input.setCustomValidity(message);
  if (!message) return;
  input.setAttribute("aria-invalid", "true");
  const error = document.createElement("small");
  error.className = "field-error";
  error.dataset.clientError = "true";
  error.id = `erro-cliente-${input.name}`;
  error.setAttribute("role", "alert");
  error.textContent = message;
  input.closest(".field")?.append(error);
  appendDescription(input, error.id);
}

function clearClientError(input) {
  input.setCustomValidity("");
  input.closest(".field")?.querySelectorAll(".field-error").forEach((error) => error.remove());
  input.removeAttribute("aria-invalid");
  input.setAttribute(
    "aria-describedby",
    (input.getAttribute("aria-describedby") || "")
      .split(/\s+/)
      .filter((id) => id && !id.startsWith("erro-"))
      .join(" "),
  );
  if (!input.getAttribute("aria-describedby")) input.removeAttribute("aria-describedby");
}

function appendDescription(input, id) {
  const ids = new Set((input.getAttribute("aria-describedby") || "").split(/\s+/).filter(Boolean));
  ids.add(id);
  input.setAttribute("aria-describedby", [...ids].join(" "));
}

function updatePasswordStrength(input) {
  const meter = input.form.querySelector("[data-password-strength]");
  if (!meter) return;
  const value = input.value;
  let level = 0;
  if (value.length >= 8) level++;
  if (value.length >= 12) level++;
  if (/[a-z]/i.test(value) && /\d/.test(value)) level++;
  if (/[^a-z0-9]/i.test(value)) level++;
  level = Math.min(level, 4);
  meter.dataset.level = String(level);
  const labels = [
    "Use pelo menos 8 caracteres.",
    "Senha básica.",
    "Senha razoável.",
    "Senha forte.",
    "Senha muito forte.",
  ];
  meter.querySelector("small").textContent = labels[level];
}

function updateRegistrationProgress(form, activeSection) {
  if (!form) return;
  const sections = [...form.querySelectorAll("[data-registration-section]")];
  const steps = document.querySelectorAll("[data-registration-step]");
  const activeName = activeSection?.dataset.registrationSection || "identity";
  steps.forEach((step) => {
    const section = sections.find((item) => item.dataset.registrationSection === step.dataset.registrationStep);
    step.classList.toggle("is-current", step.dataset.registrationStep === activeName);
    step.classList.toggle("is-complete", section ? section.querySelectorAll(":invalid").length === 0 : false);
  });
}

function focusFirstInvalidField(form) {
  const invalid = form.querySelector(":invalid");
  if (!invalid) return;
  updateRegistrationProgress(form, invalid.closest("[data-registration-section]"));
  invalid.focus({ preventScroll: true });
  invalid.scrollIntoView({ behavior: "smooth", block: "center" });
}

function resetRegistrationSubmission(form) {
  form.classList.remove("is-submitting");
  form.querySelector("button[type=submit]")?.removeAttribute("aria-busy");
}

function isValidCPF(value) {
  if (!/^\d{11}$/.test(value) || /^(\d)\1+$/.test(value)) return false;
  const calculateDigit = (base, initialWeight) => {
    const sum = [...base].reduce((total, digit, index) =>
      total + Number(digit) * (initialWeight - index), 0);
    const remainder = (sum * 10) % 11;
    return remainder === 10 ? 0 : remainder;
  };
  return calculateDigit(value.slice(0, 9), 10) === Number(value[9]) &&
    calculateDigit(value.slice(0, 10), 11) === Number(value[10]);
}

function initializeEditor(form) {
  if (editors.has(form)) return editors.get(form);
  const preview = form.querySelector("[data-photo-preview]");
  const uploader = form.querySelector("[data-photo-uploader]");
  const photos = [...preview.querySelectorAll("[data-existing-photo]")].map((item) => ({
    url: item.dataset.existingPhoto,
    preview: item.dataset.existingPhoto,
  }));
  const editor = { form, preview, uploader, photos };
  editors.set(form, editor);
  updatePhotoCount(editor);
  return editor;
}

// Reflete a quantidade de fotos no container para o CSS decidir se o botao de adicionar
// aparece grande (nenhuma foto), compacto (1 a 4) ou some (5 fotos, limite atingido).
function updatePhotoCount(editor) {
  editor.uploader?.setAttribute("data-photo-count", String(editor.photos.length));
}

function renderPhotos(editor) {
  editor.preview.replaceChildren(...editor.photos.map((photo, index) => {
    const item = document.createElement("div");
    item.className = "photo-preview-item";
    const image = document.createElement("img");
    image.src = photo.preview;
    image.alt = `Foto ${index + 1}`;
    image.referrerPolicy = "no-referrer";
    const label = document.createElement("span");
    label.textContent = index === 0 ? "Capa" : String(index + 1);
    const controls = document.createElement("div");
    controls.className = "photo-preview-controls";
    controls.append(
      photoButton("left", "←", "Mover foto para a esquerda", index === 0),
      photoButton("right", "→", "Mover foto para a direita", index === editor.photos.length - 1),
      photoButton("remove", "×", "Remover foto"),
    );
    item.append(image, label, controls);
    return item;
  }));
  updatePhotoCount(editor);
}

function photoButton(action, text, label, disabled = false) {
  const button = document.createElement("button");
  button.type = "button";
  button.dataset.photoAction = action;
  button.textContent = text;
  button.setAttribute("aria-label", label);
  button.disabled = disabled;
  return button;
}

function validatePhotos(photos) {
  if (photos.length < 2 || photos.length > 5) return "Mantenha entre 2 e 5 imagens.";
  const invalid = photos.some(({ file, url }) => file
    ? !allowedTypes.has(file.type) || file.size <= 0 || file.size > 5 * 1024 * 1024
    : !isPublicImageURL(url));
  return invalid ? "Use imagens JPEG, PNG ou WebP com até 5 MB." : "";
}

function showUploadStatus(form, message, error = false) {
  const status = form.querySelector("[data-upload-status]");
  status.textContent = message;
  status.classList.remove("hidden");
  status.classList.toggle("field-error", error);
}

function isPublicImageURL(value) {
  try {
    const url = new URL(value);
    return url.protocol === "https:" && !url.username && !url.password &&
      !url.search && !url.hash &&
      url.hostname.endsWith(".public.blob.vercel-storage.com");
  } catch {
    return false;
  }
}

// exibirAvisoDaURL le o parametro ?mensagem e cria o toast no cliente, ja com a pagina
// assentada. Como o elemento nasce depois do swap, ele fica fora do snapshot da view
// transition e aparece limpo, por cima do conteudo, sem piscar. Em seguida limpa a URL para
// o aviso nao reaparecer ao recarregar ou voltar/avancar.
function exibirAvisoDaURL() {
  const mensagem = new URL(window.location.href).searchParams.get("mensagem");
  if (!mensagem) return;
  limparMensagemDaURL();
  const regiao = document.getElementById("toast-region");
  if (!regiao) return;
  const toast = document.createElement("div");
  toast.className = "toast";
  toast.textContent = mensagem;
  regiao.appendChild(toast);
  window.setTimeout(() => {
    toast.classList.add("is-leaving");
    window.setTimeout(() => toast.remove(), 200);
  }, 3200);
}

function limparMensagemDaURL() {
  const url = new URL(window.location.href);
  if (!url.searchParams.has("mensagem")) return;
  url.searchParams.delete("mensagem");
  const destino = url.pathname + url.search + url.hash;
  window.history.replaceState(window.history.state, "", destino);
}
