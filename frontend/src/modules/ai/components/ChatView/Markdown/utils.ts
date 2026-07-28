import type {
  AlignType,
  Definition,
  Root,
  RootContent,
  RootContentMap,
  Text,
} from "mdast";
import { normalizeUri } from "micromark-util-sanitize-uri";
import type { ReactNode } from "react";
import { createElement, Fragment } from "react";

/**
 * React port of `plugins/ai/components/ChatView/Markdown/utils.ts`.
 *
 * Same mdast walker as the Vue version; `h(...)` is swapped for
 * `React.createElement(...)`. Differences from the Vue source:
 *
 * - `class` becomes `className`; HTML attributes use React's
 *   camelCased form (`htmlFor`, `tabIndex`, etc. — none of those
 *   actually appear here).
 * - `style` props use a CSS object instead of a string.
 * - Arrays of children get explicit `key`s (React warns when keys are
 *   missing; Vue auto-keys by index).
 * - `slots` is a plain mapped object (`{ code?, inlineCode?, image? }`)
 *   instead of Vue's `defineSlots<CustomRender>`.
 */

export type CustomSlotRenderer<K extends keyof RootContentMap> = (
  node: RootContentMap[K],
  state: State
) => ReactNode;

export type CustomSlots = Partial<{
  blockquote: CustomSlotRenderer<"blockquote">;
  break: CustomSlotRenderer<"break">;
  code: CustomSlotRenderer<"code">;
  delete: CustomSlotRenderer<"delete">;
  emphasis: CustomSlotRenderer<"emphasis">;
  footnoteDefinition: CustomSlotRenderer<"footnoteDefinition">;
  footnoteReference: CustomSlotRenderer<"footnoteReference">;
  heading: CustomSlotRenderer<"heading">;
  html: CustomSlotRenderer<"html">;
  image: CustomSlotRenderer<"image">;
  definition: CustomSlotRenderer<"definition">;
  imageReference: CustomSlotRenderer<"imageReference">;
  inlineCode: CustomSlotRenderer<"inlineCode">;
  link: CustomSlotRenderer<"link">;
  linkReference: CustomSlotRenderer<"linkReference">;
  list: CustomSlotRenderer<"list">;
  listItem: CustomSlotRenderer<"listItem">;
  paragraph: CustomSlotRenderer<"paragraph">;
  strong: CustomSlotRenderer<"strong">;
  table: CustomSlotRenderer<"table">;
  text: CustomSlotRenderer<"text">;
  thematicBreak: CustomSlotRenderer<"thematicBreak">;
}>;

export type State = {
  slots: CustomSlots;
  definitionById: Map<string, Definition>;
};

type GenericNodeHandler = (node: RootContent, state: State) => ReactNode;

function withKeys(children: ReactNode[]): ReactNode[] {
  return children.map((child, index) => {
    if (child === null || child === undefined || child === false) return child;
    if (typeof child === "string" || typeof child === "number") {
      // Strings can't carry keys directly — wrap in a Fragment with key
      // so React's reconciler is happy across re-renders.
      return createElement(Fragment, { key: index }, child);
    }
    // Element nodes already get an index-based key via cloneElement-by-key.
    // We rely on the caller mapping unique indices.
    return child;
  });
}

function mapChildren<T extends { type: string }>(
  children: T[],
  state: State
): ReactNode[] {
  return withKeys(
    children.map((child) => defaultMdNodeToReact(child as RootContent, state))
  );
}

function defaultMdNodeToReact(node: RootContent, state: State): ReactNode {
  const { type } = node;
  const customSlot = state.slots[type as keyof CustomSlots] as
    | GenericNodeHandler
    | undefined;
  if (customSlot) {
    return customSlot(node, state);
  }
  const handler = (mdastToReact[type as keyof typeof mdastToReact] ??
    mdastToReact.unknown) as GenericNodeHandler;
  return handler(node, state);
}

function rootToReact(node: Root, state: State): ReactNode {
  return createElement(
    "div",
    { className: "markdown" },
    mapChildren(node.children, state)
  );
}

function blockquoteToReact(
  node: RootContentMap["blockquote"],
  state: State
): ReactNode {
  return createElement("blockquote", null, mapChildren(node.children, state));
}

function breakToReact(): ReactNode {
  return createElement("br");
}

