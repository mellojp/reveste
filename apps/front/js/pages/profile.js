import { request } from "../core/api.js";
import { clearFormErrors, showFormErrors } from "../core/forms.js";
import { toast } from "../core/notifications.js";
import { state } from "../core/state.js";
import { renderHeader } from "../components/shell.js";
import { escapeHTML } from "../core/utils.js";
import { pageSkeleton, revealContent, setButtonLoading } from "../core/feedback.js";

export async function profilePage(root) {
  root.innerHTML = pageSkeleton("Carregando perfil");
  try {
    const user = await request("/v1/me");
    saveLocalUser(user);
    renderProfile(root, user);
  } catch (error) {
    toast(error.message);
    root.innerHTML = `<section class="page-section"><div class="empty-state"><h2>Não foi possível abrir seu perfil</h2><p>${escapeHTML(error.message)}</p></div></section>`;
  }
}

function renderProfile(root, user, editing = false) {
  const address = user.endereco_principal;
  const initials = user.nome.split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
  root.innerHTML = `
    <section class="profile-page page-section compact-section">
      <div class="profile-heading">
        <div class="profile-avatar">${escapeHTML(initials)}</div>
        <div><span class="eyebrow">Minha conta</span><h1>${escapeHTML(user.nome)}</h1><p>${escapeHTML(user.email)}</p></div>
      </div>
      <div class="profile-layout">
        <nav class="account-nav" aria-label="Navegação da conta">
          <a class="active" href="/perfil" data-link>Dados pessoais</a>
          <a href="/meus-anuncios" data-link>Meus anúncios</a>
          <a href="/carrinho" data-link>Minha sacola</a>
          <button type="button" data-action="logout">Sair da conta</button>
        </nav>
        <div class="profile-content">
          ${editing ? editForm(user) : profileDetails(user, address)}
        </div>
      </div>
    </section>`;

  root.querySelector("#edit-profile")?.addEventListener("click", () => renderProfile(root, user, true));
  root.querySelector("#cancel-profile")?.addEventListener("click", () => renderProfile(root, user));
  root.querySelector("#profile-form")?.addEventListener("submit", async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const values = Object.fromEntries(new FormData(form));
    const button = form.querySelector("button[type=submit]");
    clearFormErrors(form);
    setButtonLoading(button, true, "Salvando...");
    try {
      const updated = await request("/v1/me", {
        method: "PATCH",
        body: JSON.stringify({
          nome: values.nome, email: values.email, telefone: values.telefone,
          endereco: {
            cep: values.cep, logradouro: values.logradouro, numero: values.numero,
            complemento: values.complemento, bairro: values.bairro,
            cidade: values.cidade, estado: values.estado,
          },
        }),
      });
      saveLocalUser(updated);
      renderHeader();
      renderProfile(root, updated);
      toast("Perfil atualizado.");
    } catch (error) {
      showFormErrors(form, error.fields);
      toast(error.message);
    } finally {
      setButtonLoading(button, false);
    }
  });
  revealContent(root.querySelector(".profile-content"));
}

function profileDetails(user, address) {
  return `
    <section class="detail-panel">
      <div class="panel-heading"><div><span class="eyebrow">Cadastro</span><h2>Dados pessoais</h2></div><button class="button button-outline" id="edit-profile" type="button">Editar perfil</button></div>
      <dl class="detail-grid">
        <div><dt>Nome</dt><dd>${escapeHTML(user.nome)}</dd></div>
        <div><dt>E-mail</dt><dd>${escapeHTML(user.email)}</dd></div>
        <div><dt>Telefone</dt><dd>${escapeHTML(user.telefone || "Não informado")}</dd></div>
        <div><dt>Conta criada em</dt><dd>${new Date(user.criado_em).toLocaleDateString("pt-BR")}</dd></div>
      </dl>
    </section>
    <section class="detail-panel">
      <div class="panel-heading"><div><span class="eyebrow">Entregas</span><h2>Endereço principal</h2></div></div>
      <dl class="detail-grid">
        <div class="span-2"><dt>Endereço</dt><dd>${escapeHTML(`${address.logradouro}, ${address.numero}${address.complemento ? ` · ${address.complemento}` : ""}`)}</dd></div>
        <div><dt>Bairro</dt><dd>${escapeHTML(address.bairro)}</dd></div>
        <div><dt>CEP</dt><dd>${escapeHTML(address.cep)}</dd></div>
        <div><dt>Cidade</dt><dd>${escapeHTML(address.cidade)}</dd></div>
        <div><dt>Estado</dt><dd>${escapeHTML(address.estado)}</dd></div>
      </dl>
    </section>`;
}

function editForm(user) {
  const address = user.endereco_principal;
  return `
    <section class="detail-panel">
      <div class="panel-heading"><div><span class="eyebrow">Editar cadastro</span><h2>Seus dados</h2></div></div>
      <form class="stack-form" id="profile-form">
        <div class="form-grid">
          <label class="span-2">Nome completo<input name="nome" value="${escapeHTML(user.nome)}" required></label>
          <label class="span-2">E-mail<input name="email" type="email" value="${escapeHTML(user.email)}" required></label>
          <label class="span-2">Telefone<input name="telefone" type="tel" value="${escapeHTML(user.telefone || "")}"></label>
          <label>CEP<input name="cep" value="${escapeHTML(address.cep)}" required></label>
          <label>Estado<input name="estado" maxlength="2" value="${escapeHTML(address.estado)}" required></label>
          <label class="span-2">Logradouro<input name="logradouro" value="${escapeHTML(address.logradouro)}" required></label>
          <label>Número<input name="numero" value="${escapeHTML(address.numero)}" required></label>
          <label>Complemento<input name="complemento" value="${escapeHTML(address.complemento || "")}"></label>
          <label>Bairro<input name="bairro" value="${escapeHTML(address.bairro)}" required></label>
          <label>Cidade<input name="cidade" value="${escapeHTML(address.cidade)}" required></label>
        </div>
        <div class="form-submit"><button class="text-link" id="cancel-profile" type="button">Cancelar</button><button class="button button-dark" type="submit">Salvar perfil</button></div>
      </form>
    </section>`;
}

function saveLocalUser(user) {
  state.user = user;
  sessionStorage.setItem("reveste_user", JSON.stringify(user));
}

profilePage.title = "Meu perfil";
