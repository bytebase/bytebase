import { Tag } from "@markdoc/markdoc";

export const markdocConfig = {
  nodes: {
    link: {
      render: "a",
      attributes: {
        href: { type: String, required: true },
        target: { type: String, default: "_blank" },
      },
      transform(node: any, config: any) {
        const attributes = node.transformAttributes(config);
        const children = node.transformChildren(config);
        Object.assign(attributes, { target: "_blank" });
        return new Tag("a", attributes, children);
      },
    },
  },
};

export const DOMPurifyConfig = {
  ADD_ATTR: ["target"],
};
