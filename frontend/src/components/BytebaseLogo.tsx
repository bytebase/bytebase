import { useTranslation } from "react-i18next";
import { useNavigate } from "@/app/router";
import logoFull from "@/assets/logo-full.svg";
import logoFullDark from "@/assets/logo-full-dark.svg";
import { RouterLink } from "@/components/RouterLink";
import { useRecentVisit, useWorkspace } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";

type Props = {
  /** Optional route name — when set, the logo is wrapped in a link that records the visit. */
  readonly redirect?: string;
  readonly className?: string;
  readonly builtinTheme?: "light" | "dark";
};

/**
 * Replaces frontend/src/components/BytebaseLogo.vue. Shows the workspace's
 * custom logo when set, otherwise the bundled Bytebase fallback SVG.
 *
 */
export function BytebaseLogo({
  builtinTheme = "light",
  className,
  redirect,
}: Props) {
  const { t } = useTranslation();
  const workspace = useWorkspace();
  const { record } = useRecentVisit();
  const navigate = useNavigate();
  const customLogo = workspace?.logo ?? "";
  const builtinLogo = builtinTheme === "dark" ? logoFullDark : logoFull;

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
          src={builtinLogo}
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
        <RouterLink
          to={{ name: redirect }}
          className="h-full w-full cursor-pointer"
          onClick={() => {
            const route = navigate.resolve({ name: redirect });
            record(route.fullPath);
          }}
        >
          {content}
        </RouterLink>
      ) : (
        content
      )}
    </div>
  );
}
