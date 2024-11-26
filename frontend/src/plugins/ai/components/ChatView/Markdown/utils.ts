import type {
  Root,
  RootContent,
  RootContentMap,
  Text,
  Definition,
  AlignType,
} from "mdast";
import { normalizeUri } from "micromark-util-sanitize-uri";
import { h, type VNode } from "vue";

export type CustomRender = {
  blockquote(node: RootContentMap["blockquote"], state?: State): VNode | string;
  break(node: RootContentMap["break"], state?: State): VNode | string;
  code(node: RootContentMap["code"], state?: State): VNode | string;
  delete(node: RootContentMap["delete"], state?: State): VNode | string;
  emphasis(node: RootContentMap["emphasis"], state?: State): VNode | string;
  footnoteDefinition(
    node: RootContentMap["footnoteDefinition"],
    state?: State
  ): VNode | string;
  footnoteReference(
    node: RootContentMap["footnoteReference"],
    state?: State
  ): VNode | string;
  heading(node: RootContentMap["heading"], state?: State): VNode | string;
  html(node: RootContentMap["html"], state?: State): VNode | string;
  image(node: RootContentMap["image"], state?: State): VNode | string;
  definition(node: RootContentMap["definition"], state?: State): VNode | string;
  imageReference(
    node: RootContentMap["imageReference"],
    state?: State
  ): VNode | string;
  inlineCode(node: RootContentMap["inlineCode"], state?: State): VNode | string;
  link(node: RootContentMap["link"], state?: State): VNode | string;
  linkReference(
    node: RootContentMap["linkReference"],
    state?: State
  ): VNode | string;
  list(node: RootContentMap["list"], state?: State): VNode | string;
  listItem(node: RootContentMap["listItem"], state?: State): VNode | string;
  paragraph(node: RootContentMap["paragraph"], state?: State): VNode | string;
  strong(node: RootContentMap["strong"], state?: State): VNode | string;
  table(node: RootContentMap["table"], state?: State): VNode | string;
  text(node: RootContentMap["text"], state?: State): VNode | string;
  thematicBreak(
    node: RootContentMap["thematicBreak"],
    state?: State
  ): VNode | string;
};

export type State = {
  slots: CustomRender;
  definitionById: Map<string, Definition>;
};

type NodeKey = keyof CustomRender;

type NodeToVNodeKey = keyof typeof mdastToVNode;

function defaultMdNodeToVNode(node: RootContent, state?: State) {
  const type = node.type as NodeKey;

  if (state?.slots[type]) {
    return state.slots[type](node as any);
  }

  const handler: (typeof mdastToVNode)[NodeToVNodeKey] =
    type in mdastToVNode
      ? mdastToVNode[type as NodeToVNodeKey]
      : mdastToVNode.unknown;

  return handler(node as any, state);
}

function rootToVNode(node: Root, state?: State): VNode | string {
  const properties = {
    class: ["markdown"],
  };

  return h(
    "div",
    properties,
    node.children.map((child) => defaultMdNodeToVNode(child, state))
  );
}

function blockquoteToVNode(
  node: RootContentMap["blockquote"],
  state?: State
): VNode | string {
  return h(
    "blockquote",
    undefined,
    node.children.map((node) => defaultMdNodeToVNode(node, state))
  );
}

function breakToVNode(): VNode | string {
  // 貌似这货还需要多一个\n的字符……？不过看起来好像没必要……
  // {type: 'text', value: '\n'}
  return h("br");
}

function codeToVNode(node: RootContentMap["code"]): VNode | string {
  const value = node.value ? node.value + "\n" : "";
  const properties: Record<string, string> = {};

  if (node.lang) {
    properties.className = `language-${node.lang}`;
  }

  const result = h("code", properties, [value]);

  return h("pre", undefined, [result]);
}

function deleteToVNode(
  node: RootContentMap["delete"],
  state?: State
): VNode | string {
  return h(
    "del",
    undefined,
    node.children.map((node) => defaultMdNodeToVNode(node, state))
  );
}

