import { clearStoredSession, state } from "./state.js";

export class APIError extends Error {
  constructor(message, fields = {}, status = 0, code = "") {
    super(message);
    this.name = "APIError";
    this.fields = fields;
    this.status = status;
    this.code = code;
  }
}

export async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (options.body) headers["Content-Type"] = "application/json";
  if (state.token) headers.Authorization = `Bearer ${state.token}`;

  const response = await fetch(path, { ...options, headers });
  const payload = response.status === 204 ? null : await response.json().catch(() => null);
  if (!response.ok) {
    if (response.status === 401 && state.token) clearStoredSession();
    throw new APIError(
      payload?.mensagem || "Não foi possível concluir a operação.",
      payload?.campos || {},
      response.status,
      payload?.codigo || "",
    );
  }
  return payload;
}
