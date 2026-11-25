import { useTranslation } from "react-i18next";
import { ExternalUrlAlert } from "@/components/ExternalUrlAlert";
import { MCPAccessPolicySection } from "./mcp/MCPAccessPolicySection";
import { MCPConnectSection } from "./mcp/MCPConnectSection";

export function MCPPage() {
  const { t } = useTranslation();

  return (
    <div className="w-full px-4 flex flex-col gap-y-6 py-4">
      <div>
        <h2 className="text-lg font-medium">{t("settings.mcp.title")}</h2>
        <p className="textinfolabel mt-1">{t("settings.mcp.description")}</p>
      </div>

      <ExternalUrlAlert actionAppearance="outline" />

      <MCPAccessPolicySection />

      <MCPConnectSection />
    </div>
  );
}
