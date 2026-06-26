// Inicializa o Card Payment Brick do Mercado Pago na tela de pagamento com cartão.
// Carregado apenas nessa página (após o SDK MercadoPago.js). A tokenização acontece no
// navegador (padrão PCI): o número do cartão nunca chega ao nosso backend — só o token,
// que preenchemos nos campos ocultos do formulário e enviamos como um POST normal.
(function () {
  const form = document.querySelector("[data-card-form]");
  if (!form || typeof MercadoPago === "undefined") return;

  const definir = (nome, valor) => {
    const campo = form.querySelector(`[name="${nome}"]`);
    if (campo) campo.value = valor == null ? "" : String(valor);
  };

  const mp = new MercadoPago(form.dataset.publicKey, { locale: "pt-BR" });
  const bricks = mp.bricks();

  bricks.create("cardPayment", "cardPaymentBrick_container", {
    initialization: {
      amount: Number(form.dataset.amount),
    },
    customization: {
      paymentMethods: { maxInstallments: 12 },
      visual: {
        hidePaymentButton: false,
        style: {
          customVariables: {
            baseColor: "#315c46",
            baseColorFirstVariant: "#405844",
            baseColorSecondVariant: "#2f3a2c",
            textPrimaryColor: "#20231f",
            textSecondaryColor: "#6d7169",
            inputBackgroundColor: "#ffffff",
            formBackgroundColor: "#fffdf8",
            buttonTextColor: "#ffffff",
            outlinePrimaryColor: "#315c46",
            borderRadiusSmall: "10px",
            borderRadiusMedium: "12px",
            borderRadiusLarge: "16px",
            fontSizeMedium: "15px",
          },
        },
      },
    },
    callbacks: {
      onReady: () => {},
      onError: (erro) => console.error("cardPayment brick:", erro),
      onSubmit: (formData) =>
        new Promise((resolve) => {
          const ident = (formData.payer && formData.payer.identification) || {};
          definir("token", formData.token);
          definir("payment_method_id", formData.payment_method_id);
          definir("installments", formData.installments || 1);
          definir("identification_type", ident.type);
          definir("identification_number", ident.number);
          // requestSubmit dispara o evento de submit (diferente de form.submit()), então o htmx
          // intercepta e envia via fetch — incluindo o header Origin que a proteção CSRF exige.
          // O backend cria a cobrança e responde com redirect, que o htmx segue.
          form.requestSubmit();
          resolve();
        }),
    },
  });
})();
