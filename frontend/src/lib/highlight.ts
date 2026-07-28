import hljs from "highlight.js/lib/core";
import sql from "highlight.js/lib/languages/sql";
import "highlight.js/styles/github.css";

hljs.registerLanguage("sql", sql);

export default {
  install() {},
};
