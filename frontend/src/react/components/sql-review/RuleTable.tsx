import { ChevronDown, ChevronRight, ExternalLink, Pencil } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getRuleKey } from "@/components/SQLReview/components/utils";
import { t as vueT } from "@/plugins/i18n";
import { Button } from "@/react/components/ui/button";
import { SearchInput } from "@/react/components/ui/search-input";
import { Tabs, TabsList, TabsTrigger } from "@/react/components/ui/tabs";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import {
  convertToCategoryMap,
  getRuleLocalization,
  isBuiltinRule,
  ruleTypeToString,
} from "@/types/sqlReview";
import {
  RuleConfig,
  RuleEditDialog,
  RuleLevelFilter,
  RuleLevelSwitch,
  type RuleListWithCategory,
} from "./RuleComponents";

// ---- Types ----

export interface SQLRuleFilterParams {
  checkedLevel: Set<SQLReviewRule_Level>;
  selectedCategory: string;
  searchText: string;
}

// ---- useSQLRuleFilter hook ----

export function useSQLRuleFilter() {
  const [checkedLevel, setCheckedLevel] = useState<Set<SQLReviewRule_Level>>(
    () => new Set()
  );
  const [selectedCategory, setSelectedCategory] = useState("all");
  const [searchText, setSearchText] = useState("");

  const params: SQLRuleFilterParams = {
    checkedLevel,
    selectedCategory,
    searchText,
  };

  const events = useMemo(
    () => ({
      toggleCheckedLevel(level: SQLReviewRule_Level) {
        setCheckedLevel((prev) => {
          const next = new Set(prev);
          if (next.has(level)) {
            next.delete(level);
          } else {
            next.add(level);
          }
          return next;
        });
      },
      changeCategory(category: string) {
        setSelectedCategory(category);
      },
      changeSearchText(keyword: string) {
        setSearchText(keyword);
      },
      reset() {
        setSelectedCategory("all");
        setCheckedLevel(new Set());
        setSearchText("");
      },
    }),
    []
  );

  return { params, events };
}

// ---- Filtering helpers ----

function filterRuleByKeyword(rule: RuleTemplateV2, searchText: string) {
  const keyword = searchText.trim().toLowerCase();
  if (!keyword) return true;
  if (ruleTypeToString(rule.type).toLowerCase().includes(keyword)) return true;
  const localization = getRuleLocalization(
    ruleTypeToString(rule.type),
    rule.engine
  );
  if (localization.title.toLowerCase().includes(keyword)) return true;
  if (localization.description.toLowerCase().includes(keyword)) return true;
  return false;
}

function filterRuleList(
  list: RuleListWithCategory[],
  params: SQLRuleFilterParams
): RuleListWithCategory[] {
  if (params.checkedLevel.size === 0 && !params.searchText) {
    return list;
  }
  return list
    .map((item) => ({
      ...item,
      ruleList: item.ruleList.filter((rule) => {
        if (
          params.checkedLevel.size > 0 &&
          !params.checkedLevel.has(rule.level)
        ) {
          return false;
        }
        return filterRuleByKeyword(rule, params.searchText);
      }),
    }))
    .filter((item) => item.ruleList.length > 0);
}

// ---- RuleFilter ----

interface RuleFilterProps {
  ruleList: RuleTemplateV2[];
  params: SQLRuleFilterParams;
  hideLevelFilter?: boolean;
  supportSelect?: boolean;
  selectedRuleCount?: number;
  onToggleSelectAll?: (select: boolean) => void;
  onToggleCheckedLevel: (level: SQLReviewRule_Level) => void;
  onChangeCategory: (category: string) => void;
  onChangeSearchText: (keyword: string) => void;
  children: (filteredRuleList: RuleListWithCategory[]) => React.ReactNode;
}

