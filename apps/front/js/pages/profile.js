import { request } from "../core/api.js";
import { toast } from "../core/notifications.js";
import { state } from "../core/state.js";
import { escapeHTML } from "../core/utils.js";

export async function profilePage(root) {
  root.innerHTML = `<div class="page-loading">Carregando perfil...</div>`;
  try {
    const user = await request("/v1/me");
    state.user = user;
    sessionStorage.setItem("reveste_user", JSON.stringify(user));
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
            <section class="detail-panel">
              <div class="panel-heading"><div><span class="eyebrow">Cadastro</span><h2>Dados pessoais</h2></div><span class="status-badge">Ativo</span></div>
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
                <div class="span-2"><dt>Endereço</dt><dd>${escapeHTML(`${address.logradouro}, ${address.numero}`)}</dd></div>
                <div><dt>Bairro</dt><dd>${escapeHTML(address.bairro)}</dd></div>
                <div><dt>CEP</dt><dd>${escapeHTML(address.cep)}</dd></div>
                <div><dt>Cidade</dt><dd>${escapeHTML(address.cidade)}</dd></div>
                <div><dt>Estado</dt><dd>${escapeHTML(address.estado)}</dd></div>
              </dl>
            </section>
            <p class="feature-note">A edição do perfil será habilitada quando o contrato correspondente estiver disponível na API.</p>
          </div>
        </div>
      </section>`;
  } catch (error) {
    toast(error.message);
    root.innerHTML = `<section class="page-section"><div class="empty-state"><h2>Não foi possível abrir seu perfil</h2><p>${escapeHTML(error.message)}</p></div></section>`;
  }
}

profilePage.title = "Meu perfil";
