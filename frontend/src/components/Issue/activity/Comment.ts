import { h, defineComponent, PropType } from "vue";
import { Activity } from "@/types";

export default defineComponent({
  name: "ActivityComment",
  props: {
    activity: {
      type: Object as PropType<Activity>,
      required: true,
    },
  },
  render() {
    const { comment } = this.$props.activity;
    if (!comment) return null;

    const pattern = /(#\d+)\b/;
    const parts = comment.split(pattern);

    return parts.map((part) => {
      if (part === "") return null;

      if (part.startsWith("#")) {
        const id = parseInt(part.slice(1), 10);
        if (!Number.isNaN(id) && id > 0) {
          // we met a valid #{issue_id} in which issue_id is an integer and >= 0
          // render a link to the issue
          return h(
            "a",
            {
              href: `/issue/${id}`,
              class: "font-medium text-main whitespace-nowrap hover:underline",
            },
            part
          );
        }
      }

      return h("span", {}, part);
    });
  },
});