function emphasisToVNode(
  node: RootContentMap["emphasis"],
  state?: State
): VNode | string {
  return h(
    "em",
    undefined,
    node.children.map((node) => defaultMdNodeToVNode(node, state))
  );
}

function footnoteDefinitionToVNode(
  node: RootContentMap["footnoteDefinition"]
): VNode | string {
  // footnoteDefinition的children只有一个child并且为text，这里强行转一下
  const text = node.children[0] as unknown as Text;
  return h("div", { class: "footnote-definition" }, [text.value]);

  // const clobberPrefix = "user-content-";
  // const id = String(node.identifier).toUpperCase();
  // const safeId = normalizeUri(id.toLowerCase());
  // const index = state?.footnoteOrder.indexOf(id) ?? 0;
  // let counter;

  // let reuseCounter = state?.footnoteCounts.get(id);

  // if (reuseCounter === undefined) {
  //   reuseCounter = 0;
  //   state?.footnoteOrder.push(id);
  //   counter = state?.footnoteOrder.length;
  // } else {
  //   counter = index + 1;
  // }

  // reuseCounter += 1;
  // state?.footnoteCounts.set(id, reuseCounter);

  // // footnoteDefinition的children只有一个child并且为text，这里强行转一下
  // const text = node.children[0] as unknown as Text;

  // return h("span", undefined, [
  //   text.value,
  //   h(
  //     "a",
  //     {
  //       href: "#" + clobberPrefix + "fn-" + safeId,
  //       id:
  //         clobberPrefix +
  //         "fnref-" +
  //         safeId +
  //         (reuseCounter > 1 ? "-" + reuseCounter : ""),
  //       dataFootnoteRef: true,
  //       ariaDescribedBy: ["footnote-label"],
  //     },
  //     [String(counter)]
  //   ),
  // ]);
}

function footnoteReferenceToVNode(): VNode | string {
  return "";

  // const clobberPrefix = "user-content-";
  // const id = String(node.identifier).toUpperCase();
  // const safeId = normalizeUri(id.toLowerCase());
  // // TODO后面实验一下，现在LLM铁定进不到这个分支里
  // const index = state?.footnoteOrder.indexOf(id) ?? 0;
  // let counter;
  // let reuseCounter = state?.footnoteCounts.get(id);
  // if (reuseCounter === undefined) {
  //   reuseCounter = 0;
  //   state?.footnoteOrder.push(id);
  //   counter = state?.footnoteOrder.length;
  // } else {
  //   counter = index + 1;
  // }
  // reuseCounter += 1;
  // state?.footnoteCounts.set(id, reuseCounter);
  // return h("sup", undefined, [
  //   h(
  //     "a",
  //     {
  //       href: "#" + clobberPrefix + "fn-" + safeId,
  //       id:
  //         clobberPrefix +
  //         "fnref-" +
  //         safeId +
  //         (reuseCounter > 1 ? "-" + reuseCounter : ""),
  //       dataFootnoteRef: true,
  //       ariaDescribedBy: ["footnote-label"],
  //     },
  //     [String(counter)]
  //   ),
  // ]);
}

function headingToVNode(
  node: RootContentMap["heading"],
  state?: State
): VNode | string {
  const tagName = `h${node.depth}`;
  return h(
    tagName,
    undefined,
    node.children.map((node) => defaultMdNodeToVNode(node, state))
  );
}

function htmlToVNode(node: RootContentMap["html"]): VNode | string {
  return node.value;
}

type ImageProps = {
  src: string;
  alt?: string;
  title?: string;
};

function imageToVNode(node: RootContentMap["image"]): VNode | string {
  const properties: ImageProps = {
    src: normalizeUri(node.url),
  };

  if (node.alt !== null && node.alt !== undefined) {
    properties.alt = node.alt;
  }

  if (node.title !== null && node.title !== undefined) {
    properties.title = node.title;
  }

  return h("img", properties);
}

