import hljs from "highlight.js";
import "highlight.js/styles/github-gist.css";

hljs.configure({
  languages: ["sql"],
});

const directive = {
  beforeMount(el) {
    hljs.highlightBlock(el);
  },
};

export default directive;
