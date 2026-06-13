import { request } from "../core/api.js";
import { clearFormErrors, showFormErrors } from "../core/forms.js";
import { toast } from "../core/notifications.js";
import { navigate } from "../core/router.js";
import { loadCart, saveSession, state } from "../core/session.js";
import { renderHeader } from "../components/shell.js";
import { escapeHTML } from "../core/utils.js";

function authLayout(kind, content) {
  const isLogin = kind === "login";
  return `
    <section class="auth-page">
      <div class="auth-visual">
        <a class="brand brand-light" href="/" data-link><img src="/assets/logo-light.svg" alt="ReVeste"></a>
        <div>
          <span class="eyebrow">Moda de pessoa para pessoa</span>
          <h1>${isLogin ? "Sua próxima descoberta continua aqui." : "Uma conta para comprar, vender e fazer circular."}</h1>
        </div>
        <p>Peças únicas. Novas possibilidades.</p>
      </div>
      <div class="auth-content">
        <div class="auth-card">${content}</div>
      </div>
    </section>`;
}

export async function loginPage(root) {
  if (state.token) {
    navigate("/perfil", { replace: true });
    return;
  }
  const query = new URLSearchParams(window.location.search);
  const returnQuery = query.get("retorno")
    ? `?retorno=${encodeURIComponent(query.get("retorno"))}`
    : "";
  root.innerHTML = authLayout("login", `
    <span class="eyebrow">Boas-vindas</span>
    <h2>Entre na sua conta</h2>
    <p class="muted">Acesse seus anúncios e sua sacola.</p>
    <form class="stack-form" id="login-form">
      <label>E-mail ou CPF<input name="identificador" value="${escapeHTML(query.get("email") || "")}" autocomplete="username" required></label>
      <label>Senha<input name="senha" type="password" autocomplete="current-password" required></label>
      <button class="button button-dark button-large" type="submit">Entrar</button>
    </form>
    <p class="auth-switch">Ainda não tem conta? <a href="/cadastro${returnQuery}" data-link>Cadastre-se</a></p>
  `);

  const form = root.querySelector("#login-form");
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const button = form.querySelector("button[type=submit]");
    clearFormErrors(form);
    button.disabled = true;
    try {
      const session = await request("/v1/sessoes", {
        method: "POST",
        body: JSON.stringify(Object.fromEntries(new FormData(form))),
      });
      saveSession(session);
      await loadCart();
      renderHeader();
      toast(`Olá, ${session.usuario.nome.split(" ")[0]}.`);
      navigate(safeReturnPath(query.get("retorno")));
    } catch (error) {
      showFormErrors(form, error.fields);
      toast(error.message);
    } finally {
      button.disabled = false;
    }
  });
}

function safeReturnPath(value) {
  return value?.startsWith("/") && !value.startsWith("//") ? value : "/catalogo";
}

export async function registerPage(root) {
  if (state.token) {
    navigate("/perfil", { replace: true });
    return;
  }
  const query = new URLSearchParams(window.location.search);
  const returnQuery = query.get("retorno")
    ? `&retorno=${encodeURIComponent(query.get("retorno"))}`
    : "";
  root.innerHTML = authLayout("register", `
    <span class="eyebrow">Comece por aqui</span>
    <h2>Crie sua conta</h2>
    <p class="muted">Seus dados também preparam as futuras entregas.</p>
    <form class="stack-form register-form" id="register-form">
      <fieldset><legend>Dados pessoais</legend>
        <div class="form-grid">
          <label class="span-2">Nome completo<input name="nome" autocomplete="name" required></label>
          <label>CPF<input name="cpf" inputmode="numeric" autocomplete="off" required></label>
          <label>Telefone<input name="telefone" type="tel" autocomplete="tel"></label>
          <label class="span-2">E-mail<input name="email" type="email" autocomplete="email" required></label>
          <label class="span-2">Senha<input name="senha" type="password" minlength="8" autocomplete="new-password" required></label>
        </div>
      </fieldset>
      <fieldset><legend>Endereço principal</legend>
        <div class="form-grid">
          <label>CEP<input name="cep" autocomplete="postal-code" required></label>
          <label>Estado<input name="estado" maxlength="2" autocomplete="address-level1" required></label>
          <label class="span-2">Logradouro<input name="logradouro" autocomplete="address-line1" required></label>
          <label>Número<input name="numero" required></label>
          <label>Bairro<input name="bairro" required></label>
          <label class="span-2">Cidade<input name="cidade" autocomplete="address-level2" required></label>
        </div>
      </fieldset>
      <button class="button button-dark button-large" type="submit">Criar conta</button>
    </form>
    <p class="auth-switch">Já tem uma conta? <a href="/entrar${query.get("retorno") ? `?retorno=${encodeURIComponent(query.get("retorno"))}` : ""}" data-link>Entrar</a></p>
  `);

  const form = root.querySelector("#register-form");
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const values = Object.fromEntries(new FormData(form));
    const button = form.querySelector("button[type=submit]");
    clearFormErrors(form);
    button.disabled = true;
    const payload = {
      nome: values.nome,
      cpf: values.cpf,
      email: values.email,
      senha: values.senha,
      telefone: values.telefone,
      endereco: {
        cep: values.cep,
        logradouro: values.logradouro,
        numero: values.numero,
        complemento: "",
        bairro: values.bairro,
        cidade: values.cidade,
        estado: values.estado,
      },
    };
    try {
      await request("/v1/usuarios", { method: "POST", body: JSON.stringify(payload) });
      toast("Conta criada. Agora entre com sua senha.");
      navigate(`/entrar?email=${encodeURIComponent(payload.email)}${returnQuery}`);
    } catch (error) {
      showFormErrors(form, error.fields);
      toast(error.message);
    } finally {
      button.disabled = false;
    }
  });
}

loginPage.title = "Entrar";
registerPage.title = "Criar conta";
