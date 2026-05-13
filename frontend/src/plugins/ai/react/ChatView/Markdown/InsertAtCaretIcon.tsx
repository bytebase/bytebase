import { AlignLeftIcon, ArrowLeftIcon } from "lucide-react";

type Props = {
  readonly size?: number;
};

/**
 * React port of `plugins/ai/components/ChatView/Markdown/InsertAtCaretIcon.vue`.
 *
 * Composite icon = `AlignLeftIcon` (paragraph lines) with an accent-colored
 * `ArrowLeftIcon` overlaid in the bottom-right. Visualizes "insert this
 * snippet at the editor's caret position".
 */
export function InsertAtCaretIcon({ size = 16 }: Props) {
  return (
    <div className="inline-block relative">
      <AlignLeftIcon style={{ width: `${size}px`, height: `${size}px` }} />
      <ArrowLeftIcon
        className="text-accent absolute"
        strokeWidth={4}
        style={{
          width: `${size * 0.7}px`,
          height: `${size * 0.7}px`,
          right: `${size * -0.2}px`,
          bottom: `${size * 0.035}px`,
        }}
      />
    </div>
  );
}
