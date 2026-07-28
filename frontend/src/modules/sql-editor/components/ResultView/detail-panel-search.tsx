import type { ReactNode } from "react";
import { Fragment } from "react";

export const DETAIL_SEARCH_ACTIVE_MATCH_SELECTOR =
  "[data-detail-search-active-match='true']";

const matchClassName =
  "rounded-[2px] px-0.5 bg-warning-bg text-main data-[detail-search-active-match=true]:bg-accent data-[detail-search-active-match=true]:text-accent-text";

export const normalizeSearchQuery = (query: string) => query.trim();

export const findTextMatches = (content: string, query: string) => {
  const normalizedQuery = normalizeSearchQuery(query);
  if (!normalizedQuery) {
    return [];
  }

  const matches: Array<{ start: number; end: number }> = [];
  const lowerContent = content.toLocaleLowerCase();
  const lowerQuery = normalizedQuery.toLocaleLowerCase();
  let index = lowerContent.indexOf(lowerQuery);
  while (index >= 0) {
    matches.push({ start: index, end: index + normalizedQuery.length });
    index = lowerContent.indexOf(lowerQuery, index + normalizedQuery.length);
  }
  return matches;
};

export function renderTextWithSearchMatches(
  content: string,
  query: string,
  activeIndex: number
): { nodes: ReactNode; count: number } {
  const matches = findTextMatches(content, query);
  if (matches.length === 0) {
    return { nodes: content, count: 0 };
  }

  const nodes: ReactNode[] = [];
  let cursor = 0;
  for (const [index, match] of matches.entries()) {
    if (match.start > cursor) {
      nodes.push(
        <Fragment key={`text-${cursor}`}>
          {content.slice(cursor, match.start)}
        </Fragment>
      );
    }
    nodes.push(
      <mark
        key={`match-${match.start}`}
        data-detail-search-match
        data-detail-search-active-match={
          index === activeIndex ? "true" : "false"
        }
        className={matchClassName}
      >
        {content.slice(match.start, match.end)}
      </mark>
    );
    cursor = match.end;
  }
  if (cursor < content.length) {
    nodes.push(
      <Fragment key={`text-${cursor}`}>{content.slice(cursor)}</Fragment>
    );
  }

  return { nodes, count: matches.length };
}

const getTextNodes = (root: Node) => {
  const nodes: Text[] = [];
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  let current = walker.nextNode();
  while (current) {
    nodes.push(current as Text);
    current = walker.nextNode();
  }
  return nodes;
};

export function highlightHtmlText(
  html: string,
  query: string,
  activeIndex: number
): { html: string; count: number } {
  const normalizedQuery = normalizeSearchQuery(query);
  if (!normalizedQuery) {
    return { html, count: 0 };
  }

  const container = document.createElement("div");
  container.innerHTML = html;
  const textNodes = getTextNodes(container);
  const text = textNodes.map((node) => node.data).join("");
  const matches = findTextMatches(text, normalizedQuery);
  if (matches.length === 0) {
    return { html, count: 0 };
  }

  const textNodeRanges = [];
  let offset = 0;
  for (const node of textNodes) {
    textNodeRanges.push({
      node,
      start: offset,
      end: offset + node.data.length,
    });
    offset += node.data.length;
  }

  for (let matchIndex = matches.length - 1; matchIndex >= 0; matchIndex--) {
    const match = matches[matchIndex];
    const affectedRanges = textNodeRanges.filter(
      (range) => range.start < match.end && range.end > match.start
    );
    for (let index = affectedRanges.length - 1; index >= 0; index--) {
      const range = affectedRanges[index];
      const start = Math.max(match.start, range.start) - range.start;
      const end = Math.min(match.end, range.end) - range.start;
      const mark = document.createElement("mark");
      mark.dataset.detailSearchMatch = "";
      mark.dataset.detailSearchActiveMatch =
        matchIndex === activeIndex && index === 0 ? "true" : "false";
      mark.className = matchClassName;
      const highlighted = range.node.splitText(start);
      highlighted.splitText(end - start);
      mark.textContent = highlighted.data;
      highlighted.replaceWith(mark);
    }
  }

  return { html: container.innerHTML, count: matches.length };
}

export function searchMatchCountLabel(activeIndex: number, count: number) {
  if (count === 0) {
    return "0 / 0";
  }
  return `${activeIndex + 1} / ${count}`;
}
