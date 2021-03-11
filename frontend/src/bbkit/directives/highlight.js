import hljs from "highlight.js";
import "highlight.js/styles/github-gist.css";

const directive = {
  beforeMount(el) {
    let blocks = el.querySelectorAll("pre code");
    blocks.forEach((block) => {
      hljs.highlightBlock(block);
    });
  },
};

export default directive;
