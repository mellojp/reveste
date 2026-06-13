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
import { notFoundPage } from "../pages/not-found.js";

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
];

export function navigate(path, { replace = false } = {}) {
  const currentPath = window.location.pathname.replace(/\/+$/, "") || "/";
  const nextPath = new URL(path, window.location.origin).pathname.replace(/\/+$/, "") || "/";
  const update = () => {
    if (replace) history.replaceState({}, "", path);
    else history.pushState({}, "", path);
    return renderRoute();
  };
  if (authRoutes.has(currentPath) && authRoutes.has(nextPath)) {
    update();
    return;
  }
  transitionRoute(update);
}

export async function renderRoute() {
  let path = window.location.pathname.replace(/\/+$/, "") || "/";
  if (protectedRoutes.has(path) && !state.token) {
    history.replaceState({}, "", `/entrar?retorno=${encodeURIComponent(path)}`);
    path = "/entrar";
  }
  const dynamicRoute = dynamicRoutes
    .map((route) => ({ ...route, match: path.match(route.pattern) }))
    .find((route) => route.match);
  const render = routes[path] || dynamicRoute?.render || notFoundPage;
  const parameters = dynamicRoute
    ? { [dynamicRoute.parameter]: decodeURIComponent(dynamicRoute.match[1]) }
    : {};
  const page = document.querySelector("#page");
  page.dataset.route = path;
  page.innerHTML = `<div class="page-loading">Carregando...</div>`;
  renderHeader();
  document.title = `ReVeste | ${render.title || "Moda que continua"}`;
  await render(page, parameters);
  page.focus({ preventScroll: true });
  window.scrollTo({ top: 0, behavior: "auto" });
  if (!document.startViewTransition) {
    page.classList.remove("route-enter");
    requestAnimationFrame(() => page.classList.add("route-enter"));
  }
}

function transitionRoute(update) {
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (!document.startViewTransition || reduceMotion) {
    update();
    return;
  }
  document.startViewTransition(update);
}

export function startRouter() {
  window.addEventListener("popstate", () => transitionRoute(renderRoute));
  renderRoute();
}
