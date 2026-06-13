export async function notFoundPage(root) {
  root.innerHTML = `
    <section class="not-found page-section">
      <span class="error-number">404</span>
      <span class="eyebrow">Página não encontrada</span>
      <h1>Essa peça saiu da arara.</h1>
      <p>O endereço pode ter mudado ou não existe.</p>
      <a class="button button-dark" href="/" data-link>Voltar ao início</a>
    </section>`;
}

notFoundPage.title = "Página não encontrada";
