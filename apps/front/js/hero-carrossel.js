const carrosseisIniciados = new WeakSet();

export function inicializarCarrosselHero(raiz) {
  const escopo = raiz && raiz.querySelectorAll ? raiz : document;
  const encontrados = [...escopo.querySelectorAll("[data-hero-carrossel]")];
  if (escopo.matches?.("[data-hero-carrossel]")) encontrados.unshift(escopo);
  encontrados.forEach(configurarCarrossel);
}

function configurarCarrossel(hero) {
  if (carrosseisIniciados.has(hero)) return;
  const trilha = hero.querySelector("[data-hero-trilha]");
  const pontos = [...hero.querySelectorAll("[data-hero-ponto]")];
  const slides = trilha ? [...trilha.children] : [];
  if (!trilha || slides.length < 2 || pontos.length !== slides.length) return;
  carrosseisIniciados.add(hero);

  const textos = [...hero.querySelectorAll("[data-hero-texto]")];
  const paragrafos = hero.querySelector(".hero-paragraphs");
  let atual = 0;
  let assumidoPeloUsuario = false;
  let sobFoco = false;
  let foraDaTela = false;
  // Ao voltar do ultimo slide para o primeiro, os slides do meio cruzam o observador. Sem
  // este alvo o texto piscaria 4, 3, 2, 1 no caminho de volta.
  let alvoProgramatico = -1;
  let destravamento = 0;

  const observador = new IntersectionObserver((entradas) => {
    entradas.forEach((entrada) => {
      if (!entrada.isIntersecting) return;
      const indice = slides.indexOf(entrada.target);
      if (alvoProgramatico >= 0 && indice !== alvoProgramatico) return;
      alvoProgramatico = -1;
      ativar(indice);
    });
  }, { root: trilha, threshold: 0.6 });
  slides.forEach((slide) => observador.observe(slide));

  // O hero e sticky e some sob a .home-cover ao rolar: sem isto o carrossel giraria fora
  // da tela, e o usuario voltaria para um slide aleatorio.
  const observadorDeTela = new IntersectionObserver(([entrada]) => {
    foraDaTela = !entrada.isIntersecting;
    sincronizarAutoplay();
  });
  observadorDeTela.observe(hero);

  function ativar(indice) {
    atual = indice;
    pontos.forEach((ponto, posicao) => ponto.setAttribute("aria-current", String(posicao === indice)));
    textos.forEach((texto) => {
      const ativo = Number(texto.dataset.heroTexto) === indice + 1;
      texto.classList.toggle("is-active", ativo);
      texto.setAttribute("aria-hidden", String(!ativo));
    });
  }

  function irPara(indice) {
    const destino = (indice + slides.length) % slides.length;
    if (destino === atual) return;
    alvoProgramatico = destino;
    trilha.scrollTo({
      left: slides[destino].offsetLeft - slides[0].offsetLeft,
      behavior: prefereMenosMovimento() ? "auto" : "smooth",
    });
    // Rede de seguranca: se o scroll nao chegar ao destino (snap teimoso, scroll
    // interrompido), o alvo ficaria preso e o observador ignoraria todos os slides para
    // sempre, congelando o carrossel. Aqui ele converge na marra.
    window.clearTimeout(destravamento);
    destravamento = window.setTimeout(() => {
      if (alvoProgramatico < 0) return;
      const alvo = alvoProgramatico;
      alvoProgramatico = -1;
      if (atual !== alvo) ativar(alvo);
    }, 1200);
  }

  // Hover, aba escondida e hero fora da tela pausam por motivos independentes, e podem se
  // sobrepor. Em vez de transicoes de estado (que dependem da ordem dos eventos), cada
  // evento so atualiza sua flag e o estado e derivado das tres de uma vez.
  //   off      -> nao anima, o ponto ativo fica solido
  //   pausado  -> mesma animacao, animation-play-state: paused (retoma de onde parou)
  //   on       -> animando; o animationend avanca o slide
  function sincronizarAutoplay() {
    if (assumidoPeloUsuario || prefereMenosMovimento()) {
      hero.dataset.heroAutoplay = "off";
      return;
    }
    const pausado = sobFoco || foraDaTela || document.hidden;
    hero.dataset.heroAutoplay = pausado ? "pausado" : "on";
  }

  // Uma interacao explicita entrega o controle ao usuario de vez: o carrossel nao volta a
  // girar sozinho, e o texto passa a ser anunciado por leitores de tela.
  function entregarControle() {
    alvoProgramatico = -1;
    assumidoPeloUsuario = true;
    paragrafos?.setAttribute("aria-live", "polite");
    sincronizarAutoplay();
  }

  // A animacao da barrinha e o relogio do autoplay. Pausar por CSS pausa o avanco junto,
  // sem um setInterval paralelo que dessincronizaria da barra a cada hover.
  hero.addEventListener("animationend", (evento) => {
    if (evento.animationName !== "hero-dot-progress") return;
    if (hero.dataset.heroAutoplay === "on") irPara(atual + 1);
  });

  hero.addEventListener("mouseenter", () => { sobFoco = true; sincronizarAutoplay(); });
  hero.addEventListener("mouseleave", () => { sobFoco = false; sincronizarAutoplay(); });
  hero.addEventListener("focusin", () => { sobFoco = true; sincronizarAutoplay(); });
  hero.addEventListener("focusout", () => { sobFoco = hero.matches(":hover"); sincronizarAutoplay(); });
  hero.addEventListener("keydown", (evento) => {
    if (!evento.target.closest?.(".hero-art")) return;
    if (evento.key !== "ArrowRight" && evento.key !== "ArrowLeft") return;
    evento.preventDefault();
    entregarControle();
    irPara(atual + (evento.key === "ArrowRight" ? 1 : -1));
  });

  trilha.addEventListener("pointerdown", entregarControle);
  // So a rolagem horizontal e interacao com o carrossel: rolar a pagina com o cursor sobre
  // a foto e navegacao normal e nao pode desligar o autoplay.
  trilha.addEventListener("wheel", (evento) => {
    if (Math.abs(evento.deltaX) > Math.abs(evento.deltaY)) entregarControle();
  }, { passive: true });
  pontos.forEach((ponto, indice) => {
    ponto.addEventListener("click", () => {
      entregarControle();
      irPara(indice);
    });
  });

  document.addEventListener("visibilitychange", () => {
    if (hero.isConnected) sincronizarAutoplay();
  });

  // Sem autoplay o usuario e quem conduz, entao o texto precisa ser anunciado.
  if (prefereMenosMovimento()) paragrafos?.setAttribute("aria-live", "polite");
  sincronizarAutoplay();
}

function prefereMenosMovimento() {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}
