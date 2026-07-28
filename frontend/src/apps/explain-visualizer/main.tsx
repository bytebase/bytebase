import "bootstrap/dist/css/bootstrap.css";
import "pev2/dist/pev2.css";
import "html-query-plan/css/qp.css";
import "./explain-visualizer.css";
import { createRoot } from "react-dom/client";
import { ExplainVisualizerApp } from "./ExplainVisualizerApp";

// Some bundled dependencies expect a `global` reference on the window.
(globalThis as typeof globalThis & Record<string, unknown>).global = globalThis;

// Force Font Awesome SVG sizing into <head> before pev2 mounts. pev2
// renders icons via @fortawesome/vue-fontawesome and relies on the
// `autoAddCss` runtime mechanism to inject these rules. That injection
// doesn't fire reliably when pev2 is mounted as a Vue island inside
// React, leaving SVGs unconstrained (they expand to fill their parent,
// blowing up the info-circle icon to viewport size). We can't import
// `@fortawesome/fontawesome-svg-core/styles.css` directly because it's
// a transitive dep that pnpm doesn't hoist to the top-level
// node_modules. So inline the rules — marked `!important` because the
// pev2 runtime can also inject conflicting CSS later.
const STYLE_ID = "pev2-fa-svg-core";
if (!document.getElementById(STYLE_ID)) {
  const el = document.createElement("style");
  el.id = STYLE_ID;
  el.textContent = `
    .svg-inline--fa{display:inline-block!important;height:1em!important;width:auto!important;overflow:visible!important;vertical-align:-.125em!important}
    svg.svg-inline--fa:not(:root){overflow:visible!important;box-sizing:content-box!important}
    .svg-inline--fa.fa-fw{width:1.25em!important}
  `;
  document.head.appendChild(el);
}

const container = document.getElementById("app");
if (container) {
  createRoot(container).render(<ExplainVisualizerApp />);
}
