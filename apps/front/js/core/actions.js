import { navigate } from "./router.js";
import { logout } from "./session.js";
import { toast } from "./notifications.js";

export function registerGlobalActions() {
  document.addEventListener("click", async (event) => {
    const link = event.target.closest("[data-link]");
    if (link) {
      event.preventDefault();
      navigate(link.getAttribute("href"));
      return;
    }
    const action = event.target.closest("[data-action]")?.dataset.action;
    if (action === "logout") {
      await logout();
      toast("Sessão encerrada.");
      navigate("/");
    }
  });
}
