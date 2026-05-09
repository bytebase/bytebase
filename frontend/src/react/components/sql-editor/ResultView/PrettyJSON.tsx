import { escape } from "lodash-es";
import { useEffect, useState } from "react";

interface PrettyJSONProps {
  content: string;
}

/**
 * Pretty-prints a JSON string. Heavy deps (`pretty-print-json`,
 * `lossless-json`) load lazily so that JSON-cell hover doesn't pay
 * for them eagerly. An AbortController per `content` prevents the
 * older pretty-print result from racing in after a quick switch.
 */
export function PrettyJSON({ content }: PrettyJSONProps) {
  const [html, setHtml] = useState<string>("");

  useEffect(() => {
    const controller = new AbortController();
    (async () => {
      try {
        const [{ prettyPrintJson }, { parse }, { losslessReviver }] =
          await Promise.all([
            import("pretty-print-json"),
            import("lossless-json"),
            import("@/composables/utils"),
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

  return <div dangerouslySetInnerHTML={{ __html: html }} />;
}
