/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
import { hash } from '../../../base/common/hash.js';
import { Emitter } from '../../../base/common/event.js';
import { Disposable } from '../../../base/common/lifecycle.js';
import { basename, joinPath } from '../../../base/common/resources.js';
import { URI } from '../../../base/common/uri.js';
import { localizeWithPath } from '../../../nls.js';
import { IEnvironmentService } from '../../environment/common/environment.js';
import { IFileService, toFileOperationResult } from '../../files/common/files.js';
import { createDecorator } from '../../instantiation/common/instantiation.js';
import { ILogService } from '../../log/common/log.js';
import { isSingleFolderWorkspaceIdentifier, isWorkspaceIdentifier } from '../../workspace/common/workspace.js';
import { ResourceMap } from '../../../base/common/map.js';
import { IUriIdentityService } from '../../uriIdentity/common/uriIdentity.js';
import { Promises } from '../../../base/common/async.js';
import { generateUuid } from '../../../base/common/uuid.js';
import { escapeRegExpCharacters } from '../../../base/common/strings.js';
import { isString } from '../../../base/common/types.js';
export function isUserDataProfile(thing) {
    const candidate = thing;
    return !!(candidate && typeof candidate === 'object'
        && typeof candidate.id === 'string'
        && typeof candidate.isDefault === 'boolean'
        && typeof candidate.name === 'string'
        && URI.isUri(candidate.location)
        && URI.isUri(candidate.globalStorageHome)
        && URI.isUri(candidate.settingsResource)
        && URI.isUri(candidate.keybindingsResource)
        && URI.isUri(candidate.tasksResource)
        && URI.isUri(candidate.snippetsHome)
        && URI.isUri(candidate.extensionsResource));
}
export const IUserDataProfilesService = createDecorator('IUserDataProfilesService');
export function reviveProfile(profile, scheme) {
    return {
        id: profile.id,
        isDefault: profile.isDefault,
        name: profile.name,
        shortName: profile.shortName,
        icon: profile.icon,
        location: URI.revive(profile.location).with({ scheme }),
        globalStorageHome: URI.revive(profile.globalStorageHome).with({ scheme }),
        settingsResource: URI.revive(profile.settingsResource).with({ scheme }),
        keybindingsResource: URI.revive(profile.keybindingsResource).with({ scheme }),
        tasksResource: URI.revive(profile.tasksResource).with({ scheme }),
        snippetsHome: URI.revive(profile.snippetsHome).with({ scheme }),
        extensionsResource: URI.revive(profile.extensionsResource).with({ scheme }),
        cacheHome: URI.revive(profile.cacheHome).with({ scheme }),
        useDefaultFlags: profile.useDefaultFlags,
        isTransient: profile.isTransient,
    };
}
export function toUserDataProfile(id, name, location, profilesCacheHome, options, defaultProfile) {
    return {
        id,
        name,
        location,
        isDefault: false,
        shortName: options?.shortName,
        icon: options?.icon,
        globalStorageHome: defaultProfile && options?.useDefaultFlags?.globalState ? defaultProfile.globalStorageHome : joinPath(location, 'globalStorage'),
        settingsResource: defaultProfile && options?.useDefaultFlags?.settings ? defaultProfile.settingsResource : joinPath(location, 'settings.json'),
        keybindingsResource: defaultProfile && options?.useDefaultFlags?.keybindings ? defaultProfile.keybindingsResource : joinPath(location, 'keybindings.json'),
        tasksResource: defaultProfile && options?.useDefaultFlags?.tasks ? defaultProfile.tasksResource : joinPath(location, 'tasks.json'),
        snippetsHome: defaultProfile && options?.useDefaultFlags?.snippets ? defaultProfile.snippetsHome : joinPath(location, 'snippets'),
        extensionsResource: defaultProfile && options?.useDefaultFlags?.extensions ? defaultProfile.extensionsResource : joinPath(location, 'extensions.json'),
        cacheHome: joinPath(profilesCacheHome, id),
        useDefaultFlags: options?.useDefaultFlags,
        isTransient: options?.transient
    };
}
let UserDataProfilesService = class UserDataProfilesService extends Disposable {
    get defaultProfile() { return this.profiles[0]; }
    get profiles() { return [...this.profilesObject.profiles, ...this.transientProfilesObject.profiles]; }
    constructor(environmentService, fileService, uriIdentityService, logService) {
        super();
        this.environmentService = environmentService;
        this.fileService = fileService;
        this.uriIdentityService = uriIdentityService;
        this.logService = logService;
        this.enabled = true;
        this._onDidChangeProfiles = this._register(new Emitter());
        this.onDidChangeProfiles = this._onDidChangeProfiles.event;
        this._onWillCreateProfile = this._register(new Emitter());
        this.onWillCreateProfile = this._onWillCreateProfile.event;
        this._onWillRemoveProfile = this._register(new Emitter());
        this.onWillRemoveProfile = this._onWillRemoveProfile.event;
        this._onDidResetWorkspaces = this._register(new Emitter());
        this.onDidResetWorkspaces = this._onDidResetWorkspaces.event;
        this.profileCreationPromises = new Map();
        this.transientProfilesObject = {
            profiles: [],
            workspaces: new ResourceMap(),
            emptyWindows: new Map()
        };
        this.profilesHome = joinPath(this.environmentService.userRoamingDataHome, 'profiles');
        this.profilesCacheHome = joinPath(this.environmentService.cacheHome, 'CachedProfilesData');
    }
    init() {
        this._profilesObject = undefined;
    }
    setEnablement(enabled) {
        if (this.enabled !== enabled) {
            this._profilesObject = undefined;
            this.enabled = enabled;
        }
    }
    isEnabled() {
        return this.enabled;
    }
    get profilesObject() {
        if (!this._profilesObject) {
            const defaultProfile = this.createDefaultProfile();
            const profiles = [defaultProfile];
            if (this.enabled) {
                try {
                    for (const storedProfile of this.getStoredProfiles()) {
                        if (!storedProfile.name || !isString(storedProfile.name) || !storedProfile.location) {
                            this.logService.warn('Skipping the invalid stored profile', storedProfile.location || storedProfile.name);
                            continue;
                        }
                        profiles.push(toUserDataProfile(basename(storedProfile.location), storedProfile.name, storedProfile.location, this.profilesCacheHome, { shortName: storedProfile.shortName, icon: storedProfile.icon, useDefaultFlags: storedProfile.useDefaultFlags }, defaultProfile));
                    }
                }
                catch (error) {
                    this.logService.error(error);
                }
            }
            const workspaces = new ResourceMap();
            const emptyWindows = new Map();
            if (profiles.length) {
                try {
                    const profileAssociaitions = this.getStoredProfileAssociations();
                    if (profileAssociaitions.workspaces) {
                        for (const [workspacePath, profileId] of Object.entries(profileAssociaitions.workspaces)) {
                            const workspace = URI.parse(workspacePath);
                            const profile = profiles.find(p => p.id === profileId);
                            if (profile) {
                                workspaces.set(workspace, profile);
                            }
                        }
                    }
                    if (profileAssociaitions.emptyWindows) {
                        for (const [windowId, profileId] of Object.entries(profileAssociaitions.emptyWindows)) {
                            const profile = profiles.find(p => p.id === profileId);
                            if (profile) {
                                emptyWindows.set(windowId, profile);
                            }
                        }
                    }
                }
                catch (error) {
                    this.logService.error(error);
                }
            }
            this._profilesObject = { profiles, workspaces, emptyWindows };
        }
        return this._profilesObject;
    }
    createDefaultProfile() {
        const defaultProfile = toUserDataProfile('__default__profile__', localizeWithPath('vs/platform/userDataProfile/common/userDataProfile', 'defaultProfile', "Default"), this.environmentService.userRoamingDataHome, this.profilesCacheHome);
        return { ...defaultProfile, extensionsResource: this.getDefaultProfileExtensionsLocation() ?? defaultProfile.extensionsResource, isDefault: true };
    }
    async createTransientProfile(workspaceIdentifier) {
        const namePrefix = `Temp`;
        const nameRegEx = new RegExp(`${escapeRegExpCharacters(namePrefix)}\\s(\\d+)`);
        let nameIndex = 0;
        for (const profile of this.profiles) {
            const matches = nameRegEx.exec(profile.name);
            const index = matches ? parseInt(matches[1]) : 0;
            nameIndex = index > nameIndex ? index : nameIndex;
        }
        const name = `${namePrefix} ${nameIndex + 1}`;
        return this.createProfile(hash(generateUuid()).toString(16), name, { transient: true }, workspaceIdentifier);
    }
    async createNamedProfile(name, options, workspaceIdentifier) {
        return this.createProfile(hash(generateUuid()).toString(16), name, options, workspaceIdentifier);
    }
    async createProfile(id, name, options, workspaceIdentifier) {
        if (!this.enabled) {
            throw new Error(`Profiles are disabled in the current environment.`);
        }
        const profile = await this.doCreateProfile(id, name, options);
        if (workspaceIdentifier) {
            await this.setProfileForWorkspace(workspaceIdentifier, profile);
        }
        return profile;
    }
    async doCreateProfile(id, name, options) {
        if (!isString(name) || !name) {
            throw new Error('Name of the profile is mandatory and must be of type `string`');
        }
        let profileCreationPromise = this.profileCreationPromises.get(name);
        if (!profileCreationPromise) {
            profileCreationPromise = (async () => {
                try {
                    const existing = this.profiles.find(p => p.name === name || p.id === id);
                    if (existing) {
                        return existing;
                    }
                    const profile = toUserDataProfile(id, name, joinPath(this.profilesHome, id), this.profilesCacheHome, options, this.defaultProfile);
                    await this.fileService.createFolder(profile.location);
                    const joiners = [];
                    this._onWillCreateProfile.fire({
                        profile,
                        join(promise) {
                            joiners.push(promise);
                        }
                    });
                    await Promises.settled(joiners);
                    this.updateProfiles([profile], [], []);
                    return profile;
                }
                finally {
                    this.profileCreationPromises.delete(name);
                }
            })();
            this.profileCreationPromises.set(name, profileCreationPromise);
        }
        return profileCreationPromise;
    }
    async updateProfile(profileToUpdate, options) {
        if (!this.enabled) {
            throw new Error(`Profiles are disabled in the current environment.`);
        }
        let profile = this.profiles.find(p => p.id === profileToUpdate.id);
        if (!profile) {
            throw new Error(`Profile '${profileToUpdate.name}' does not exist`);
        }
        profile = toUserDataProfile(profile.id, options.name ?? profile.name, profile.location, this.profilesCacheHome, {
            shortName: options.shortName ?? profile.shortName,
            icon: options.icon === null ? undefined : options.icon ?? profile.icon,
            transient: options.transient ?? profile.isTransient,
            useDefaultFlags: options.useDefaultFlags ?? profile.useDefaultFlags
        }, this.defaultProfile);
        this.updateProfiles([], [], [profile]);
        return profile;
    }
    async removeProfile(profileToRemove) {
        if (!this.enabled) {
            throw new Error(`Profiles are disabled in the current environment.`);
        }
        if (profileToRemove.isDefault) {
            throw new Error('Cannot remove default profile');
        }
        const profile = this.profiles.find(p => p.id === profileToRemove.id);
        if (!profile) {
            throw new Error(`Profile '${profileToRemove.name}' does not exist`);
        }
        const joiners = [];
        this._onWillRemoveProfile.fire({
            profile,
            join(promise) {
                joiners.push(promise);
            }
        });
        try {
            await Promise.allSettled(joiners);
        }
        catch (error) {
            this.logService.error(error);
        }
        for (const windowId of [...this.profilesObject.emptyWindows.keys()]) {
            if (profile.id === this.profilesObject.emptyWindows.get(windowId)?.id) {
                this.profilesObject.emptyWindows.delete(windowId);
            }
        }
        for (const workspace of [...this.profilesObject.workspaces.keys()]) {
            if (profile.id === this.profilesObject.workspaces.get(workspace)?.id) {
                this.profilesObject.workspaces.delete(workspace);
            }
        }
        this.updateStoredProfileAssociations();
        this.updateProfiles([], [profile], []);
        try {
            await this.fileService.del(profile.cacheHome, { recursive: true });
        }
        catch (error) {
            if (toFileOperationResult(error) !== 1 /* FileOperationResult.FILE_NOT_FOUND */) {
                this.logService.error(error);
            }
        }
    }
    async setProfileForWorkspace(workspaceIdentifier, profileToSet) {
        if (!this.enabled) {
            throw new Error(`Profiles are disabled in the current environment.`);
        }
        const profile = this.profiles.find(p => p.id === profileToSet.id);
        if (!profile) {
            throw new Error(`Profile '${profileToSet.name}' does not exist`);
        }
        this.updateWorkspaceAssociation(workspaceIdentifier, profile);
    }
    unsetWorkspace(workspaceIdentifier, transient) {
        if (!this.enabled) {
            throw new Error(`Profiles are disabled in the current environment.`);
        }
        this.updateWorkspaceAssociation(workspaceIdentifier, undefined, transient);
    }
    async resetWorkspaces() {
        this.transientProfilesObject.workspaces.clear();
        this.transientProfilesObject.emptyWindows.clear();
        this.profilesObject.workspaces.clear();
        this.profilesObject.emptyWindows.clear();
        this.updateStoredProfileAssociations();
        this._onDidResetWorkspaces.fire();
    }
    async cleanUp() {
        if (!this.enabled) {
            return;
        }
        if (await this.fileService.exists(this.profilesHome)) {
            const stat = await this.fileService.resolve(this.profilesHome);
            await Promise.all((stat.children || [])
                .filter(child => child.isDirectory && this.profiles.every(p => !this.uriIdentityService.extUri.isEqual(p.location, child.resource)))
                .map(child => this.fileService.del(child.resource, { recursive: true })));
        }
    }
    async cleanUpTransientProfiles() {
        if (!this.enabled) {
            return;
        }
        const unAssociatedTransientProfiles = this.transientProfilesObject.profiles.filter(p => !this.isProfileAssociatedToWorkspace(p));
        await Promise.allSettled(unAssociatedTransientProfiles.map(p => this.removeProfile(p)));
    }
    getProfileForWorkspace(workspaceIdentifier) {
        const workspace = this.getWorkspace(workspaceIdentifier);
        return URI.isUri(workspace) ? this.transientProfilesObject.workspaces.get(workspace) ?? this.profilesObject.workspaces.get(workspace) : this.transientProfilesObject.emptyWindows.get(workspace) ?? this.profilesObject.emptyWindows.get(workspace);
    }
    getWorkspace(workspaceIdentifier) {
        if (isSingleFolderWorkspaceIdentifier(workspaceIdentifier)) {
            return workspaceIdentifier.uri;
        }
        if (isWorkspaceIdentifier(workspaceIdentifier)) {
            return workspaceIdentifier.configPath;
        }
        return workspaceIdentifier.id;
    }
    isProfileAssociatedToWorkspace(profile) {
        if ([...this.transientProfilesObject.emptyWindows.values()].some(windowProfile => this.uriIdentityService.extUri.isEqual(windowProfile.location, profile.location))) {
            return true;
        }
        if ([...this.transientProfilesObject.workspaces.values()].some(workspaceProfile => this.uriIdentityService.extUri.isEqual(workspaceProfile.location, profile.location))) {
            return true;
        }
        if ([...this.profilesObject.emptyWindows.values()].some(windowProfile => this.uriIdentityService.extUri.isEqual(windowProfile.location, profile.location))) {
            return true;
        }
        if ([...this.profilesObject.workspaces.values()].some(workspaceProfile => this.uriIdentityService.extUri.isEqual(workspaceProfile.location, profile.location))) {
            return true;
        }
        return false;
    }
    updateProfiles(added, removed, updated) {
        const allProfiles = [...this.profiles, ...added];
        const storedProfiles = [];
        this.transientProfilesObject.profiles = [];
        for (let profile of allProfiles) {
            if (profile.isDefault) {
                continue;
            }
            if (removed.some(p => profile.id === p.id)) {
                continue;
            }
            profile = updated.find(p => profile.id === p.id) ?? profile;
            if (profile.isTransient) {
                this.transientProfilesObject.profiles.push(profile);
            }
            else {
                storedProfiles.push({ location: profile.location, name: profile.name, shortName: profile.shortName, icon: profile.icon, useDefaultFlags: profile.useDefaultFlags });
            }
        }
        this.saveStoredProfiles(storedProfiles);
        this._profilesObject = undefined;
        this.triggerProfilesChanges(added, removed, updated);
    }
    triggerProfilesChanges(added, removed, updated) {
        this._onDidChangeProfiles.fire({ added, removed, updated, all: this.profiles });
    }
    updateWorkspaceAssociation(workspaceIdentifier, newProfile, transient) {
        // Force transient if the new profile to associate is transient
        transient = newProfile?.isTransient ? true : transient;
        if (!transient) {
            // Unset the transiet workspace association if any
            this.updateWorkspaceAssociation(workspaceIdentifier, undefined, true);
        }
        const workspace = this.getWorkspace(workspaceIdentifier);
        const profilesObject = transient ? this.transientProfilesObject : this.profilesObject;
        // Folder or Multiroot workspace
        if (URI.isUri(workspace)) {
            profilesObject.workspaces.delete(workspace);
            if (newProfile) {
                profilesObject.workspaces.set(workspace, newProfile);
            }
        }
        // Empty Window
        else {
            profilesObject.emptyWindows.delete(workspace);
            if (newProfile) {
                profilesObject.emptyWindows.set(workspace, newProfile);
            }
        }
        if (!transient) {
            this.updateStoredProfileAssociations();
        }
    }
    updateStoredProfileAssociations() {
        const workspaces = {};
        for (const [workspace, profile] of this.profilesObject.workspaces.entries()) {
            workspaces[workspace.toString()] = profile.id;
        }
        const emptyWindows = {};
        for (const [windowId, profile] of this.profilesObject.emptyWindows.entries()) {
            emptyWindows[windowId.toString()] = profile.id;
        }
        this.saveStoredProfileAssociations({ workspaces, emptyWindows });
        this._profilesObject = undefined;
    }
    // TODO: @sandy081 Remove migration after couple of releases
    migrateStoredProfileAssociations(storedProfileAssociations) {
        const workspaces = {};
        const defaultProfile = this.createDefaultProfile();
        if (storedProfileAssociations.workspaces) {
            for (const [workspace, location] of Object.entries(storedProfileAssociations.workspaces)) {
                const uri = URI.parse(location);
                workspaces[workspace] = this.uriIdentityService.extUri.isEqual(uri, defaultProfile.location) ? defaultProfile.id : this.uriIdentityService.extUri.basename(uri);
            }
        }
        const emptyWindows = {};
        if (storedProfileAssociations.emptyWindows) {
            for (const [workspace, location] of Object.entries(storedProfileAssociations.emptyWindows)) {
                const uri = URI.parse(location);
                emptyWindows[workspace] = this.uriIdentityService.extUri.isEqual(uri, defaultProfile.location) ? defaultProfile.id : this.uriIdentityService.extUri.basename(uri);
            }
        }
        return { workspaces, emptyWindows };
    }
    getStoredProfiles() { return []; }
    saveStoredProfiles(storedProfiles) { throw new Error('not implemented'); }
    getStoredProfileAssociations() { return {}; }
    saveStoredProfileAssociations(storedProfileAssociations) { throw new Error('not implemented'); }
    getDefaultProfileExtensionsLocation() { return undefined; }
};
UserDataProfilesService.PROFILES_KEY = 'userDataProfiles';
UserDataProfilesService.PROFILE_ASSOCIATIONS_KEY = 'profileAssociations';
UserDataProfilesService = __decorate([
    __param(0, IEnvironmentService),
    __param(1, IFileService),
    __param(2, IUriIdentityService),
    __param(3, ILogService)
], UserDataProfilesService);
export { UserDataProfilesService };
export class InMemoryUserDataProfilesService extends UserDataProfilesService {
    constructor() {
        super(...arguments);
        this.storedProfiles = [];
        this.storedProfileAssociations = {};
    }
    getStoredProfiles() { return this.storedProfiles; }
    saveStoredProfiles(storedProfiles) { this.storedProfiles = storedProfiles; }
    getStoredProfileAssociations() { return this.storedProfileAssociations; }
    saveStoredProfileAssociations(storedProfileAssociations) { this.storedProfileAssociations = storedProfileAssociations; }
}
