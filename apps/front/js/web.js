import { uploadPhoto } from "./uploads.js";

const allowedTypes = new Set(["image/jpeg", "image/png", "image/webp"]);
const editors = new WeakMap();
const registrationForms = new WeakSet();
const brazilianStates = new Set([
  "AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG",
  "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
]);

document.addEventListener("click", (event) => {
  const filter = event.target.closest("#filter-toggle");
  if (filter) {
    const panel = document.querySelector("#filter-panel");
    const open = panel?.classList.toggle("is-open") || false;
    filter.setAttribute("aria-expanded", String(open));
    return;
  }

  const thumbnail = event.target.closest("[data-gallery-photo]");
  if (thumbnail) {
    const url = thumbnail.dataset.galleryPhoto;
    if (!isPublicImageURL(url)) return;
    const image = document.createElement("img");
    image.src = url;
    image.alt = thumbnail.dataset.galleryAlt || "";
    image.referrerPolicy = "no-referrer";
    document.querySelector("#ad-main-photo")?.replaceChildren(image);
    document.querySelectorAll("[data-gallery-photo]").forEach((item) => {
      item.classList.toggle("active", item === thumbnail);
    });
    return;
  }

  const deletion = event.target.closest("[data-confirm-delete]");
  if (deletion && deletion.dataset.confirming !== "true") {
    event.preventDefault();
    deletion.dataset.confirming = "true";
    deletion.textContent = "Confirmar exclusão";
    window.setTimeout(() => {
      if (deletion.isConnected) {
        deletion.dataset.confirming = "false";
        deletion.textContent = "Excluir";
      }
    }, 5000);
  }
});

document.addEventListener("submit", async (event) => {
  const registrationForm = event.target.closest("[data-register-form]");
  if (registrationForm) {
    validateRegistrationForm(registrationForm);
    if (!registrationForm.checkValidity()) {
      event.preventDefault();
      focusFirstInvalidField(registrationForm);
      return;
    }
    registrationForm.classList.add("is-submitting");
    registrationForm.querySelector("button[type=submit]")?.setAttribute("aria-busy", "true");
    return;
  }

  const form = event.target.closest("[data-ad-form]");
  if (!form || form.dataset.uploadReady === "true") return;
  event.preventDefault();
  event.stopImmediatePropagation();

  const editor = initializeEditor(form);
  const error = validatePhotos(editor.photos);
  if (error) {
    showUploadStatus(form, error, true);
    return;
  }

  const button = form.querySelector("button[type=submit]");
  button.disabled = true;
  button.classList.add("is-loading");
  try {
    const urls = [];
    for (const [index, photo] of editor.photos.entries()) {
      showUploadStatus(form, `Preparando foto ${index + 1} de ${editor.photos.length}...`);
      urls.push(photo.url || await uploadPhoto(photo.file));
    }
    form.querySelectorAll('input[name="urls_fotos"]').forEach((input) => input.remove());
    for (const url of urls) {
      const input = document.createElement("input");
      input.type = "hidden";
      input.name = "urls_fotos";
      input.value = url;
      form.append(input);
    }
    form.dataset.uploadReady = "true";
    form.requestSubmit();
  } catch (error) {
    showUploadStatus(form, error.message || "Não foi possível enviar as fotos.", true);
    button.disabled = false;
    button.classList.remove("is-loading");
  }
}, true);

document.addEventListener("change", (event) => {
  const input = event.target.closest("[data-photo-input]");
  if (!input) return;
  const form = input.closest("[data-ad-form]");
  const editor = initializeEditor(form);
  for (const file of input.files) {
    if (editor.photos.length >= 5) break;
    editor.photos.push({ file, preview: URL.createObjectURL(file) });
  }
  input.value = "";
  renderPhotos(editor);
});

