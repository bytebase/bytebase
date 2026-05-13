import { createElement, useMemo } from "react";
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import { unified } from "unified";
import { AstToReact } from "./AstToReact";
import { CodeBlock, type CodeBlockProps } from "./CodeBlock";
import type { CustomSlots } from "./utils";

const processor = unified().use(remarkParse).use(remarkGfm);

type Props = {
  readonly content: string;
  readonly codeBlockProps: CodeBlockProps;
};

/**
 * React port of `plugins/ai/components/ChatView/Markdown/Markdown.vue`.
 *
 * Parses the message content with `remark-parse` + `remark-gfm` and
 * walks the AST via `AstToReact`. Three slots customize rendering:
 *   - `code` (block-level fenced code) → `<CodeBlock>` (interactive,
 *     SQL editor with Run/Insert/Copy)
 *   - `inlineCode` → plain styled `<code>` (the Vue version used
 *     `HighlightCodeBlock`; for parity we render the same gray pill
 *     without syntax highlighting — short snippets rarely benefit)
 *   - `image` → unstyled `<img>`
 */
export function Markdown({ content, codeBlockProps }: Props) {
  const ast = useMemo(() => processor.parse(content ?? ""), [content]);

  const slots: CustomSlots = useMemo(
    () => ({
      code: (node) =>
        createElement(CodeBlock, { code: node.value, ...codeBlockProps }),
      inlineCode: (node) =>
        createElement(
          "code",
          { className: "inline-block bg-gray-200 px-0.5 mx-0.5" },
          node.value.replace(/\r?\n|\r/g, " ")
        ),
      image: (node) => createElement("img", { src: node.url }),
    }),
    [codeBlockProps]
  );

  return <AstToReact ast={ast} slots={slots} />;
}
