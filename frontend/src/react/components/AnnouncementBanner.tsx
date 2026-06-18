import { ArrowRight } from "lucide-react";
import { cn } from "@/react/lib/utils";

interface AnnouncementBannerProps {
  /** Banner message. */
  text: string;
  /** When set, an arrow affordance is shown next to the text. */
  link?: string;
  /** Background color as an `"r g b"` triple. */
  background: string;
  /** Text color as an `"r g b"` triple. */
  textColor: string;
  /**
   * When true (default), the link navigates (`<a target="_blank">`). Pass
   * `false` for a static, non-navigating render — e.g. the settings preview.
   */
  interactive?: boolean;
  /** Extra container classes (e.g. `rounded` for a boxed preview). */
  className?: string;
}

/**
 * Presentational announcement banner shared by the real workspace banner
 * (`BannersWrapper`) and the settings theme preview, so the two stay in visual
 * sync. It only renders — callers own fetching, feature-gating, and the
 * empty-text guard.
 */
export function AnnouncementBanner({
  text,
  link,
  background,
  textColor,
  interactive = true,
  className,
}: Readonly<AnnouncementBannerProps>) {
  const linkBody = (
    <>
      <span className="px-1">{text}</span>
      <ArrowRight className="mr-3 size-5" />
    </>
  );

  // Plain text by default; with a link, navigate (real banner) or stay static
  // (preview). Kept as if/else to avoid a nested JSX ternary.
  let content = <span>{text}</span>;
  if (link) {
    content = interactive ? (
      <a
        href={link}
        target="_blank"
        rel="noreferrer"
        className="flex flex-row items-center hover:underline hover:opacity-90"
      >
        {linkBody}
      </a>
    ) : (
      <span className="flex flex-row items-center">{linkBody}</span>
    );
  }

  return (
    <div
      className={cn(
        "mx-auto flex w-full flex-row flex-wrap justify-center px-3 py-1 text-center font-medium",
        className
      )}
      style={{
        backgroundColor: `rgb(${background})`,
        color: `rgb(${textColor})`,
      }}
    >
      {content}
    </div>
  );
}
