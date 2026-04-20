import logoFull from "@/assets/logo-full.svg";
import { useVueState } from "@/react/hooks/useVueState";
import { useWorkspaceV1Store } from "@/store";

type Props = {
  /** Optional route name — when set, the logo is wrapped in a link that records the visit. */
  readonly redirect?: string;
};

/**
 * Replaces frontend/src/components/BytebaseLogo.vue. Shows the workspace's
 * custom logo when set, otherwise the bundled Bytebase fallback SVG.
 *
 * The `redirect` prop is supported for future callers. It is not used by
 * the SQL Editor Welcome screen. Router-link behavior will be added when
 * the first React caller needs it.
 */
export function BytebaseLogo(_props: Props) {
  const customLogo = useVueState(
    () => useWorkspaceV1Store().currentWorkspace?.logo ?? ""
  );

  return (
    <div className="shrink-0 max-w-44 flex items-center overflow-hidden">
      <span className="h-full w-full select-none flex flex-row justify-center items-center">
        {customLogo ? (
          <img
            src={customLogo}
            alt="branding logo"
            className="h-full object-contain"
          />
        ) : (
          <img
            src={logoFull}
            alt="Bytebase"
            className="h-8 md:h-10 w-auto object-contain"
          />
        )}
      </span>
    </div>
  );
}
