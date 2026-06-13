export function setButtonLoading(button, loading, label = "Carregando...") {
  if (!button) return;
  if (loading) {
    if (!button.dataset.originalContent) {
      button.dataset.originalContent = button.innerHTML;
    }
    button.disabled = true;
    button.classList.add("is-loading");
    button.setAttribute("aria-busy", "true");
    button.innerHTML = `<span class="button-spinner" aria-hidden="true"></span><span>${label}</span>`;
    return;
  }
  button.disabled = false;
  button.classList.remove("is-loading");
  button.removeAttribute("aria-busy");
  if (button.dataset.originalContent) {
    button.innerHTML = button.dataset.originalContent;
    delete button.dataset.originalContent;
  }
}

export function pageSkeleton(label = "Carregando conteúdo") {
  return `
    <div class="page-skeleton" role="status" aria-label="${label}">
      <span class="skeleton-line skeleton-eyebrow"></span>
      <span class="skeleton-line skeleton-title"></span>
      <span class="skeleton-line skeleton-copy"></span>
      <div class="skeleton-panel"></div>
    </div>`;
}

export function gridSkeleton(count = 8) {
  return Array.from({ length: count }, () => `
    <div class="product-skeleton" aria-hidden="true">
      <span class="skeleton-image"></span>
      <span class="skeleton-line"></span>
      <span class="skeleton-line short"></span>
    </div>`).join("");
}

export function revealContent(element) {
  if (!element) return;
  element.classList.remove("content-reveal");
  requestAnimationFrame(() => element.classList.add("content-reveal"));
}
