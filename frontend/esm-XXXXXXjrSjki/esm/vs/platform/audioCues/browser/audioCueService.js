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
import { Disposable, toDisposable } from '../../../base/common/lifecycle.js';
import { FileAccess } from '../../../base/common/network.js';
import { IAccessibilityService } from '../../accessibility/common/accessibility.js';
import { IConfigurationService } from '../../configuration/common/configuration.js';
import { createDecorator } from '../../instantiation/common/instantiation.js';
import { Event } from '../../../base/common/event.js';
import { localizeWithPath } from '../../../nls.js';
import { observableFromEvent, derived } from '../../../base/common/observable.js';
import { ITelemetryService } from '../../telemetry/common/telemetry.js';
export const IAudioCueService = createDecorator('audioCue');
let AudioCueService = class AudioCueService extends Disposable {
    constructor(configurationService, accessibilityService, telemetryService) {
        super();
        this.configurationService = configurationService;
        this.accessibilityService = accessibilityService;
        this.telemetryService = telemetryService;
        this.sounds = new Map();
        this.screenReaderAttached = observableFromEvent(this.accessibilityService.onDidChangeScreenReaderOptimized, () => /** @description accessibilityService.onDidChangeScreenReaderOptimized */ this.accessibilityService.isScreenReaderOptimized());
        this.sentTelemetry = new Set();
        this.playingSounds = new Set();
        this.obsoleteAudioCuesEnabled = observableFromEvent(Event.filter(this.configurationService.onDidChangeConfiguration, (e) => e.affectsConfiguration('audioCues.enabled')), () => /** @description config: audioCues.enabled */ this.configurationService.getValue('audioCues.enabled'));
        this.isEnabledCache = new Cache((cue) => {
            const settingObservable = observableFromEvent(Event.filter(this.configurationService.onDidChangeConfiguration, (e) => e.affectsConfiguration(cue.settingsKey)), () => this.configurationService.getValue(cue.settingsKey));
            return derived(reader => {
                /** @description audio cue enabled */
                const setting = settingObservable.read(reader);
                if (setting === 'on' ||
                    (setting === 'auto' && this.screenReaderAttached.read(reader))) {
                    return true;
                }
                const obsoleteSetting = this.obsoleteAudioCuesEnabled.read(reader);
                if (obsoleteSetting === 'on' ||
                    (obsoleteSetting === 'auto' && this.screenReaderAttached.read(reader))) {
                    return true;
                }
                return false;
            });
        });
    }
    async playAudioCue(cue, options = {}) {
        if (this.isEnabled(cue)) {
            this.sendAudioCueTelemetry(cue, options.source);
            await this.playSound(cue.sound.getSound(), options.allowManyInParallel);
        }
    }
    async playAudioCues(cues) {
        for (const cue of cues) {
            this.sendAudioCueTelemetry('cue' in cue ? cue.cue : cue, 'source' in cue ? cue.source : undefined);
        }
        // Some audio cues might reuse sounds. Don't play the same sound twice.
        const sounds = new Set(cues.map(c => 'cue' in c ? c.cue : c).filter(cue => this.isEnabled(cue)).map(cue => cue.sound.getSound()));
        await Promise.all(Array.from(sounds).map(sound => this.playSound(sound, true)));
    }
    sendAudioCueTelemetry(cue, source) {
        const isScreenReaderOptimized = this.accessibilityService.isScreenReaderOptimized();
        const key = cue.name + (source ? `::${source}` : '') + (isScreenReaderOptimized ? '{screenReaderOptimized}' : '');
        // Only send once per user session
        if (this.sentTelemetry.has(key) || this.getVolumeInPercent() === 0) {
            return;
        }
        this.sentTelemetry.add(key);
        this.telemetryService.publicLog2('audioCue.played', {
            audioCue: cue.name,
            source: source ?? '',
            isScreenReaderOptimized,
        });
    }
    getVolumeInPercent() {
        const volume = this.configurationService.getValue('audioCues.volume');
        if (typeof volume !== 'number') {
            return 50;
        }
        return Math.max(Math.min(volume, 100), 0);
    }
    async playSound(sound, allowManyInParallel = false) {
        if (!allowManyInParallel && this.playingSounds.has(sound)) {
            return;
        }
        this.playingSounds.add(sound);
        const url = FileAccess.asBrowserUri(`vs/platform/audioCues/browser/media/${sound.fileName}`).toString(true);
        try {
            const sound = this.sounds.get(url);
            if (sound) {
                sound.volume = this.getVolumeInPercent() / 100;
                sound.currentTime = 0;
                await sound.play();
            }
            else {
                const playedSound = await playAudio(url, this.getVolumeInPercent() / 100);
                this.sounds.set(url, playedSound);
            }
        }
        catch (e) {
            if (!e.message.includes('play() can only be initiated by a user gesture')) {
                // tracking this issue in #178642, no need to spam the console
                console.error('Error while playing sound', e);
            }
        }
        finally {
            this.playingSounds.delete(sound);
        }
    }
    playAudioCueLoop(cue, milliseconds) {
        let playing = true;
        const playSound = () => {
            if (playing) {
                this.playAudioCue(cue, { allowManyInParallel: true }).finally(() => {
                    setTimeout(() => {
                        if (playing) {
                            playSound();
                        }
                    }, milliseconds);
                });
            }
        };
        playSound();
        return toDisposable(() => playing = false);
    }
    isEnabled(cue) {
        return this.isEnabledCache.get(cue).get();
    }
    onEnabledChanged(cue) {
        return Event.fromObservableLight(this.isEnabledCache.get(cue));
    }
};
AudioCueService = __decorate([
    __param(0, IConfigurationService),
    __param(1, IAccessibilityService),
    __param(2, ITelemetryService)
], AudioCueService);
export { AudioCueService };
/**
 * Play the given audio url.
 * @volume value between 0 and 1
 */
