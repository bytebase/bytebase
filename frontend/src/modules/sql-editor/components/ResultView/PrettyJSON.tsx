import { escape } from "lodash-es";
import { useEffect, useLayoutEffect, useState } from "react";
import { highlightHtmlText } from "./detail-panel-search";

interface PrettyJSONProps {
  content: string;
  searchQuery?: string;
  activeMatchIndex?: number;
  onMatchCountChange?: (count: number) => void;
  onHighlightedContentChange?: () => void;
}

/**
 * Pretty-prints a JSON string. Heavy deps (`pretty-print-json`,
 * `lossless-json`) load lazily so that JSON-cell hover doesn't pay
 * for them eagerly. An AbortController per `content` prevents the
 * older pretty-print result from racing in after a quick switch.
 */
export function PrettyJSON({
  content,
  searchQuery = "",
  activeMatchIndex = 0,
  onMatchCountChange,
  onHighlightedContentChange,
}: Readonly<PrettyJSONProps>) {
  const [html, setHtml] = useState<string>("");

  useEffect(() => {
    const controller = new AbortController();
    (async () => {
      try {
        const [{ prettyPrintJson }, { parse }, { losslessReviver }] =
          await Promise.all([
            import("pretty-print-json"),
            import("lossless-json"),
            import("@/utils/sqlResult"),
          ]);
        await import("./pretty-print-json.css");
        if (controller.signal.aborted) return;
        const obj = parse(content, null, losslessReviver);
        const next = prettyPrintJson.toHtml(obj, {
          quoteKeys: true,
          trailingCommas: false,
        });
        if (!controller.signal.aborted) setHtml(next);
      } catch (err) {
        console.warn("[PrettyJSON]", err);
        if (!controller.signal.aborted) setHtml(escape(content));
      }
    })();
    return () => controller.abort();
  }, [content]);

  const highlighted = highlightHtmlText(html, searchQuery, activeMatchIndex);

  useEffect(() => {
    onMatchCountChange?.(highlighted.count);
  }, [highlighted.count, onMatchCountChange]);

  useLayoutEffect(() => {
    onHighlightedContentChange?.();
  }, [highlighted.html, onHighlightedContentChange]);

  return <div dangerouslySetInnerHTML={{ __html: highlighted.html }} />;
}
