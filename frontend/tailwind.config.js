const defaultTheme = require("tailwindcss/defaultTheme");
// const colors = require("tailwindcss/colors");

module.exports = {
  future: {
    // removeDeprecatedGapUtilities: true,
    // purgeLayersByDefault: true,
  },
  content: ["./index.html", "./src/**/*.{vue,js,ts,jsx,tsx}"],
  mode: "jit",
  safelist: [
    // "w-xxx" is used by BBTab
    // "pl-xxx" is used by BBOutline
    // "bg-xxx", "text-xxx" is used by BBAttention TaskCheckBadgeBar BBBadge
    // "hover:bg-xxx" is used by TaskCheckBadgeBar BBBadge
    // "grid-cols-xxx" is used by AnomalyCenterDashboard
    { pattern: /^w-/ },
    { pattern: /^pl-/ },
    { pattern: /^bg-gray-/, variants: ["hover"] },
    { pattern: /^bg-blue-/, variants: ["hover"] },
    { pattern: /^bg-yellow-/, variants: ["hover"] },
    { pattern: /^bg-red-/, variants: ["hover"] },
    { pattern: /^bg-indigo-/, variants: ["hover"] },
    { pattern: /^text-indigo-/ },
    { pattern: /^text-gray-/ },
    { pattern: /^text-blue-/ },
    { pattern: /^text-yellow-/ },
    { pattern: /^text-red-/ },
    { pattern: /^grid-cols-/ },
    { pattern: /^(uppercase|lowercase|capitalize)$/ },
  ],
  theme: {
    extend: {
      screens: {
        "3xl": "1920px",
        "4xl": "2160px",
      },
      colors: {
        accent: "rgb(var(--color-accent) / <alpha-value>)",
        "accent-tw": "rgb(var(--color-accent-tw) / <alpha-value>)",
        "accent-disabled": "rgb(var(--color-accent-disabled) / <alpha-value>)",
        "accent-hover": "rgb(var(--color-accent-hover) / <alpha-value>)",
        "accent-text": "rgb(var(--color-accent-text) / <alpha-value>)",

        main: "rgb(var(--color-main) / <alpha-value>)",
        "main-hover": "rgb(var(--color-main-hover) / <alpha-value>)",
        "main-text": "rgb(var(--color-main-text) / <alpha-value>)",

        control: "rgb(var(--color-control) / <alpha-value>)",
        "control-hover": "rgb(var(--color-control-hover) / <alpha-value>)",

        "control-light": "rgb(var(--color-control-light) / <alpha-value>)",
        "control-light-hover":
          "rgb(var(--color-control-light-hover) / <alpha-value>)",

        "control-bg": "rgb(var(--color-control-bg) / <alpha-value>)",
        "control-bg-hover":
          "rgb(var(--color-control-bg-hover) / <alpha-value>)",

        "control-placeholder":
          "rgb(var(--color-control-placeholder) / <alpha-value>)",

        info: "rgb(var(--color-info) / <alpha-value>)",
        "info-hover": "rgb(var(--color-info-hover) / <alpha-value>)",

        warning: "rgb(var(--color-warning) / <alpha-value>)",
        "warning-hover": "rgb(var(--color-warning-hover) / <alpha-value>)",

        error: "rgb(var(--color-error) / <alpha-value>)",
        "error-hover": "rgb(var(--color-error-hover) / <alpha-value>)",

        success: "rgb(var(--color-success) / <alpha-value>)",
        "success-hover": "rgb(var(--color-success-hover) / <alpha-value>)",

        "link-hover": "rgb(var(--color-link-hover) / <alpha-value>)",

        "block-border": "rgb(var(--color-block-border) / <alpha-value>)",
        "control-border": "rgb(var(--color-control-border) / <alpha-value>)",

        "dark-bg": "rgb(var(--color-dark-bg) / <alpha-value>)",
        "matrix-green": "rgb(var(--color-matrix-green) / <alpha-value>)",
        "matrix-green-hover":
          "rgb(var(--color-matrix-green-hover) / <alpha-value>)",
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
        320: "80rem",
      },
      animation: {
        "ping-slow": "ping-slow 2500ms cubic-bezier(0.4, 0, 0.6, 1) infinite",
      },
      keyframes: {
        "ping-slow": {
          "50%": {
            transform: "scale(3)",
            opacity: "0.05",
          },
          "100%": {
            transform: "scale(3)",
            opacity: "0",
          },
        },
      },
    },
  },
  darkMode: "class",
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
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};