function definitionToVNode(
  node: RootContentMap["definition"],
  state?: State
): VNode | string {
  const id = String(node.identifier).toUpperCase();

  state?.definitionById.set(id, {
    url: node.url,
    type: "definition",
    identifier: node.identifier,
  });

  return "";
}

function referenceToText(
  node: RootContentMap["imageReference"] | RootContentMap["linkReference"],
  state?: State
) {
  if (node.type === "imageReference") {
    return `![${node.alt ?? ""}]`;
  }

  const children = node.children.map((node) =>
    defaultMdNodeToVNode(node, state)
  );

  const hasVnode = children.some((node) => typeof node !== "string");

  return hasVnode
    ? h("span", undefined, ["[", ...children, "]"])
    : `[${children.join("")}]`;
}

function imageReferenceToVNode(
  node: RootContentMap["imageReference"],
  state?: State
): VNode | string {
  const id = String(node.identifier).toUpperCase();
  const definition = state?.definitionById.get(id);

  if (!definition) {
    return referenceToText(node, state);
  }

  const properties: ImageProps = {
    src: normalizeUri(definition.url || ""),
  };
  if (node.alt) {
    properties.alt = node.alt;
  }

  if (definition.title !== null && definition.title !== undefined) {
    properties.title = definition.title;
  }

  return h("img", properties);
}

function inlineCodeToVNode(node: RootContentMap["inlineCode"]): VNode | string {
  const value = node.value.replace(/\r?\n|\r/g, " ");
  return h("code", undefined, [value]);
}

type LinkProps = {
  href: string;
  target: string;
  rel: string;
  title?: string;
};

function linkToVNode(
  node: RootContentMap["link"],
  state?: State
): VNode | string {
  const properties: LinkProps = {
    href: normalizeUri(node.url),
    target: "_blank",
    rel: "noreferrer nofollow noopener",
  };

  if (node.title !== null && node.title !== undefined) {
    properties.title = node.title;
  }

  return h(
    "a",
    properties,
    node.children.map((node) => defaultMdNodeToVNode(node, state))
  );
}

function linkReferenceToVNode(
  node: RootContentMap["linkReference"],
  state?: State
): VNode | string {
  const id = String(node.identifier).toUpperCase();
  const definition = state?.definitionById.get(id);

  if (!definition) {
    return referenceToText(node, state);
  }

  const properties: LinkProps = {
    href: normalizeUri(definition.url || ""),
    target: "_blank",
    rel: "noreferrer nofollow noopener",
  };

  if (definition.title !== null && definition.title !== undefined) {
    properties.title = definition.title;
  }

  return h(
    "a",
    properties,
    node.children.map((node) => defaultMdNodeToVNode(node, state))
  );
}

type ListProps = {
  start?: number;
};

function listToVNode(
  node: RootContentMap["list"],
  state?: State
): VNode | string {
  const properties: ListProps = {};

  const children = node.children.map((node) =>
    defaultMdNodeToVNode(node, state)
  );

  if (node.ordered) {
    if (typeof node.start === "number" && node.start !== 1) {
      properties.start = node.start;
    } else {
      properties.start = 1;
    }
  }

  return h(
    node.ordered ? "ol" : "ul",
    {
      style: `list-style: ${node.ordered ? "auto" : "initial"}; margin-left: 1rem;`,
      ...properties,
    },
    children
  );
}

type ListItemProps = {
  className?: string;
};

function listItemToVNode(
  node: RootContentMap["listItem"],
  state?: State
): VNode | string {
  const children = node.children.map((node) =>
    defaultMdNodeToVNode(node, state)
  );

  const properties: ListItemProps = {};

  if (typeof node.checked === "boolean") {
    const head = children[0];

    const checkedVNode = h("input", {
      type: "checkbox",
      checked: node.checked,
      disabled: true,
    });

    if (typeof head !== "string" && head?.type === "p") {
      head.children = [checkedVNode, ...((head.children as VNode[]) ?? [])];
    } else {
      children.unshift(h("p", undefined, [checkedVNode]));
    }

    // According to github-markdown-css, this class hides bullet.
    // See: <https://github.com/sindresorhus/github-markdown-css>.
    properties.className = "task-list-item";
  }

  return h("li", properties, children);
}