export function RuleFilter({
  ruleList,
  params,
  hideLevelFilter = false,
  supportSelect = false,
  selectedRuleCount = 0,
  onToggleSelectAll,
  onToggleCheckedLevel,
  onChangeCategory,
  onChangeSearchText,
  children,
}: RuleFilterProps) {
  const { t } = useTranslation();

  const tabItemList = useMemo(() => {
    const list: RuleListWithCategory[] = [
      { value: "all", label: vueT("common.all"), ruleList: [] },
    ];
    for (const [category, rules] of convertToCategoryMap(ruleList).entries()) {
      list.push({
        value: category,
        label: vueT(`sql-review.category.${category.toLowerCase()}`),
        ruleList: rules,
      });
    }
    return list;
  }, [ruleList]);

  const currentRuleListByCategory = useMemo(() => {
    if (params.selectedCategory === "all") {
      return tabItemList.slice(1);
    }
    const found = tabItemList.find(
      (item) => item.value === params.selectedCategory
    );
    return found ? [found] : [];
  }, [tabItemList, params.selectedCategory]);

  const filteredRuleList = useMemo(
    () => filterRuleList(currentRuleListByCategory, params),
    [currentRuleListByCategory, params]
  );

  return (
    <div className="gap-y-3">
      <Tabs
        value={params.selectedCategory}
        onValueChange={(val) => onChangeCategory(val as string)}
      >
        <TabsList className="flex-wrap">
          {tabItemList.map((item) => (
            <TabsTrigger key={item.value} value={item.value}>
              {item.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <div className="mt-2 flex flex-col justify-start items-start md:flex-row md:items-center md:justify-between">
          {!hideLevelFilter && (
            <RuleLevelFilter
              ruleList={currentRuleListByCategory}
              isCheckedLevel={(level) => params.checkedLevel.has(level)}
              onToggleCheckedLevel={onToggleCheckedLevel}
            />
          )}
          {supportSelect && (
            <div className="flex items-center gap-x-2">
              <input
                type="checkbox"
                checked={
                  selectedRuleCount === ruleList.length && ruleList.length > 0
                }
                ref={(el) => {
                  if (el) {
                    el.indeterminate =
                      selectedRuleCount > 0 &&
                      selectedRuleCount !== ruleList.length;
                  }
                }}
                onChange={(e) => onToggleSelectAll?.(e.target.checked)}
              />
              <span className="text-xl text-main font-medium">
                {vueT("sql-review.select-all")}
              </span>
            </div>
          )}
          <div className="ml-auto mt-2 md:mt-0 md:max-w-72 w-full md:w-auto">
            <SearchInput
              placeholder={t("common.filter-by-name")}
              value={params.searchText}
              onChange={(e) => onChangeSearchText(e.target.value)}
            />
          </div>
        </div>

        {/* We render the content outside TabsPanel since we control filtering ourselves */}
        <hr className="my-3 border-control-border" />
        {children(filteredRuleList)}
      </Tabs>
    </div>
  );
}

// ---- RuleTable ----

interface RuleTableProps {
  ruleList: RuleListWithCategory[];
  editable: boolean;
  supportSelect?: boolean;
  hideLevel?: boolean;
  selectedRuleKeys?: string[];
  size?: "small" | "medium";
  onRuleUpsert?: (
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ) => void;
  onRuleRemove?: (rule: RuleTemplateV2) => void;
  onSelectedRuleKeysChange?: (keys: string[]) => void;
}

export function RuleTable({
  ruleList,
  editable,
  supportSelect = false,
  hideLevel = false,
  selectedRuleKeys = [],
  size: _size = "medium",
  onRuleUpsert,
  onRuleRemove,
  onSelectedRuleKeysChange,
}: RuleTableProps) {
  const { t } = useTranslation();
  const [expandedRows, setExpandedRows] = useState<Set<string>>(
    () => new Set()
  );
  const [activeRule, setActiveRule] = useState<RuleTemplateV2 | undefined>();

  const toggleExpand = (key: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const toggleRule = (rule: RuleTemplateV2) => {
    const key = getRuleKey(rule);
    const index = selectedRuleKeys.indexOf(key);
    if (index < 0) {
      onSelectedRuleKeysChange?.([...selectedRuleKeys, key]);
    } else {
      onSelectedRuleKeysChange?.([
        ...selectedRuleKeys.slice(0, index),
        ...selectedRuleKeys.slice(index + 1),
      ]);
    }
  };

  const updateLevel = (rule: RuleTemplateV2, level: SQLReviewRule_Level) => {
    onRuleUpsert?.(rule, { level });
  };

  const onRuleChanged = (update: Partial<RuleTemplateV2>) => {
    if (activeRule) {
      onRuleUpsert?.(activeRule, update);
    }
  };

  const isExpandable = (rule: RuleTemplateV2) => {
    const loc = getRuleLocalization(ruleTypeToString(rule.type), rule.engine);
    return !!(loc.description || rule.componentList.length > 0);
  };

  return (
    <div>
      {ruleList.map((category) => (
        <div key={category.value}>
          <div className="flex my-3 items-center">
            <span className="text-xl text-main font-semibold">
              {category.label}
            </span>
            <span className="text-control-light text-md ml-1">
              ({category.ruleList.length})
            </span>
          </div>

          {/* Desktop table */}
          <div className="hidden lg:block border rounded-sm overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-control-bg">
                  <th className="w-8 px-3 py-2" />
                  {supportSelect && <th className="w-8 px-3 py-2" />}
                  <th className="text-left px-4 py-2 font-medium whitespace-nowrap">
                    {t("common.name")}
                  </th>
                  {!hideLevel && (
                    <th className="text-left px-4 py-2 font-medium whitespace-nowrap w-[12rem]">
                      {vueT("sql-review.level.name")}
                    </th>
                  )}
                  {!supportSelect && (
                    <th className="text-right px-4 py-2 font-medium whitespace-nowrap w-48">
                      {t("common.operations")}
                    </th>
                  )}
                </tr>
              </thead>
              <tbody>
                {category.ruleList.map((rule, idx) => {
                  const key = getRuleKey(rule);
                  const loc = getRuleLocalization(
                    ruleTypeToString(rule.type),
                    rule.engine
                  );
                  const expanded = expandedRows.has(key);
                  const expandable = isExpandable(rule);

                  return (
                    <RuleTableRow
                      key={key}
                      rule={rule}
                      loc={loc}
                      expanded={expanded}
                      expandable={expandable}
                      striped={idx % 2 === 1}
                      supportSelect={supportSelect}
                      hideLevel={hideLevel}
                      editable={editable}
                      isSelected={selectedRuleKeys.includes(key)}
                      onToggleExpand={() => toggleExpand(key)}
                      onToggleRule={() => toggleRule(rule)}
                      onEdit={() => setActiveRule(rule)}
                      onRemove={() => onRuleRemove?.(rule)}
                      onLevelChange={(level) => updateLevel(rule, level)}
                    />
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="flex flex-col lg:hidden border px-2 pb-4 divide-y divide-block-border">
            {category.ruleList.map((rule) => {
              const loc = getRuleLocalization(
                ruleTypeToString(rule.type),
                rule.engine
              );
              return (
                <div
                  key={getRuleKey(rule)}
                  className="pt-4 flex flex-col gap-y-3"
                >
                  <div className="flex justify-between items-center gap-x-2">
                    <div className="flex items-center gap-x-1">
                      <span>
                        {loc.title}
                        <a
                          href={`https://docs.bytebase.com/sql-review/review-rules#${rule.type}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-block ml-1"
                        >
                          <ExternalLink className="w-4 h-4" />
                        </a>
                      </span>
                    </div>
                    {supportSelect ? (
                      <input
                        type="checkbox"
                        checked={selectedRuleKeys.includes(getRuleKey(rule))}
                        onChange={() => toggleRule(rule)}
                      />
                    ) : (
                      <div className="flex items-center gap-x-2">
                        {editable && (
                          <Pencil
                            className="w-4 h-4 cursor-pointer hover:text-accent"
                            onClick={() => setActiveRule(rule)}
                          />
                        )}
                        {editable && !isBuiltinRule(rule) && (
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => onRuleRemove?.(rule)}
                          >
                            {t("common.delete")}
                          </Button>
                        )}
                      </div>
                    )}
                  </div>
                  {!hideLevel && (
                    <RuleLevelSwitch
                      level={rule.level}
                      disabled={!editable}
                      onLevelChange={(level) => updateLevel(rule, level)}
                    />
                  )}
                  <p className="text-sm text-control-placeholder">
                    {loc.description}
                  </p>
                </div>
              );
            })}
          </div>
        </div>
      ))}

      {activeRule && (
        <RuleEditDialog
          key={getRuleKey(activeRule)}
          rule={activeRule}
          disabled={!editable}
          onCancel={() => setActiveRule(undefined)}
          onUpdateRule={onRuleChanged}
        />
      )}
    </div>
  );
}

// ---- RuleTableRow (desktop, internal) ----

interface RuleTableRowProps {
  rule: RuleTemplateV2;
  loc: { title: string; description: string };
  expanded: boolean;
  expandable: boolean;
  striped: boolean;
  supportSelect: boolean;
  hideLevel: boolean;
  editable: boolean;
  isSelected: boolean;
  onToggleExpand: () => void;
  onToggleRule: () => void;
  onEdit: () => void;
  onRemove: () => void;
  onLevelChange: (level: SQLReviewRule_Level) => void;
}

function RuleTableRow({
  rule,
  loc,
  expanded,
  expandable,
  striped,
  supportSelect,
  hideLevel,
  editable,
  isSelected,
  onToggleExpand,
  onToggleRule,
  onEdit,
  onRemove,
  onLevelChange,
}: RuleTableRowProps) {
  const { t } = useTranslation();

  const colSpan =
    2 + (supportSelect ? 1 : 0) + (hideLevel ? 0 : 1) + (supportSelect ? 0 : 1);

  return (
    <>
      <tr
        className={`border-b last:border-b-0 ${striped ? "bg-gray-50" : ""} ${supportSelect ? "cursor-pointer hover:bg-gray-50" : ""}`}
        onClick={supportSelect ? onToggleRule : undefined}
      >
        <td className="px-3 py-2 w-8" onClick={(e) => e.stopPropagation()}>
          {expandable && (
            <button
              type="button"
              className="cursor-pointer p-0.5 text-control-light hover:text-control"
              onClick={onToggleExpand}
            >
              {expanded ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
            </button>
          )}
        </td>
        {supportSelect && (
          <td className="px-3 py-2 w-8" onClick={(e) => e.stopPropagation()}>
            <input
              type="checkbox"
              checked={isSelected}
              onChange={onToggleRule}
              className="h-4 w-4 rounded-xs border-control-border accent-accent"
            />
          </td>
        )}
        <td className="px-4 py-2">
          <div className="flex items-center gap-x-2">
            <span>{loc.title}</span>
            <a
              href={`https://docs.bytebase.com/sql-review/review-rules#${rule.type}`}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center text-control-light hover:text-control"
            >
              <ExternalLink className="w-4 h-4" />
            </a>
          </div>
        </td>
        {!hideLevel && (
          <td className="px-4 py-2" onClick={(e) => e.stopPropagation()}>
            <RuleLevelSwitch
              level={rule.level}
              disabled={!editable}
              onLevelChange={onLevelChange}
            />
          </td>
        )}
        {!supportSelect && (
          <td className="px-4 py-2" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-end gap-x-2">
              <Button variant="outline" size="sm" onClick={onEdit}>
                {editable ? t("common.edit") : t("common.view")}
              </Button>
              {editable && !isBuiltinRule(rule) && (
                <Button variant="destructive" size="sm" onClick={onRemove}>
                  {t("common.delete")}
                </Button>
              )}
            </div>
          </td>
        )}
      </tr>
      {expanded && (
        <tr className="border-b last:border-b-0">
          <td colSpan={colSpan} className="px-10 py-3 bg-gray-50/50">
            {loc.description && (
              <p className="text-gray-500">{loc.description}</p>
            )}
            {rule.componentList.length > 0 && loc.description && (
              <hr className="my-4 border-control-border" />
            )}
            {rule.componentList.length > 0 && (
              <RuleConfig rule={rule} disabled size="small" />
            )}
          </td>
        </tr>
      )}
    </>
  );
}

// ---- RuleTableWithFilter ----

interface RuleTableWithFilterProps {
  engine: Engine;
  ruleList: RuleTemplateV2[];
  editable: boolean;
  hideLevel?: boolean;
  supportSelect?: boolean;
  selectedRuleKeys?: string[];
  size?: "small" | "medium";
  onRuleUpsert?: (
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ) => void;
  onRuleRemove?: (rule: RuleTemplateV2) => void;
  onSelectedRuleKeysChange?: (keys: string[]) => void;
}

export function RuleTableWithFilter({
  engine,
  ruleList,
  editable,
  hideLevel = false,
  supportSelect = false,
  selectedRuleKeys = [],
  size = "medium",
  onRuleUpsert,
  onRuleRemove,
  onSelectedRuleKeysChange,
}: RuleTableWithFilterProps) {
  const { t } = useTranslation();
  const { params, events } = useSQLRuleFilter();

  useEffect(() => {
    events.reset();
  }, [engine, events]);

  const toggleSelectAll = useCallback(
    (select: boolean) => {
      if (!select) {
        onSelectedRuleKeysChange?.([]);
      } else {
        onSelectedRuleKeysChange?.(ruleList.map(getRuleKey));
      }
    },
    [ruleList, onSelectedRuleKeysChange]
  );

  return (
    <RuleFilter
      ruleList={ruleList}
      params={params}
      hideLevelFilter={hideLevel}
      supportSelect={supportSelect}
      selectedRuleCount={selectedRuleKeys.length}
      onToggleSelectAll={toggleSelectAll}
      onToggleCheckedLevel={events.toggleCheckedLevel}
      onChangeCategory={events.changeCategory}
      onChangeSearchText={events.changeSearchText}
    >
      {(filteredRuleList) =>
        filteredRuleList.length > 0 ? (
          <RuleTable
            ruleList={filteredRuleList}
            editable={editable}
            hideLevel={hideLevel}
            supportSelect={supportSelect}
            selectedRuleKeys={selectedRuleKeys}
            size={size}
            onRuleUpsert={onRuleUpsert}
            onRuleRemove={onRuleRemove}
            onSelectedRuleKeysChange={onSelectedRuleKeysChange}
          />
        ) : (
          <div className="py-12 border rounded-sm text-center text-control-placeholder">
            {t("common.no-data")}
          </div>
        )
      }
    </RuleFilter>
  );
}
