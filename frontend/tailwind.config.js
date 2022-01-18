const defaultTheme = require("tailwindcss/defaultTheme");
// const colors = require("tailwindcss/colors");

module.exports = {
  future: {
    // removeDeprecatedGapUtilities: true,
    // purgeLayersByDefault: true,
  },
  purge: {
    content: ["./index.html", "./src/**/*.{vue,js,ts,jsx,tsx}"],
    // "w-xxx" is used by BBTab
    // "pl-xxx" is used by BBOutline
    // "bg-xxx", "text-xxx" is used by BBAttention TaskCheckBadgeBar
    // "hover:bg-xxx" is used by TaskCheckBadgeBar
    // "grid-cols-xxx" is used by AnomalyCenterDashboard
    safelist: [
      /^w-/,
      /^pl-/,
      /^bg-gray-/,
      /^text-gray-/,
      /^bg-blue-/,
      /^text-blue-/,
      /^bg-yellow-/,
      /^text-yellow-/,
      /^bg-red-/,
      /^text-red-/,
      /^hover:bg-gray-/,
      /^hover:bg-blue-/,
      /^hover:bg-yellow-/,
      /^hover:bg-red-/,
      /^grid-cols-/,
    ],
  },
  theme: {
    extend: {
      colors: {
        accent: "var(--color-accent)",
        "accent-disabled": "var(--color-accent-disabled)",
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

        info: "var(--color-info)",
        "info-hover": "var(--color-info-hover)",

        warning: "var(--color-warning)",
        "warning-hover": "var(--color-warning-hover)",

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
        176: "44rem",
        192: "48rem",
        208: "52rem",
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
      textColor: ["disabled"],
    },
  },
  plugins: [
    require("@tailwindcss/forms"),
    require("@tailwindcss/line-clamp"),
    require("@tailwindcss/typography"),
  ],
};