document.addEventListener("click", (event) => {
  const control = event.target.closest("[data-photo-action]");
  if (!control) return;
  const form = control.closest("[data-ad-form]");
  const editor = initializeEditor(form);
  const item = control.closest(".photo-preview-item");
  const index = [...editor.preview.children].indexOf(item);
  if (index < 0) return;
  const action = control.dataset.photoAction;
  if (action === "remove") {
    const [removed] = editor.photos.splice(index, 1);
    if (removed?.file) URL.revokeObjectURL(removed.preview);
  } else if (action === "left" && index > 0) {
    [editor.photos[index - 1], editor.photos[index]] = [editor.photos[index], editor.photos[index - 1]];
  } else if (action === "right" && index < editor.photos.length - 1) {
    [editor.photos[index + 1], editor.photos[index]] = [editor.photos[index], editor.photos[index + 1]];
  }
  renderPhotos(editor);
});

document.addEventListener("input", (event) => {
  const input = event.target.closest("[data-register-form] input");
  if (!input) return;
  applyRegistrationMask(input);
  clearClientError(input);
  if (input.matches("[data-password]")) {
    updatePasswordStrength(input);
    const confirmation = input.form.querySelector("[data-password-confirmation]");
    if (confirmation?.value) validateRegistrationInput(confirmation);
  }
  updateRegistrationProgress(input.form, input.closest("[data-registration-section]"));
});

document.addEventListener("focusout", (event) => {
  const input = event.target.closest("[data-register-form] input");
  if (!input) return;
  validateRegistrationInput(input);
  updateRegistrationProgress(input.form, input.closest("[data-registration-section]"));
});

document.addEventListener("focusin", (event) => {
  const section = event.target.closest("[data-register-form] [data-registration-section]");
  if (section) updateRegistrationProgress(section.closest("form"), section);
});

document.addEventListener("click", (event) => {
  const toggle = event.target.closest("[data-password-toggle]");
  if (!toggle) return;
  const input = document.getElementById(toggle.getAttribute("aria-controls"));
  if (!input) return;
  const showing = input.type === "text";
  input.type = showing ? "password" : "text";
  toggle.textContent = showing ? "Mostrar" : "Ocultar";
  toggle.setAttribute("aria-label", showing ? "Mostrar senha" : "Ocultar senha");
  input.focus({ preventScroll: true });
});

document.addEventListener("htmx:load", (event) => {
  event.detail.elt.querySelectorAll?.("[data-ad-form]").forEach(initializeEditor);
  event.detail.elt.querySelectorAll?.("[data-register-form]").forEach(initializeRegistrationForm);
  scheduleToasts();
});
document.addEventListener("htmx:responseError", () => {
  document.querySelectorAll("[data-register-form].is-submitting").forEach(resetRegistrationSubmission);
});
document.addEventListener("DOMContentLoaded", () => {
  document.querySelectorAll("[data-ad-form]").forEach(initializeEditor);
  document.querySelectorAll("[data-register-form]").forEach(initializeRegistrationForm);
  scheduleToasts();
});

function initializeRegistrationForm(form) {
  if (registrationForms.has(form)) return;
  registrationForms.add(form);
  form.noValidate = true;
  form.querySelectorAll("input").forEach((input) => {
    applyRegistrationMask(input);
    if (input.matches("[data-password]")) updatePasswordStrength(input);
  });
  const invalid = form.querySelector('[aria-invalid="true"]');
  const activeSection = invalid?.closest("[data-registration-section]") ||
    form.querySelector("[data-registration-section]");
  updateRegistrationProgress(form, activeSection);
}

function applyRegistrationMask(input) {
  if (input.dataset.stateCode !== undefined) {
    input.value = input.value.replace(/[^a-z]/gi, "").slice(0, 2).toUpperCase();
    return;
  }
  const digits = input.value.replace(/\D/g, "");
  if (input.dataset.mask === "cpf") {
    input.value = digits.slice(0, 11)
      .replace(/^(\d{3})(\d)/, "$1.$2")
      .replace(/^(\d{3})\.(\d{3})(\d)/, "$1.$2.$3")
      .replace(/\.(\d{3})(\d)/, ".$1-$2");
  } else if (input.dataset.mask === "cep") {
    input.value = digits.slice(0, 8).replace(/^(\d{5})(\d)/, "$1-$2");
  } else if (input.dataset.mask === "phone") {
    const value = digits.slice(0, 11);
    input.value = value.length > 10
      ? value.replace(/^(\d{2})(\d{5})(\d{0,4})/, "($1) $2-$3")
      : value.replace(/^(\d{2})(\d{4})(\d{0,4})/, "($1) $2-$3");
  }
}