function paragraphToVNode(
  node: RootContentMap["paragraph"],
  state?: State
): VNode | string {
  return h(
    // 避免出现p套p的情况，尤其是各种扩展组件（比如mermaid）之类的会出现
    "div",
    { class: "paragraph" },
    node.children.map((node) => defaultMdNodeToVNode(node, state))
  );
}

function strongToVNode(
  node: RootContentMap["strong"],
  state?: State
): VNode | string {
  return h(
    "strong",
    undefined,
    node.children.map((node) => defaultMdNodeToVNode(node, state))
  );
}

function tableRowToVNode(
  node: RootContentMap["tableRow"],
  align: AlignType[],
  rowIndex: number,
  state?: State
): VNode | string {
  const tagName = rowIndex === 0 ? "th" : "td";

  const children = node.children.map((node, index) => {
    const alignInfo = align[index];
    const properties = alignInfo
      ? {
          align: alignInfo,
        }
      : undefined;

    return h(
      tagName,
      properties,
      node.children.map((node) => defaultMdNodeToVNode(node, state))
    );
  });

  return h("tr", undefined, children);
}

function tableToVNode(
  node: RootContentMap["table"],
  state?: State
): VNode | string {
  const tableHeadChildren: Array<VNode | string> = [];
  const tableContentChildren: Array<VNode | string> = [];

  const align = node.align ?? [];

  node.children.forEach((node, index) => {
    const vnode = tableRowToVNode(node, align, index, state);

    if (index === 0) {
      tableHeadChildren.push(vnode);
    } else {
      tableContentChildren.push(vnode);
    }
  });

  const children: Array<VNode | string> = [];
  if (tableHeadChildren.length > 0) {
    children.push(h("thead", undefined, tableHeadChildren));
  }

  if (tableContentChildren.length > 0) {
    children.push(h("tbody", undefined, tableContentChildren));
  }

  return h("table", undefined, children);
}

function textToVNode(node: RootContentMap["text"]): VNode | string {
  return node.value;
}

function thematicBreakToVNode(): VNode | string {
  return h("hr");
}

function defaultUnknownHandler(
  node: RootContent,
  state?: State
): VNode | string {
  if ("children" in node) {
    const properties = ("properties" in node ? node.properties : {}) as Record<
      string,
      any
    >;
    if ("className" in properties && Array.isArray(properties.className)) {
      // hast和vnode的结构不同导致的，JSX上的区别
      properties.className = properties.className.join(" ");
    }

    return h(
      "div",
      properties,
      node.children.map((node) => defaultMdNodeToVNode(node, state))
    );
  }

  if ("value" in node) {
    return node.value;
  }

  // empty node……？？？
  return "";
}

export const mdastToVNode = {
  root: rootToVNode,
  blockquote: blockquoteToVNode,
  break: breakToVNode,
  code: codeToVNode,
  delete: deleteToVNode,
  emphasis: emphasisToVNode,

  footnoteDefinition: footnoteDefinitionToVNode,
  footnoteReference: footnoteReferenceToVNode,

  heading: headingToVNode,
  html: htmlToVNode,
  image: imageToVNode,

  // TODO同上
  definition: definitionToVNode,
  imageReference: imageReferenceToVNode,

  inlineCode: inlineCodeToVNode,
  link: linkToVNode,
  linkReference: linkReferenceToVNode,
  list: listToVNode,
  listItem: listItemToVNode,
  paragraph: paragraphToVNode,
  strong: strongToVNode,
  table: tableToVNode,
  text: textToVNode,
  thematicBreak: thematicBreakToVNode,
  unknown: defaultUnknownHandler,
};
