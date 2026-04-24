import { useTranslation } from "react-i18next";
import logoFull from "@/assets/logo-full.svg";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useWorkspaceV1Store } from "@/store";

type Props = {
  /** Optional route name — when set, the logo is wrapped in a link that records the visit. */
  readonly redirect?: string;
  readonly className?: string;
};

/**
 * Replaces frontend/src/components/BytebaseLogo.vue. Shows the workspace's
 * custom logo when set, otherwise the bundled Bytebase fallback SVG.
 *
 */
export function BytebaseLogo({ className, redirect }: Props) {
  const { t } = useTranslation();
  const workspaceStore = useWorkspaceV1Store();
  const { record } = useRecentVisit();
  const customLogo = useVueState(
    () => workspaceStore.currentWorkspace?.logo ?? ""
  );

  const content = (
    <span className="h-full w-full select-none flex flex-row justify-center items-center">
      {customLogo ? (
        <img
          src={customLogo}
          alt={t("settings.general.workspace.logo")}
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
  );

  return (
    <div
      className={cn(
        "shrink-0 max-w-44 flex items-center overflow-hidden",
        className
      )}
    >
      {redirect ? (
        <button
          type="button"
          className="h-full w-full cursor-pointer"
          onClick={() => {
            const route = router.resolve({ name: redirect });
            record(route.fullPath);
            void router.push(route);
          }}
        >
          {content}
        </button>
      ) : (
        content
      )}
    </div>
  );
}
