import { h, defineComponent } from "vue";
import hljs from "highlight.js/lib/core";

export default defineComponent({
  name: "CodeHighlight",
  props: {
    code: {
      type: String,
      required: true,
    },
    language: {
      type: String,
      default: "sql",
    },
    tag: {
      type: String,
      default: "div",
    },
  },
  render() {
    const { code, language, tag } = this.$props;

    const result = hljs.highlight(code, {
      language: language,
    });
    result.language;

    return h(tag, {
      class: language,
      innerHTML: result.value,
    });
  },
});
