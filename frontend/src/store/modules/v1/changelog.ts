import { defineStore } from "pinia";
import { reactive } from "vue";
import { create } from "@bufbuild/protobuf";
import { databaseServiceClientConnect } from "@/grpcweb";
import { 
  ListChangelogsRequestSchema,
  GetChangelogRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { 
  convertNewChangelogToOld,
  convertOldChangelogViewToNew,
} from "@/utils/v1/database-conversions";
import { useCache } from "@/store/cache";
import { UNKNOWN_ID } from "@/types";
import {
  ChangelogView,
  GetChangelogRequest,
  ListChangelogsRequest,
  type Changelog,
} from "@/types/proto/v1/database_service";
import { extractChangelogUID } from "@/utils/v1/changelog";
import { DEFAULT_PAGE_SIZE } from "../common";

type CacheKeyType = [string /* name */, ChangelogView];

export const useChangelogStore = defineStore("changelog", () => {
  const cache = useCache<CacheKeyType, Changelog>("bb.changelog.by-name");
  const changelogsMapByDatabase = reactive(new Map<string, Changelog[]>());

  const upsertChangelogsMap = async (
    parent: string,
    changelogs: Changelog[]
  ) => {
    changelogsMapByDatabase.set(parent, changelogs);
    changelogs.forEach((changelog) => {
      cache.setEntity(
        [changelog.name, ChangelogView.CHANGELOG_VIEW_BASIC],
        changelog
      );
    });
  };

  const clearCache = (parent: string) => {
    changelogsMapByDatabase.delete(parent);
  };

  const fetchChangelogList = async (params: Partial<ListChangelogsRequest>) => {
    const { parent } = params;
    if (!parent) throw new Error('"parent" field is required');
    const request = create(ListChangelogsRequestSchema, {
      parent: params.parent,
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      view: params.view ? convertOldChangelogViewToNew(params.view) : undefined,
      filter: params.filter,
    });
    const response = await databaseServiceClientConnect.listChangelogs(request);
    const changelogs = response.changelogs.map((cl) => convertNewChangelogToOld(cl));
    const { nextPageToken } = response;
    await upsertChangelogsMap(parent, changelogs);
    return { changelogs, nextPageToken };
  };
  const getOrFetchChangelogListOfDatabase = async (
    databaseName: string,
    pageSize = DEFAULT_PAGE_SIZE,
    view = ChangelogView.CHANGELOG_VIEW_BASIC,
    filter = ""
  ) => {
    if (changelogsMapByDatabase.has(databaseName)) {
      return changelogsMapByDatabase.get(databaseName) ?? [];
    }
    // Fetch all changelogs of the database with max DEFAULT_PAGE_SIZE.
    const { changelogs } = await fetchChangelogList({
      parent: databaseName,
      pageSize,
      view,
      filter,
    });
    return changelogs;
  };
  const changelogListByDatabase = (name: string) => {
    return changelogsMapByDatabase.get(name) ?? [];
  };
  const fetchChangelog = async (params: Partial<GetChangelogRequest>) => {
    const request = create(GetChangelogRequestSchema, {
      name: params.name,
      view: params.view ? convertOldChangelogViewToNew(params.view) : undefined,
      sdlFormat: params.sdlFormat,
    });
    const newChangelog = await databaseServiceClientConnect.getChangelog(request);
    const changelog = convertNewChangelogToOld(newChangelog);
    cache.setEntity(
      [changelog.name, params.view ?? ChangelogView.CHANGELOG_VIEW_BASIC],
      changelog
    );
    return changelog;
  };
  const getOrFetchChangelogByName = async (
    name: string,
    view: ChangelogView = ChangelogView.CHANGELOG_VIEW_BASIC
  ) => {
    const uid = extractChangelogUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    const entity = cache.getEntity([name, view]);
    if (entity) {
      return entity;
    }
    const request = cache.getRequest([name, view]);
    if (request) {
      return request;
    }
    const promise = fetchChangelog({ name, view });
    cache.setRequest([name, view], promise);
    return promise;
  };
  /**
   *
   * @param name
   * @param view default undefined to any view (full -> basic)
   * @returns
   */
  const getChangelogByName = (
    name: string,
    view: ChangelogView | undefined = undefined
  ) => {
    if (view === undefined) {
      return (
        cache.getEntity([name, ChangelogView.CHANGELOG_VIEW_FULL]) ??
        cache.getEntity([name, ChangelogView.CHANGELOG_VIEW_BASIC])
      );
    }
    return cache.getEntity([name, view]);
  };

  return {
    clearCache,
    fetchChangelogList,
    getOrFetchChangelogListOfDatabase,
    changelogListByDatabase,
    fetchChangelog,
    getOrFetchChangelogByName,
    getChangelogByName,
  };
});