function codeToReact(node: RootContentMap["code"]): ReactNode {
  const value = node.value ? node.value + "\n" : "";
  const codeProps: { className?: string } = {};
  if (node.lang) {
    codeProps.className = `language-${node.lang}`;
  }
  return createElement("pre", null, createElement("code", codeProps, value));
}

function deleteToReact(
  node: RootContentMap["delete"],
  state: State
): ReactNode {
  return createElement("del", null, mapChildren(node.children, state));
}

function emphasisToReact(
  node: RootContentMap["emphasis"],
  state: State
): ReactNode {
  return createElement("em", null, mapChildren(node.children, state));
}

function footnoteDefinitionToReact(
  node: RootContentMap["footnoteDefinition"]
): ReactNode {
  // Same shape as the Vue source — render the first text child inside a
  // `<div class="footnote-definition">`. The richer footnote-numbering
  // path was commented out in Vue and isn't needed for LLM output.
  const text = node.children[0] as unknown as Text;
  return createElement("div", { className: "footnote-definition" }, text.value);
}

function footnoteReferenceToReact(): ReactNode {
  return null;
}

function headingToReact(
  node: RootContentMap["heading"],
  state: State
): ReactNode {
  const tagName = `h${node.depth}`;
  return createElement(tagName, null, mapChildren(node.children, state));
}

function htmlToReact(node: RootContentMap["html"]): ReactNode {
  return node.value;
}

type ImageProps = {
  src: string;
  alt?: string;
  title?: string;
};

function imageToReact(node: RootContentMap["image"]): ReactNode {
  const props: ImageProps = { src: normalizeUri(node.url) };
  if (node.alt) props.alt = node.alt;
  if (node.title) props.title = node.title;
  return createElement("img", props);
}

function definitionToReact(
  node: RootContentMap["definition"],
  state: State
): ReactNode {
  const id = String(node.identifier).toUpperCase();
  state.definitionById.set(id, {
    url: node.url,
    type: "definition",
    identifier: node.identifier,
  });
  return null;
}

function referenceToText(
  node: RootContentMap["imageReference"] | RootContentMap["linkReference"],
  state: State
): ReactNode {
  if (node.type === "imageReference") {
    return `![${node.alt ?? ""}]`;
  }
  const children = mapChildren(node.children, state);
  const hasReactNode = children.some(
    (c) => typeof c !== "string" && typeof c !== "number"
  );
  return hasReactNode
    ? createElement("span", null, "[", ...children, "]")
    : `[${children.join("")}]`;
}

function imageReferenceToReact(
  node: RootContentMap["imageReference"],
  state: State
): ReactNode {
  const id = String(node.identifier).toUpperCase();
  const definition = state.definitionById.get(id);
  if (!definition) {
    return referenceToText(node, state);
  }
  const props: ImageProps = { src: normalizeUri(definition.url || "") };
  if (node.alt) props.alt = node.alt;
  if (definition.title) props.title = definition.title;
  return createElement("img", props);
}

function inlineCodeToReact(node: RootContentMap["inlineCode"]): ReactNode {
  const value = node.value.replace(/\r?\n|\r/g, " ");
  return createElement("code", null, value);
}

type LinkProps = {
  href: string;
  target: string;
  rel: string;
  title?: string;
};

function linkToReact(node: RootContentMap["link"], state: State): ReactNode {
  const props: LinkProps = {
    href: normalizeUri(node.url),
    target: "_blank",
    rel: "noreferrer nofollow noopener",
  };
  if (node.title) props.title = node.title;
  return createElement("a", props, mapChildren(node.children, state));
}

function linkReferenceToReact(
  node: RootContentMap["linkReference"],
  state: State
): ReactNode {
  const id = String(node.identifier).toUpperCase();
  const definition = state.definitionById.get(id);
  if (!definition) {
    return referenceToText(node, state);
  }
  const props: LinkProps = {
    href: normalizeUri(definition.url || ""),
    target: "_blank",
    rel: "noreferrer nofollow noopener",
  };
  if (definition.title) props.title = definition.title;
  return createElement("a", props, mapChildren(node.children, state));
}

function listToReact(node: RootContentMap["list"], state: State): ReactNode {
  const children = mapChildren(node.children, state);
  const ordered = node.ordered === true;
  const startProp =
    ordered && typeof node.start === "number" && node.start !== 1
      ? { start: node.start }
      : ordered
        ? { start: 1 }
        : {};
  return createElement(
    ordered ? "ol" : "ul",
    {
      style: {
        listStyle: ordered ? "auto" : "initial",
        marginLeft: "1rem",
      },
      ...startProp,
    },
    children
  );
}

