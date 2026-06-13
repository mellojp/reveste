export const categories = [
  ["", "Tudo"],
  ["vestidos", "Vestidos"],
  ["camisetas", "Camisetas"],
  ["calcas", "Calças"],
  ["saias_e_shorts", "Saias e shorts"],
  ["casacos", "Casacos"],
  ["acessorios", "Acessórios"],
  ["calcados", "Calçados"],
  ["outros", "Outros"],
];

export const conditions = [
  ["", "Todos os estados"],
  ["novo", "Novo"],
  ["seminovo", "Seminovo"],
  ["usado", "Usado"],
  ["muito_usado", "Muito usado"],
  ["desgastado", "Desgastado"],
];

export function money(cents = 0) {
  return new Intl.NumberFormat("pt-BR", {
    style: "currency",
    currency: "BRL",
  }).format(cents / 100);
}

export function escapeHTML(value = "") {
  return String(value).replace(/[&<>"']/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#039;",
  })[character]);
}

export function isPublicImageURL(value) {
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
}

export function placeholderSVG() {
  return `<svg class="product-placeholder" viewBox="0 0 180 190" aria-hidden="true"><path d="M58 30 76 18h28l18 12 34 19-19 35-17-9v89H60V75l-17 9-19-35 34-19Z"/><path d="M76 18c0 18 28 18 28 0"/></svg>`;
}

export function adImage(ad) {
  const url = ad.fotos?.[0]?.url;
  return isPublicImageURL(url)
    ? `<img src="${escapeHTML(url)}" alt="" loading="lazy" onerror="this.remove()">`
    : placeholderSVG();
}
