export const state = {
  token: sessionStorage.getItem("reveste_token") || "",
  user: JSON.parse(sessionStorage.getItem("reveste_user") || "null"),
  cart: { anuncios: [], total_centavos: 0 },
  catalog: [],
};

export function persistSession(session) {
  state.token = session.token;
  state.user = session.usuario;
  sessionStorage.setItem("reveste_token", state.token);
  sessionStorage.setItem("reveste_user", JSON.stringify(state.user));
}

export function clearStoredSession() {
  state.token = "";
  state.user = null;
  state.cart = { anuncios: [], total_centavos: 0 };
  sessionStorage.removeItem("reveste_token");
  sessionStorage.removeItem("reveste_user");
}
