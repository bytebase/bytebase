import { type Config, type Node, type Schema, Tag } from "@markdoc/markdoc";

export const markdocConfig: { nodes: Record<string, Schema> } = {
  nodes: {
    link: {
      render: "a",
      attributes: {
        href: { type: String, required: true },
        target: { type: String, default: "_blank" },
      },
      transform(node: Node, config: Config) {
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
