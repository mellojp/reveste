export function toast(message) {
  const region = document.querySelector("#toast-region");
  const notification = document.createElement("div");
  notification.className = "toast";
  notification.textContent = message;
  region.append(notification);
  setTimeout(() => {
    notification.classList.add("is-leaving");
    notification.addEventListener("animationend", () => notification.remove(), { once: true });
  }, 3800);
}
