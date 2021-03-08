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
      screens: {
        xs: "425px",
      },
      colors: {
        accent: "var(--color-accent)",
        "accent-hover": "var(--color-accent-hover)",
        "accent-text": "var(--color-accent-text)",

        main: "var(--color-main)",
        "main-hover": "var(--color-main-hover)",
        "main-text": "var(--color-main-text)",

        control: "var(--color-control)",
        "control-hover": "var(--color-control-hover)",

        "control-light": "var(--color-control-light)",
        "control-light-hover": "var(--color-control-light-hover)",

        "control-bg": "var(--color-control-bg)",
        "control-bg-hover": "var(--color-control-bg-hover)",

        "control-placeholder": "var(--color-control-placeholder)",

        error: "var(--color-error)",
        "error-hover": "var(--color-error-hover)",

        success: "var(--color-success)",
        "success-hover": "var(--color-success-hover)",

        "link-hover": "var(--color-link-hover)",

        "block-border": "var(--color-block-border)",
        "control-border": "var(--color-control-border)",
      },
      fontFamily: {
        sans: ["Inter var", ...defaultTheme.fontFamily.sans],
      },
      spacing: {
        112: "28rem",
        128: "32rem",
        144: "36rem",
        160: "40rem",
      },
    },
  },
  variants: {
    extend: {
      ringWidth: ["focus-visible"],
      opacity: ["disabled"],
      backgroundColor: ["disabled"],
      cursor: ["disabled"],
      margin: ["focus"],
    },
  },
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};
