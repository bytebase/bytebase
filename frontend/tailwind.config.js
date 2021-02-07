const defaultTheme = require("tailwindcss/defaultTheme");
const colors = require("tailwindcss/colors");

module.exports = {
  future: {
    // removeDeprecatedGapUtilities: true,
    // purgeLayersByDefault: true,
  },
  purge: [],
  theme: {
    extend: {
      colors: {
        accent: "var(--color-accent)",
        "accent-hover": "var(--color-accent-hover)",
        "accent-text": "var(--color-accent-text)",

        normal: "var(--color-normal)",
        "normal-hover": "var(--color-normal-hover)",

        main: "var(--color-main)",

        control: "var(--color-control)",
        "control-hover": "var(--color-control-hover)",

        "control-light": "var(--color-control-light)",
        "control-light-hover": "var(--color-control-light-hover)",

        "control-placeholder": "var(--color-control-placeholder)",

        error: "var(--color-error)",
        "error-hover": "var(--color-error-hover)",

        "link-hover": "var(--color-link-hover)",

        "block-border": "var(--color-block-border)",
        "control-border": "var(--color-control-border)",
      },
      fontFamily: {
        sans: ["Inter var", ...defaultTheme.fontFamily.sans],
      },
    },
  },
  variants: {
    extend: {
      ringWidth: ["focus-visible"],
      opacity: ["disabled"],
      backgroundColor: ["disabled"],
      cursor: ["disabled"],
    },
  },
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};
