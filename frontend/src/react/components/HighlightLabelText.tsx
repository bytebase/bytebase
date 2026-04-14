import { getHighlightHTMLByRegExp } from "@/utils";

interface HighlightLabelTextProps {
  text: string;
  keyword?: string;
  className?: string;
}

export function HighlightLabelText({
  text,
  keyword,
  className,
}: HighlightLabelTextProps) {
  const trimmed = keyword?.trim() ?? "";
  if (!trimmed) {
    return <span className={className}>{text}</span>;
  }
  return (
    <span
      className={className}
      // getHighlightHTMLByRegExp escapes the input and sanitizes with
      // DOMPurify, so the returned string is safe to inject.
      dangerouslySetInnerHTML={{
        __html: getHighlightHTMLByRegExp(text, trimmed),
      }}
    />
  );
}