function validateRegistrationForm(form) {
  form.querySelectorAll("input").forEach(validateRegistrationInput);
}

function validateRegistrationInput(input) {
  clearClientError(input);
  let message = "";
  const value = input.value.trim();
  const digits = value.replace(/\D/g, "");
  if (input.validity.valueMissing) {
    message = "Preencha este campo.";
  } else if (input.validity.typeMismatch) {
    message = "Informe um e-mail válido.";
  } else if (input.validity.tooShort) {
    message = `Use pelo menos ${input.minLength} caracteres.`;
  } else if (input.name === "cpf" && value && !isValidCPF(digits)) {
    message = "Informe um CPF válido.";
  } else if (input.name === "telefone" && value && ![10, 11].includes(digits.length)) {
    message = "Informe um telefone com DDD.";
  } else if (input.name === "cep" && value && digits.length !== 8) {
    message = "O CEP deve conter 8 dígitos.";
  } else if (input.name === "estado" && value && !brazilianStates.has(value.toUpperCase())) {
    message = "Informe uma sigla de estado válida.";
  } else if (input.matches("[data-password]") && value && value.length < 8) {
    message = "A senha deve conter pelo menos 8 caracteres.";
  } else if (input.matches("[data-password-confirmation]") && value !== input.form.elements.senha.value) {
    message = "As senhas informadas não coincidem.";
  }
  setClientError(input, message);
  return message === "";
}

function setClientError(input, message) {
  input.setCustomValidity(message);
  if (!message) return;
  input.setAttribute("aria-invalid", "true");
  const error = document.createElement("small");
  error.className = "field-error";
  error.dataset.clientError = "true";
  error.id = `erro-cliente-${input.name}`;
  error.setAttribute("role", "alert");
  error.textContent = message;
  input.closest(".field")?.append(error);
  appendDescription(input, error.id);
}

function clearClientError(input) {
  input.setCustomValidity("");
  input.closest(".field")?.querySelectorAll(".field-error").forEach((error) => error.remove());
  input.removeAttribute("aria-invalid");
  input.setAttribute(
    "aria-describedby",
    (input.getAttribute("aria-describedby") || "")
      .split(/\s+/)
      .filter((id) => id && !id.startsWith("erro-"))
      .join(" "),
  );
  if (!input.getAttribute("aria-describedby")) input.removeAttribute("aria-describedby");
}

function appendDescription(input, id) {
  const ids = new Set((input.getAttribute("aria-describedby") || "").split(/\s+/).filter(Boolean));
  ids.add(id);
  input.setAttribute("aria-describedby", [...ids].join(" "));
}

function updatePasswordStrength(input) {
  const meter = input.form.querySelector("[data-password-strength]");
  if (!meter) return;
  const value = input.value;
  let level = 0;
  if (value.length >= 8) level++;
  if (value.length >= 12) level++;
  if (/[a-z]/i.test(value) && /\d/.test(value)) level++;
  if (/[^a-z0-9]/i.test(value)) level++;
  level = Math.min(level, 4);
  meter.dataset.level = String(level);
  const labels = [
    "Use pelo menos 8 caracteres.",
    "Senha básica.",
    "Senha razoável.",
    "Senha forte.",
    "Senha muito forte.",
  ];
  meter.querySelector("small").textContent = labels[level];
}

function updateRegistrationProgress(form, activeSection) {
  if (!form) return;
  const sections = [...form.querySelectorAll("[data-registration-section]")];
  const steps = document.querySelectorAll("[data-registration-step]");
  const activeName = activeSection?.dataset.registrationSection || "identity";
  steps.forEach((step) => {
    const section = sections.find((item) => item.dataset.registrationSection === step.dataset.registrationStep);
    step.classList.toggle("is-current", step.dataset.registrationStep === activeName);
    step.classList.toggle("is-complete", section ? section.querySelectorAll(":invalid").length === 0 : false);
  });
}

