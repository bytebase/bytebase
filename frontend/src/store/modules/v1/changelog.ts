import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { databaseServiceClientConnect } from "@/grpcweb";
import { useCache } from "@/store/cache";
import { UNKNOWN_ID } from "@/types";
import {
  type Changelog,
  ChangelogView,
  type GetChangelogRequest,
  GetChangelogRequestSchema,
  type ListChangelogsRequest,
  ListChangelogsRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { extractChangelogUID } from "@/utils/v1/changelog";

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
      cache.setEntity([changelog.name, ChangelogView.BASIC], changelog);
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
      view: params.view,
      filter: params.filter,
    });
    const response = await databaseServiceClientConnect.listChangelogs(request);
    const changelogs = response.changelogs;
    const { nextPageToken } = response;
    await upsertChangelogsMap(parent, changelogs);
    return { changelogs, nextPageToken };
  };
  const getOrFetchChangelogListOfDatabase = async (
    databaseName: string,
    pageSize: number,
    view = ChangelogView.BASIC
  ) => {
    if (changelogsMapByDatabase.has(databaseName)) {
      return changelogsMapByDatabase.get(databaseName) ?? [];
    }
    const { changelogs } = await fetchChangelogList({
      parent: databaseName,
      pageSize,
      view,
    });
    return changelogs;
  };
  const changelogListByDatabase = (name: string) => {
    return changelogsMapByDatabase.get(name) ?? [];
  };
  const fetchChangelog = async (params: Partial<GetChangelogRequest>) => {
    const request = create(GetChangelogRequestSchema, {
      name: params.name,
      view: params.view,
    });
    const changelog = await databaseServiceClientConnect.getChangelog(request);
    cache.setEntity(
      [changelog.name, params.view ?? ChangelogView.BASIC],
      changelog
    );
    return changelog;
  };
  const getOrFetchChangelogByName = async (
    name: string,
    view: ChangelogView = ChangelogView.BASIC
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
        cache.getEntity([name, ChangelogView.FULL]) ??
        cache.getEntity([name, ChangelogView.BASIC])
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