function listItemToReact(
  node: RootContentMap["listItem"],
  state: State
): ReactNode {
  const children = mapChildren(node.children, state);
  const liProps: { className?: string } = {};
  if (typeof node.checked === "boolean") {
    // GitHub-style task list items: prepend a disabled checkbox + use
    // `task-list-item` so github-markdown-css hides the bullet.
    children.unshift(
      createElement("input", {
        key: "task-checkbox",
        type: "checkbox",
        checked: node.checked,
        disabled: true,
        // React expects `readOnly` when a controlled `checked` is set
        // without an `onChange`; suppress the warning.
        readOnly: true,
      })
    );
    liProps.className = "task-list-item";
  }
  return createElement("li", liProps, children);
}

function paragraphToReact(
  node: RootContentMap["paragraph"],
  state: State
): ReactNode {
  // Match the Vue version's `<div class="paragraph">` (avoids `<p>` inside
  // surrounding `<p>` when host components nest markdown).
  return createElement(
    "div",
    { className: "paragraph" },
    mapChildren(node.children, state)
  );
}

function strongToReact(
  node: RootContentMap["strong"],
  state: State
): ReactNode {
  return createElement("strong", null, mapChildren(node.children, state));
}

function tableRowToReact(
  node: RootContentMap["tableRow"],
  align: AlignType[],
  rowIndex: number,
  state: State
): ReactNode {
  const tagName = rowIndex === 0 ? "th" : "td";
  const cells = node.children.map((cell, cellIndex) => {
    const cellAlign = align[cellIndex];
    const cellProps = cellAlign
      ? { key: cellIndex, align: cellAlign as string }
      : { key: cellIndex };
    return createElement(tagName, cellProps, mapChildren(cell.children, state));
  });
  return createElement("tr", null, cells);
}

function tableToReact(node: RootContentMap["table"], state: State): ReactNode {
  const align = node.align ?? [];
  const headCells: ReactNode[] = [];
  const bodyRows: ReactNode[] = [];
  node.children.forEach((row, rowIndex) => {
    const tr = tableRowToReact(row, align, rowIndex, state);
    if (rowIndex === 0) headCells.push(tr);
    else bodyRows.push(tr);
  });
  const sections: ReactNode[] = [];
  if (headCells.length > 0) {
    sections.push(createElement("thead", { key: "thead" }, headCells));
  }
  if (bodyRows.length > 0) {
    sections.push(createElement("tbody", { key: "tbody" }, bodyRows));
  }
  return createElement("table", null, sections);
}

function textToReact(node: RootContentMap["text"]): ReactNode {
  return node.value;
}

function thematicBreakToReact(): ReactNode {
  return createElement("hr");
}

function defaultUnknownHandler(node: RootContent, state: State): ReactNode {
  if ("children" in node) {
    const childrenArray = (node as { children: RootContent[] }).children;
    const props: { className?: string } & Record<string, unknown> = {};
    if ("properties" in node) {
      const properties = (node as { properties?: Record<string, unknown> })
        .properties;
      if (properties && typeof properties === "object") {
        Object.assign(props, properties);
        if (
          "className" in properties &&
          Array.isArray((properties as Record<string, unknown>).className)
        ) {
          props.className = (
            (properties as Record<string, unknown>).className as unknown[]
          ).join(" ");
        }
      }
    }
    return createElement("div", props, mapChildren(childrenArray, state));
  }
  if ("value" in node) {
    return (node as { value: unknown }).value as ReactNode;
  }
  return null;
}

export const mdastToReact = {
  root: rootToReact,
  blockquote: blockquoteToReact,
  break: breakToReact,
  code: codeToReact,
  delete: deleteToReact,
  emphasis: emphasisToReact,
  footnoteDefinition: footnoteDefinitionToReact,
  footnoteReference: footnoteReferenceToReact,
  heading: headingToReact,
  html: htmlToReact,
  image: imageToReact,
  definition: definitionToReact,
  imageReference: imageReferenceToReact,
  inlineCode: inlineCodeToReact,
  link: linkToReact,
  linkReference: linkReferenceToReact,
  list: listToReact,
  listItem: listItemToReact,
  paragraph: paragraphToReact,
  strong: strongToReact,
  table: tableToReact,
  text: textToReact,
  thematicBreak: thematicBreakToReact,
  unknown: defaultUnknownHandler,
};
