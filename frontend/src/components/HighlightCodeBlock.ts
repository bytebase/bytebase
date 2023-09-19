import hljs from "highlight.js/lib/core";
import { h, defineComponent } from "vue";

export default defineComponent({
  name: "HighlightCodeBlock",
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
      default: "pre",
    },
  },
  render() {
    const { code, language, tag } = this.$props;
    const { class: additionalClass } = this.$attrs;

    const result = hljs.highlight(code, {
      language: language,
    });

    return h(tag, {
      class: [language, additionalClass],
      innerHTML: result.value,
    });
  },
});
