import "highlight.js/lib/common";
import hljs from "highlight.js/lib/core";
import sql from "highlight.js/lib/languages/sql";
import "highlight.js/styles/github.css";
import type { App } from "vue";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";

export default {
  install(app: App) {
    hljs.registerLanguage("sql", sql);
    hljs.configure({
      languages: ["sql"],
    });
    app.component("HighlightCodeBlock", HighlightCodeBlock);
  },
};
