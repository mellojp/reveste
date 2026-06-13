import { request } from "./api.js";
import { toast } from "./notifications.js";
import { state } from "./state.js";
import { renderHeader } from "../components/shell.js";

export async function addToCart(adID) {
  state.cart = await request("/v1/carrinho/itens", {
    method: "POST",
    body: JSON.stringify({ id_anuncio: adID }),
  });
  renderHeader();
  toast("Peça adicionada à sacola.");
}

export async function removeFromCart(adID) {
  state.cart = await request(`/v1/carrinho/itens/${encodeURIComponent(adID)}`, {
    method: "DELETE",
  });
  renderHeader();
  return state.cart;
}