function focusFirstInvalidField(form) {
  const invalid = form.querySelector(":invalid");
  if (!invalid) return;
  updateRegistrationProgress(form, invalid.closest("[data-registration-section]"));
  invalid.focus({ preventScroll: true });
  invalid.scrollIntoView({ behavior: "smooth", block: "center" });
}

function resetRegistrationSubmission(form) {
  form.classList.remove("is-submitting");
  form.querySelector("button[type=submit]")?.removeAttribute("aria-busy");
}

function isValidCPF(value) {
  if (!/^\d{11}$/.test(value) || /^(\d)\1+$/.test(value)) return false;
  const calculateDigit = (base, initialWeight) => {
    const sum = [...base].reduce((total, digit, index) =>
      total + Number(digit) * (initialWeight - index), 0);
    const remainder = (sum * 10) % 11;
    return remainder === 10 ? 0 : remainder;
  };
  return calculateDigit(value.slice(0, 9), 10) === Number(value[9]) &&
    calculateDigit(value.slice(0, 10), 11) === Number(value[10]);
}

function initializeEditor(form) {
  if (editors.has(form)) return editors.get(form);
  const preview = form.querySelector("[data-photo-preview]");
  const photos = [...preview.querySelectorAll("[data-existing-photo]")].map((item) => ({
    url: item.dataset.existingPhoto,
    preview: item.dataset.existingPhoto,
  }));
  const editor = { form, preview, photos };
  editors.set(form, editor);
  return editor;
}

function renderPhotos(editor) {
  editor.preview.replaceChildren(...editor.photos.map((photo, index) => {
    const item = document.createElement("div");
    item.className = "photo-preview-item";
    const image = document.createElement("img");
    image.src = photo.preview;
    image.alt = `Foto ${index + 1}`;
    image.referrerPolicy = "no-referrer";
    const label = document.createElement("span");
    label.textContent = index === 0 ? "Capa" : String(index + 1);
    const controls = document.createElement("div");
    controls.className = "photo-preview-controls";
    controls.append(
      photoButton("left", "←", "Mover foto para a esquerda", index === 0),
      photoButton("right", "→", "Mover foto para a direita", index === editor.photos.length - 1),
      photoButton("remove", "×", "Remover foto"),
    );
    item.append(image, label, controls);
    return item;
  }));
}

function photoButton(action, text, label, disabled = false) {
  const button = document.createElement("button");
  button.type = "button";
  button.dataset.photoAction = action;
  button.textContent = text;
  button.setAttribute("aria-label", label);
  button.disabled = disabled;
  return button;
}

function validatePhotos(photos) {
  if (photos.length < 2 || photos.length > 5) return "Mantenha entre 2 e 5 imagens.";
  const invalid = photos.some(({ file, url }) => file
    ? !allowedTypes.has(file.type) || file.size <= 0 || file.size > 5 * 1024 * 1024
    : !isPublicImageURL(url));
  return invalid ? "Use imagens JPEG, PNG ou WebP com até 5 MB." : "";
}

function showUploadStatus(form, message, error = false) {
  const status = form.querySelector("[data-upload-status]");
  status.textContent = message;
  status.classList.remove("hidden");
  status.classList.toggle("field-error", error);
}

function isPublicImageURL(value) {
  try {
    const url = new URL(value);
    return url.protocol === "https:" && !url.username && !url.password &&
      !url.search && !url.hash &&
      url.hostname.endsWith(".public.blob.vercel-storage.com");
  } catch {
    return false;
  }
}

function scheduleToasts() {
  document.querySelectorAll(".toast:not([data-scheduled])").forEach((toast) => {
    toast.dataset.scheduled = "true";
    window.setTimeout(() => {
      toast.classList.add("is-leaving");
      window.setTimeout(() => toast.remove(), 200);
    }, 3200);
  });
}