function playAudio(url, volume) {
    return new Promise((resolve, reject) => {
        const audio = new Audio(url);
        audio.volume = volume;
        audio.addEventListener('ended', () => {
            resolve(audio);
        });
        audio.addEventListener('error', (e) => {
            // When the error event fires, ended might not be called
            reject(e.error);
        });
        audio.play().catch(e => {
            // When play fails, the error event is not fired.
            reject(e);
        });
    });
}
class Cache {
    constructor(getValue) {
        this.getValue = getValue;
        this.map = new Map();
    }
    get(arg) {
        if (this.map.has(arg)) {
            return this.map.get(arg);
        }
        const value = this.getValue(arg);
        this.map.set(arg, value);
        return value;
    }
}
/**
 * Corresponds to the audio files in ./media.
*/
export class Sound {
    static register(options) {
        const sound = new Sound(options.fileName);
        return sound;
    }
    constructor(fileName) {
        this.fileName = fileName;
    }
}
Sound.error = Sound.register({ fileName: 'error.mp3' });
Sound.warning = Sound.register({ fileName: 'warning.mp3' });
Sound.foldedArea = Sound.register({ fileName: 'foldedAreas.mp3' });
Sound.break = Sound.register({ fileName: 'break.mp3' });
Sound.quickFixes = Sound.register({ fileName: 'quickFixes.mp3' });
Sound.taskCompleted = Sound.register({ fileName: 'taskCompleted.mp3' });
Sound.taskFailed = Sound.register({ fileName: 'taskFailed.mp3' });
Sound.terminalBell = Sound.register({ fileName: 'terminalBell.mp3' });
Sound.diffLineInserted = Sound.register({ fileName: 'diffLineInserted.mp3' });
Sound.diffLineDeleted = Sound.register({ fileName: 'diffLineDeleted.mp3' });
Sound.diffLineModified = Sound.register({ fileName: 'diffLineModified.mp3' });
Sound.chatRequestSent = Sound.register({ fileName: 'chatRequestSent.mp3' });
Sound.chatResponsePending = Sound.register({ fileName: 'chatResponsePending.mp3' });
Sound.chatResponseReceived1 = Sound.register({ fileName: 'chatResponseReceived1.mp3' });
Sound.chatResponseReceived2 = Sound.register({ fileName: 'chatResponseReceived2.mp3' });
Sound.chatResponseReceived3 = Sound.register({ fileName: 'chatResponseReceived3.mp3' });
Sound.chatResponseReceived4 = Sound.register({ fileName: 'chatResponseReceived4.mp3' });
Sound.clear = Sound.register({ fileName: 'clear.mp3' });
Sound.save = Sound.register({ fileName: 'save.mp3' });
Sound.format = Sound.register({ fileName: 'format.mp3' });
export class SoundSource {
    constructor(randomOneOf) {
        this.randomOneOf = randomOneOf;
    }
    getSound(deterministic = false) {
        if (deterministic || this.randomOneOf.length === 1) {
            return this.randomOneOf[0];
        }
        else {
            const index = Math.floor(Math.random() * this.randomOneOf.length);
            return this.randomOneOf[index];
        }
    }
}
export class AudioCue {
    static register(options) {
        const soundSource = new SoundSource('randomOneOf' in options.sound ? options.sound.randomOneOf : [options.sound]);
        const audioCue = new AudioCue(soundSource, options.name, options.settingsKey);
        AudioCue._audioCues.add(audioCue);
        return audioCue;
    }
    static get allAudioCues() {
        return [...this._audioCues];
    }
    constructor(sound, name, settingsKey) {
        this.sound = sound;
        this.name = name;
        this.settingsKey = settingsKey;
    }
}
AudioCue._audioCues = new Set();
AudioCue.error = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.lineHasError.name', 'Error on Line'),
    sound: Sound.error,
    settingsKey: 'audioCues.lineHasError',
});
AudioCue.warning = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.lineHasWarning.name', 'Warning on Line'),
    sound: Sound.warning,
    settingsKey: 'audioCues.lineHasWarning',
});
AudioCue.foldedArea = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.lineHasFoldedArea.name', 'Folded Area on Line'),
    sound: Sound.foldedArea,
    settingsKey: 'audioCues.lineHasFoldedArea',
});
AudioCue.break = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.lineHasBreakpoint.name', 'Breakpoint on Line'),
    sound: Sound.break,
    settingsKey: 'audioCues.lineHasBreakpoint',
});
AudioCue.inlineSuggestion = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.lineHasInlineSuggestion.name', 'Inline Suggestion on Line'),
    sound: Sound.quickFixes,
    settingsKey: 'audioCues.lineHasInlineSuggestion',
});
AudioCue.terminalQuickFix = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.terminalQuickFix.name', 'Terminal Quick Fix'),
    sound: Sound.quickFixes,
    settingsKey: 'audioCues.terminalQuickFix',
});
AudioCue.onDebugBreak = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.onDebugBreak.name', 'Debugger Stopped on Breakpoint'),
    sound: Sound.break,
    settingsKey: 'audioCues.onDebugBreak',
});
AudioCue.noInlayHints = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.noInlayHints', 'No Inlay Hints on Line'),
    sound: Sound.error,
    settingsKey: 'audioCues.noInlayHints'
});
AudioCue.taskCompleted = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.taskCompleted', 'Task Completed'),
    sound: Sound.taskCompleted,
    settingsKey: 'audioCues.taskCompleted'
});
AudioCue.taskFailed = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.taskFailed', 'Task Failed'),
    sound: Sound.taskFailed,
    settingsKey: 'audioCues.taskFailed'
});
AudioCue.terminalCommandFailed = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.terminalCommandFailed', 'Terminal Command Failed'),
    sound: Sound.error,
    settingsKey: 'audioCues.terminalCommandFailed'
});
AudioCue.terminalBell = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.terminalBell', 'Terminal Bell'),
    sound: Sound.terminalBell,
    settingsKey: 'audioCues.terminalBell'
});
AudioCue.notebookCellCompleted = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.notebookCellCompleted', 'Notebook Cell Completed'),
    sound: Sound.taskCompleted,
    settingsKey: 'audioCues.notebookCellCompleted'
});
AudioCue.notebookCellFailed = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.notebookCellFailed', 'Notebook Cell Failed'),
    sound: Sound.taskFailed,
    settingsKey: 'audioCues.notebookCellFailed'
});
AudioCue.diffLineInserted = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.diffLineInserted', 'Diff Line Inserted'),
    sound: Sound.diffLineInserted,
    settingsKey: 'audioCues.diffLineInserted'
});
AudioCue.diffLineDeleted = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.diffLineDeleted', 'Diff Line Deleted'),
    sound: Sound.diffLineDeleted,
    settingsKey: 'audioCues.diffLineDeleted'
});
AudioCue.diffLineModified = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.diffLineModified', 'Diff Line Modified'),
    sound: Sound.diffLineModified,
    settingsKey: 'audioCues.diffLineModified'
});
AudioCue.chatRequestSent = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.chatRequestSent', 'Chat Request Sent'),
    sound: Sound.chatRequestSent,
    settingsKey: 'audioCues.chatRequestSent'
});
AudioCue.chatResponseReceived = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.chatResponseReceived', 'Chat Response Received'),
    settingsKey: 'audioCues.chatResponseReceived',
    sound: {
        randomOneOf: [
            Sound.chatResponseReceived1,
            Sound.chatResponseReceived2,
            Sound.chatResponseReceived3,
            Sound.chatResponseReceived4
        ]
    }
});
AudioCue.chatResponsePending = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.chatResponsePending', 'Chat Response Pending'),
    sound: Sound.chatResponsePending,
    settingsKey: 'audioCues.chatResponsePending'
});
AudioCue.clear = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.clear', 'Clear'),
    sound: Sound.clear,
    settingsKey: 'audioCues.clear'
});
AudioCue.save = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.save', 'Save'),
    sound: Sound.save,
    settingsKey: 'audioCues.save'
});
AudioCue.format = AudioCue.register({
    name: localizeWithPath('vs/platform/audioCues/browser/audioCueService', 'audioCues.format', 'Format'),
    sound: Sound.format,
    settingsKey: 'audioCues.format'
});
