import { escapeRegExp } from "lodash-es";

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
  const keywords =
    typeof keyword === "string"
      ? [keyword.trim()].filter(Boolean)
      : (keyword ?? []).map((k) => k.trim()).filter(Boolean);
  if (keywords.length === 0) {
    return <span className={className}>{text}</span>;
  }

  const pattern = keywords.map(escapeRegExp).join("|");
  const parts = text.split(new RegExp(`(${pattern})`, "gi"));
  const isMatch = new RegExp(`^(?:${pattern})$`, "i");

  return (
    <span className={className}>
      {parts.map((part, index) =>
        isMatch.test(part) ? (
          <b key={index} className="text-accent">
            {part}
          </b>
        ) : (
          part
        )
      )}
    </span>
  );
}
