import { request } from "./api.js";
import { clearStoredSession, persistSession, state } from "./state.js";

export { state };

export async function refreshSession() {
  if (!state.token) return;
  try {
    state.user = await request("/v1/me");
    sessionStorage.setItem("reveste_user", JSON.stringify(state.user));
  } catch {
    clearStoredSession();
  }
}

export function saveSession(session) {
  persistSession(session);
}

export async function logout() {
  try {
    await request("/v1/sessoes/atual", { method: "DELETE" });
  } catch {
    // A sessão local deve ser encerrada mesmo se já expirou no servidor.
  } finally {
    clearStoredSession();
  }
}

export async function loadCart() {
  if (!state.token) return state.cart;
  state.cart = await request("/v1/carrinho");
  return state.cart;
}
