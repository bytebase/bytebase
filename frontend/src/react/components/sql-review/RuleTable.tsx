import { ChevronDown, ChevronRight, ExternalLink, Pencil } from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { SearchInput } from "@/react/components/ui/search-input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { Tabs, TabsList, TabsTrigger } from "@/react/components/ui/tabs";
import { getRuleKey } from "@/react/lib/sql-review/utils";
import { cn } from "@/react/lib/utils";
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
      { value: "all", label: t("common.all"), ruleList: [] },
    ];
    for (const [category, rules] of convertToCategoryMap(ruleList).entries()) {
      list.push({
        value: category,
        label: t(`sql-review.category.${category.toLowerCase()}`),
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
        <TabsList className="flex-wrap border-b-0!">
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
              <Checkbox
                checked={
                  selectedRuleCount === ruleList.length && ruleList.length > 0
                    ? true
                    : selectedRuleCount > 0 &&
                        selectedRuleCount !== ruleList.length
                      ? "indeterminate"
                      : false
                }
                onCheckedChange={(checked) => onToggleSelectAll?.(checked)}
              />
              <span className="text-xl text-main font-medium">
                {t("sql-review.select-all")}
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
  focusRuleKey?: string;
  focusRuleSignal?: number;
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
  focusRuleKey,
  focusRuleSignal,
}: RuleTableProps) {
  const { t } = useTranslation();
  const [highlightedRuleKey, setHighlightedRuleKey] = useState<
    string | undefined
  >();

  useEffect(() => {
    if (!focusRuleKey) return;
    setHighlightedRuleKey(focusRuleKey);
    window.requestAnimationFrame(() => {
      document
        .querySelector(`[data-sql-review-rule-key="${focusRuleKey}"]`)
        ?.scrollIntoView({ block: "center", behavior: "smooth" });
    });
    const timer = window.setTimeout(() => {
      setHighlightedRuleKey(undefined);
    }, 4000);
    return () => window.clearTimeout(timer);
  }, [focusRuleKey, focusRuleSignal]);

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
            <Table>
              <TableHeader>
                <TableRow className="bg-control-bg">
                  <TableHead className="w-8" />
                  {supportSelect && <TableHead className="w-8" />}
                  <TableHead className="whitespace-nowrap">
                    {t("common.name")}
                  </TableHead>
                  {!hideLevel && (
                    <TableHead className="whitespace-nowrap w-[12rem]">
                      {t("sql-review.level.name")}
                    </TableHead>
                  )}
                  {!supportSelect && (
                    <TableHead className="text-right whitespace-nowrap w-48">
                      {t("common.operations")}
                    </TableHead>
                  )}
                </TableRow>
              </TableHeader>
              <TableBody>
                {category.ruleList.map((rule) => {
                  const key = getRuleKey(rule);
                  const highlighted = highlightedRuleKey === key;

                  return (
                    <MemoizedRuleTableRow
                      key={key}
                      ruleKey={key}
                      rule={rule}
                      highlighted={highlighted}
                      focusRuleSignal={
                        focusRuleKey === key ? focusRuleSignal : undefined
                      }
                      supportSelect={supportSelect}
                      hideLevel={hideLevel}
                      editable={editable}
                      isSelected={selectedRuleKeys.includes(key)}
                      onToggleRule={() => toggleRule(rule)}
                      onRuleUpsert={onRuleUpsert}
                      onRuleRemove={onRuleRemove}
                    />
                  );
                })}
              </TableBody>
            </Table>
          </div>

          {/* Mobile cards */}
          <div className="flex flex-col lg:hidden border px-2 pb-4 divide-y divide-block-border">
            {category.ruleList.map((rule) => {
              const key = getRuleKey(rule);
              const highlighted = highlightedRuleKey === key;
              return (
                <MobileRuleTableRow
                  key={key}
                  ruleKey={key}
                  rule={rule}
                  highlighted={highlighted}
                  supportSelect={supportSelect}
                  hideLevel={hideLevel}
                  editable={editable}
                  isSelected={selectedRuleKeys.includes(key)}
                  onToggleRule={() => toggleRule(rule)}
                  onRuleUpsert={onRuleUpsert}
                  onRuleRemove={onRuleRemove}
                />
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

// ---- RuleTableRow (desktop, internal) ----

interface RuleTableRowProps {
  ruleKey: string;
  rule: RuleTemplateV2;
  highlighted: boolean;
  focusRuleSignal?: number;
  supportSelect: boolean;
  hideLevel: boolean;
  editable: boolean;
  isSelected: boolean;
  onToggleRule: () => void;
  onRuleUpsert?: (
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ) => void;
  onRuleRemove?: (rule: RuleTemplateV2) => void;
}

function RuleTableRow({
  ruleKey,
  rule,
  highlighted,
  focusRuleSignal,
  supportSelect,
  hideLevel,
  editable,
  isSelected,
  onToggleRule,
  onRuleUpsert,
  onRuleRemove,
}: RuleTableRowProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const [editing, setEditing] = useState(false);
  const loc = useMemo(
    () => getRuleLocalization(ruleTypeToString(rule.type), rule.engine),
    [rule.engine, rule.type]
  );
  const expandable = !!(loc.description || rule.componentList.length > 0);

  const colSpan =
    2 + (supportSelect ? 1 : 0) + (hideLevel ? 0 : 1) + (supportSelect ? 0 : 1);

  const updateLevel = (level: SQLReviewRule_Level) => {
    onRuleUpsert?.(rule, { level });
  };

  const onRuleChanged = (update: Partial<RuleTemplateV2>) => {
    onRuleUpsert?.(rule, update);
  };

  useEffect(() => {
    if (focusRuleSignal === undefined) {
      return;
    }
    setExpanded(true);
  }, [focusRuleSignal]);

  return (
    <>
      <TableRow
        data-sql-review-rule-key={ruleKey}
        className={cn(
          supportSelect ? "cursor-pointer" : "",
          highlighted &&
            "outline outline-1 outline-error outline-offset-[-1px] bg-error/5!"
        )}
        onClick={supportSelect ? onToggleRule : undefined}
      >
        <TableCell className="w-8" onClick={(e) => e.stopPropagation()}>
          {expandable && (
            <button
              type="button"
              className="cursor-pointer p-0.5 text-control-light hover:text-control"
              onClick={() => setExpanded((prev) => !prev)}
            >
              {expanded ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )}
            </button>
          )}
        </TableCell>
        {supportSelect && (
          <TableCell className="w-8" onClick={(e) => e.stopPropagation()}>
            <Checkbox
              checked={isSelected}
              onCheckedChange={() => onToggleRule()}
            />
          </TableCell>
        )}
        <TableCell>
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
        </TableCell>
        {!hideLevel && (
          <TableCell onClick={(e) => e.stopPropagation()}>
            <RuleLevelSwitch
              level={rule.level}
              disabled={!editable}
              onLevelChange={updateLevel}
            />
          </TableCell>
        )}
        {!supportSelect && (
          <TableCell onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-end gap-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setEditing(true)}
              >
                {editable ? t("common.edit") : t("common.view")}
              </Button>
              {editable && !isBuiltinRule(rule) && (
                <Button
                  variant="ghost-destructive"
                  size="sm"
                  onClick={() => onRuleRemove?.(rule)}
                >
                  {t("common.delete")}
                </Button>
              )}
            </div>
          </TableCell>
        )}
      </TableRow>
      {expanded && (
        <TableRow>
          <TableCell
            colSpan={colSpan}
            className={cn(
              "px-10 bg-control-bg/40",
              highlighted && "bg-error/5"
            )}
          >
            {loc.description && (
              <p className="text-control-light">{loc.description}</p>
            )}
            {rule.componentList.length > 0 && loc.description && (
              <hr className="my-4 border-control-border" />
            )}
            {rule.componentList.length > 0 && (
              <RuleConfig rule={rule} disabled size="small" />
            )}
          </TableCell>
        </TableRow>
      )}
      {editing && (
        <RuleEditDialog
          key={ruleKey}
          rule={rule}
          disabled={!editable}
          onCancel={() => setEditing(false)}
          onUpdateRule={onRuleChanged}
        />
      )}
    </>
  );
}

const MemoizedRuleTableRow = memo(RuleTableRow);

interface MobileRuleTableRowProps {
  ruleKey: string;
  rule: RuleTemplateV2;
  highlighted: boolean;
  supportSelect: boolean;
  hideLevel: boolean;
  editable: boolean;
  isSelected: boolean;
  onToggleRule: () => void;
  onRuleUpsert?: (
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ) => void;
  onRuleRemove?: (rule: RuleTemplateV2) => void;
}

const MobileRuleTableRow = memo(function MobileRuleTableRow({
  ruleKey,
  rule,
  highlighted,
  supportSelect,
  hideLevel,
  editable,
  isSelected,
  onToggleRule,
  onRuleUpsert,
  onRuleRemove,
}: MobileRuleTableRowProps) {
  const { t } = useTranslation();
  const [editing, setEditing] = useState(false);
  const loc = useMemo(
    () => getRuleLocalization(ruleTypeToString(rule.type), rule.engine),
    [rule.engine, rule.type]
  );

  const updateLevel = (level: SQLReviewRule_Level) => {
    onRuleUpsert?.(rule, { level });
  };

  const onRuleChanged = (update: Partial<RuleTemplateV2>) => {
    onRuleUpsert?.(rule, update);
  };

  return (
    <div
      data-sql-review-rule-key={ruleKey}
      className={cn(
        "pt-4 flex flex-col gap-y-3",
        highlighted &&
          "rounded-sm outline outline-1 outline-error outline-offset-[-1px] bg-error/5"
      )}
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
          <Checkbox checked={isSelected} onCheckedChange={onToggleRule} />
        ) : (
          <div className="flex items-center gap-x-2">
            {editable && (
              <Pencil
                className="w-4 h-4 cursor-pointer hover:text-accent"
                onClick={() => setEditing(true)}
              />
            )}
            {editable && !isBuiltinRule(rule) && (
              <Button
                variant="ghost-destructive"
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
          onLevelChange={updateLevel}
        />
      )}
      <p className="text-sm text-control-placeholder">{loc.description}</p>
      {editing && (
        <RuleEditDialog
          key={ruleKey}
          rule={rule}
          disabled={!editable}
          onCancel={() => setEditing(false)}
          onUpdateRule={onRuleChanged}
        />
      )}
    </div>
  );
});

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
  focusRuleKey?: string;
  focusRuleSignal?: number;
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
  focusRuleKey,
  focusRuleSignal,
}: RuleTableWithFilterProps) {
  const { t } = useTranslation();
  const { params, events } = useSQLRuleFilter();

  useEffect(() => {
    events.reset();
  }, [engine, events]);

  useEffect(() => {
    if (focusRuleKey) {
      events.reset();
    }
  }, [focusRuleKey, events]);

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
      {(filteredRuleList) => {
        return filteredRuleList.length > 0 ? (
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
            focusRuleKey={focusRuleKey}
            focusRuleSignal={focusRuleSignal}
          />
        ) : (
          <div className="py-12 border rounded-sm text-center text-control-placeholder">
            {t("common.no-data")}
          </div>
        );
      }}
    </RuleFilter>
  );
}
