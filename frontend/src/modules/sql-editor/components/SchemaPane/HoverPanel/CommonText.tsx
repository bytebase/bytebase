type Props = {
  readonly content: string;
};

/** Replaces `HoverPanel/CommonText.vue`. Wraps long text to ~18rem. */
export function CommonText({ content }: Props) {
  return (
    <div className="max-w-[18rem] whitespace-pre-wrap break-all wrap-break-word">
      {content}
    </div>
  );
}
