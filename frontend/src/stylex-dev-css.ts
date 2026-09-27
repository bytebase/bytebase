const STYLEX_DEV_CSS_ID = "__bytebase_stylex_dev_css__";
const STYLEX_DEV_CSS_PATH = "/virtual:stylex.css";

function installStyleXDevCSS() {
  if (!import.meta.env.DEV) {
    return;
  }

  if (!document.getElementById(STYLEX_DEV_CSS_ID)) {
    const link = document.createElement("link");
    link.id = STYLEX_DEV_CSS_ID;
    link.rel = "stylesheet";
    link.href = STYLEX_DEV_CSS_PATH;
    document.head.appendChild(link);
  }

  void import("virtual:stylex:css-only");
}

installStyleXDevCSS();
