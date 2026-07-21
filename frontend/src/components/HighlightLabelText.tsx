import { getHighlightHTMLByRegExp } from "@/utils/util";

interface HighlightLabelTextProps {
  text: string;
  keyword?: string | readonly string[];
  className?: string;
}

export function HighlightLabelText({
  text,
  keyword,
  className,
}: HighlightLabelTextProps) {
  const pattern: string | string[] =
    typeof keyword === "string"
      ? keyword.trim()
      : (keyword ?? []).map((k) => k.trim()).filter(Boolean);
  const isEmpty =
    typeof pattern === "string" ? pattern === "" : pattern.length === 0;
  if (isEmpty) {
    return <span className={className}>{text}</span>;
  }
  return (
    <span
      className={className}
      // getHighlightHTMLByRegExp escapes the input and sanitizes with
      // DOMPurify, so the returned string is safe to inject.
      dangerouslySetInnerHTML={{
        __html: getHighlightHTMLByRegExp(text, pattern),
      }}
    />
  );
}
