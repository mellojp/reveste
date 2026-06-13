import { state } from "./state.js";
import { renderHeader } from "../components/shell.js";
import { homePage } from "../pages/home.js";
import { catalogPage } from "../pages/catalog.js";
import { loginPage, registerPage } from "../pages/auth.js";
import { sellPage } from "../pages/sell.js";
import { profilePage } from "../pages/profile.js";
import { myAdsPage } from "../pages/my-ads.js";
import { cartPage } from "../pages/cart.js";
import { adDetailPage } from "../pages/ad-detail.js";
import { editAdPage } from "../pages/edit-ad.js";
import { sellerProfilePage } from "../pages/seller-profile.js";
import { notFoundPage } from "../pages/not-found.js";
import { pageSkeleton } from "./feedback.js";

const routes = {
  "/": homePage,
  "/catalogo": catalogPage,
  "/entrar": loginPage,
  "/cadastro": registerPage,
  "/vender": sellPage,
  "/perfil": profilePage,
  "/meus-anuncios": myAdsPage,
  "/carrinho": cartPage,
};

const protectedRoutes = new Set(["/vender", "/perfil", "/meus-anuncios", "/carrinho"]);
const authRoutes = new Set(["/entrar", "/cadastro"]);
const dynamicRoutes = [
  { pattern: /^\/anuncios\/([^/]+)$/, render: adDetailPage, parameter: "idAnuncio" },
  { pattern: /^\/meus-anuncios\/([^/]+)\/editar$/, render: editAdPage, parameter: "idAnuncio", protected: true },
  { pattern: /^\/vendedores\/([^/]+)$/, render: sellerProfilePage, parameter: "idVendedor" },
];

export function navigate(path, { replace = false, preserveScroll = false } = {}) {
  const currentPath = window.location.pathname.replace(/\/+$/, "") || "/";
  const nextPath = new URL(path, window.location.origin).pathname.replace(/\/+$/, "") || "/";
  const samePage = currentPath === nextPath;
  const keepScroll = preserveScroll || samePage;
  const update = () => {
    if (replace) history.replaceState({}, "", path);
    else history.pushState({}, "", path);
    return renderRoute({ preserveScroll: keepScroll, animateFallback: !samePage });
  };
  if (authRoutes.has(currentPath) && authRoutes.has(nextPath)) {
    transitionRoute(update, "auth");
    return;
  }
  if (samePage) {
    update();
    return;
  }
  transitionRoute(update);
}

export async function renderRoute({ preserveScroll = false, animateFallback = true } = {}) {
  const previousScroll = window.scrollY;
  const page = document.querySelector("#page");
  if (preserveScroll) {
    page.style.minHeight = `${page.offsetHeight}px`;
  }
  let path = window.location.pathname.replace(/\/+$/, "") || "/";
  if (protectedRoutes.has(path) && !state.token) {
    history.replaceState({}, "", `/entrar?retorno=${encodeURIComponent(path)}`);
    path = "/entrar";
  }
  const dynamicRoute = dynamicRoutes
    .map((route) => ({ ...route, match: path.match(route.pattern) }))
    .find((route) => route.match);
  if (dynamicRoute?.protected && !state.token) {
    history.replaceState({}, "", `/entrar?retorno=${encodeURIComponent(path)}`);
    return renderRoute();
  }
  const render = routes[path] || dynamicRoute?.render || notFoundPage;
  const parameters = dynamicRoute
    ? { [dynamicRoute.parameter]: decodeURIComponent(dynamicRoute.match[1]) }
    : {};
  page.dataset.route = path;
  page.innerHTML = pageSkeleton();
  renderHeader();
  document.title = `ReVeste | ${render.title || "Moda que continua"}`;
  await render(page, parameters);
  page.focus({ preventScroll: true });
  window.scrollTo({ top: preserveScroll ? previousScroll : 0, behavior: "auto" });
  if (preserveScroll) {
    requestAnimationFrame(() => {
      window.scrollTo({ top: previousScroll, behavior: "auto" });
      page.style.removeProperty("min-height");
    });
  }
  if (!document.startViewTransition && animateFallback) {
    page.classList.remove("route-enter");
    requestAnimationFrame(() => page.classList.add("route-enter"));
  }
}

function transitionRoute(update, kind = "page") {
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (!document.startViewTransition || reduceMotion) {
    update();
    return;
  }
  document.documentElement.dataset.transition = kind;
  const transition = document.startViewTransition(update);
  transition.finished.finally(() => {
    delete document.documentElement.dataset.transition;
  });
}

export function startRouter() {
  window.addEventListener("popstate", () => transitionRoute(renderRoute));
  renderRoute();
}
