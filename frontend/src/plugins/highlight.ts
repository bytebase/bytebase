import type { App } from "vue";
import hljs from "highlight.js/lib/core";
import sql from "highlight.js/lib/languages/sql";
import "highlight.js/lib/common";
import "highlight.js/styles/github.css";
import CodeHighlight from "@/components/CodeHighlight";

export default {
  install(app: App) {
    hljs.registerLanguage("sql", sql);
    hljs.configure({
      languages: ["sql"],
    });
    app.component("CodeHighlight", CodeHighlight);
  },
};
