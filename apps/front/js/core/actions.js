import { navigate } from "./router.js";
import { logout } from "./session.js";
import { toast } from "./notifications.js";
import { setButtonLoading } from "./feedback.js";

export function registerGlobalActions() {
  document.addEventListener("click", async (event) => {
    const link = event.target.closest("[data-link]");
    if (link) {
      event.preventDefault();
      link.classList.add("is-navigating");
      link.setAttribute("aria-busy", "true");
      navigate(link.getAttribute("href"));
      return;
    }
    const trigger = event.target.closest("[data-action]");
    const action = trigger?.dataset.action;
    if (action === "logout") {
      setButtonLoading(trigger, true, "Saindo...");
      try {
        await logout();
        toast("Sessão encerrada.");
        navigate("/");
      } catch (error) {
        setButtonLoading(trigger, false);
        toast(error.message || "Não foi possível sair.", "error");
      }
    }
  });
}
