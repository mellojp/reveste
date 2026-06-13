import { renderShell } from "./js/components/shell.js";
import { loadCart, refreshSession, state } from "./js/core/session.js";
import { startRouter } from "./js/core/router.js";
import { registerGlobalActions } from "./js/core/actions.js";

renderShell();
registerGlobalActions();
await refreshSession();
if (state.token) await loadCart();
startRouter();
