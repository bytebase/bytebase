import { useEffect, useMemo, useState } from "react";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import { engineNameV1, supportedEngineV1List } from "@/utils";

interface TabsByEngineProps {
  ruleMapByEngine: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>;
  children: (ruleList: RuleTemplateV2[], engine: Engine) => React.ReactNode;
}

export function TabsByEngine({ ruleMapByEngine, children }: TabsByEngineProps) {
  const [selectedEngine, setSelectedEngine] = useState<Engine>(
    Engine.ENGINE_UNSPECIFIED
  );

  // Only reset to first engine on initial load or when the selected engine disappears
  useEffect(() => {
    if (
      selectedEngine === Engine.ENGINE_UNSPECIFIED ||
      !ruleMapByEngine.has(selectedEngine)
    ) {
      const firstEngine =
        [...ruleMapByEngine.keys()][0] ?? Engine.ENGINE_UNSPECIFIED;
      setSelectedEngine(firstEngine);
    }
  }, [ruleMapByEngine, selectedEngine]);

  const sortedData = useMemo(() => {
    const orderRank = new Map<Engine, number>();
    supportedEngineV1List().forEach((engine, index) => {
      orderRank.set(engine, index);
    });

    return [...ruleMapByEngine.entries()].sort(([e1], [e2]) => {
      return (orderRank.get(e1) ?? 0) - (orderRank.get(e2) ?? 0);
    });
  }, [ruleMapByEngine]);

  const RE_SUBTITLE = /\(.+?\)/;

  const engineParts = (engine: Engine): { title: string; subtitle: string } => {
    const name = engineNameV1(engine);
    const match = name.match(RE_SUBTITLE);
    if (!match) return { title: name, subtitle: "" };
    return {
      title: name.replace(match[0], "").trim(),
      subtitle: match[0],
    };
  };

  if (sortedData.length === 0) {
    return null;
  }

  return (
    <Tabs
      value={String(selectedEngine)}
      onValueChange={(val) => setSelectedEngine(Number(val) as Engine)}
    >
      <TabsList className="gap-x-4">
        {sortedData.map(([engine, ruleMap]) => (
          <TabsTrigger key={engine} value={String(engine)}>
            <div className="flex items-center gap-x-1">
              <img
                src={EngineIconPath[engine]}
                alt=""
                className="h-4 w-auto object-contain"
              />
              <span className="text-sm">{engineParts(engine).title}</span>
              {engineParts(engine).subtitle && (
                <span className="text-xs text-control-light">
                  {engineParts(engine).subtitle}
                </span>
              )}
              <span className="text-xs px-1 py-0.5 rounded-full bg-gray-200 text-gray-800 ml-1">
                {ruleMap.size}
              </span>
            </div>
          </TabsTrigger>
        ))}
      </TabsList>
      {sortedData.map(([engine, ruleMap]) => (
        <TabsPanel key={engine} value={String(engine)}>
          {children([...ruleMap.values()], engine)}
        </TabsPanel>
      ))}
    </Tabs>
  );
}
