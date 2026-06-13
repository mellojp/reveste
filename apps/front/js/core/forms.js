export function clearFormErrors(form) {
  form.querySelectorAll(".field-error").forEach((element) => element.remove());
  form.querySelectorAll("[aria-invalid=true]").forEach((field) => {
    field.removeAttribute("aria-invalid");
    field.removeAttribute("aria-describedby");
  });
}

export function showFormErrors(form, fields = {}) {
  clearFormErrors(form);
  let firstInvalidField = null;
  const aliases = {
    preco_centavos: "preco",
    urls_fotos: "fotos",
  };
  Object.entries(fields).forEach(([path, message], index) => {
    const apiName = path.split(".").at(-1);
    const name = aliases[apiName] || apiName;
    const field = form.elements.namedItem(name);
    if (!(field instanceof HTMLElement)) return;
    const error = document.createElement("small");
    error.id = `${form.id}-${name}-error-${index}`;
    error.className = "field-error";
    error.textContent = message;
    field.setAttribute("aria-invalid", "true");
    field.setAttribute("aria-describedby", error.id);
    field.closest("label")?.append(error);
    firstInvalidField ||= field;
  });
  firstInvalidField?.focus();
}
