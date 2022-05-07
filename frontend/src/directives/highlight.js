import hljs from "highlight.js";
import "highlight.js/styles/github-gist.css";

hljs.configure({
  languages: ["sql"],
});

const directive = {
  beforeMount(el) {
    hljs.highlightElement(el);
  },
};

export default directive;
