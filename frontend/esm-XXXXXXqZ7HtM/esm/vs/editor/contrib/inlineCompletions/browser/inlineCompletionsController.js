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
var InlineCompletionsController_1;
import { alert } from '../../../../base/browser/ui/aria/aria.js';
import { Disposable, toDisposable } from '../../../../base/common/lifecycle.js';
import { autorun, autorunHandleChanges, constObservable, derived, disposableObservableValue, observableFromEvent, observableSignal, observableValue, transaction } from '../../../../base/common/observable.js';
import { CoreEditingCommands } from '../../../browser/coreCommands.js';
import { Position } from '../../../common/core/position.js';
import { ILanguageFeatureDebounceService } from '../../../common/services/languageFeatureDebounce.js';
import { ILanguageFeaturesService } from '../../../common/services/languageFeatures.js';
import { inlineSuggestCommitId } from './commandIds.js';
import { GhostTextWidget } from './ghostTextWidget.js';
import { InlineCompletionContextKeys } from './inlineCompletionContextKeys.js';
import { InlineCompletionsHintsWidget, InlineSuggestionHintsContentWidget } from './inlineCompletionsHintsWidget.js';
import { InlineCompletionsModel, VersionIdChangeReason } from './inlineCompletionsModel.js';
import { SuggestWidgetAdaptor } from './suggestWidgetInlineCompletionProvider.js';
import { localizeWithPath } from '../../../../nls.js';
import { AudioCue, IAudioCueService } from '../../../../platform/audioCues/browser/audioCueService.js';
import { ICommandService } from '../../../../platform/commands/common/commands.js';
import { IConfigurationService } from '../../../../platform/configuration/common/configuration.js';
import { IContextKeyService } from '../../../../platform/contextkey/common/contextkey.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
import { IKeybindingService } from '../../../../platform/keybinding/common/keybinding.js';
let InlineCompletionsController = InlineCompletionsController_1 = class InlineCompletionsController extends Disposable {
    static get(editor) {
        return editor.getContribution(InlineCompletionsController_1.ID);
    }
    constructor(editor, _instantiationService, _contextKeyService, _configurationService, _commandService, _debounceService, _languageFeaturesService, _audioCueService, _keybindingService) {
        super();
        this.editor = editor;
        this._instantiationService = _instantiationService;
        this._contextKeyService = _contextKeyService;
        this._configurationService = _configurationService;
        this._commandService = _commandService;
        this._debounceService = _debounceService;
        this._languageFeaturesService = _languageFeaturesService;
        this._audioCueService = _audioCueService;
        this._keybindingService = _keybindingService;
        this.model = disposableObservableValue('inlineCompletionModel', undefined);
        this._textModelVersionId = observableValue(this, -1);
        this._cursorPosition = observableValue(this, new Position(1, 1));
        this._suggestWidgetAdaptor = this._register(new SuggestWidgetAdaptor(this.editor, () => this.model.get()?.selectedInlineCompletion.get()?.toSingleTextEdit(undefined), (tx) => this.updateObservables(tx, VersionIdChangeReason.Other), (item) => {
            transaction(tx => {
                /** @description InlineCompletionsController.handleSuggestAccepted */
                this.updateObservables(tx, VersionIdChangeReason.Other);
                this.model.get()?.handleSuggestAccepted(item);
            });
        }));
        this._enabled = observableFromEvent(this.editor.onDidChangeConfiguration, () => this.editor.getOption(62 /* EditorOption.inlineSuggest */).enabled);
        this._ghostTextWidget = this._register(this._instantiationService.createInstance(GhostTextWidget, this.editor, {
            ghostText: this.model.map((v, reader) => /** ghostText */ v?.ghostText.read(reader)),
            minReservedLineCount: constObservable(0),
            targetTextModel: this.model.map(v => v?.textModel),
        }));
        this._debounceValue = this._debounceService.for(this._languageFeaturesService.inlineCompletionsProvider, 'InlineCompletionsDebounce', { min: 50, max: 50 });
        this._playAudioCueSignal = observableSignal(this);
        this._isReadonly = observableFromEvent(this.editor.onDidChangeConfiguration, () => this.editor.getOption(90 /* EditorOption.readOnly */));
        this._textModel = observableFromEvent(this.editor.onDidChangeModel, () => this.editor.getModel());
        this._textModelIfWritable = derived(reader => this._isReadonly.read(reader) ? undefined : this._textModel.read(reader));
        this._register(new InlineCompletionContextKeys(this._contextKeyService, this.model));
        this._register(autorun(reader => {
            /** @description InlineCompletionsController.update model */
            const textModel = this._textModelIfWritable.read(reader);
            transaction(tx => {
                /** @description InlineCompletionsController.onDidChangeModel/readonly */
                this.model.set(undefined, tx);
                this.updateObservables(tx, VersionIdChangeReason.Other);
                if (textModel) {
                    const model = _instantiationService.createInstance(InlineCompletionsModel, textModel, this._suggestWidgetAdaptor.selectedItem, this._cursorPosition, this._textModelVersionId, this._debounceValue, observableFromEvent(editor.onDidChangeConfiguration, () => editor.getOption(117 /* EditorOption.suggest */).preview), observableFromEvent(editor.onDidChangeConfiguration, () => editor.getOption(117 /* EditorOption.suggest */).previewMode), observableFromEvent(editor.onDidChangeConfiguration, () => editor.getOption(62 /* EditorOption.inlineSuggest */).mode), this._enabled);
                    this.model.set(model, tx);
                }
            });
        }));
        const getReason = (e) => {
            if (e.isUndoing) {
                return VersionIdChangeReason.Undo;
            }
            if (e.isRedoing) {
                return VersionIdChangeReason.Redo;
            }
            if (this.model.get()?.isAcceptingPartially) {
                return VersionIdChangeReason.AcceptWord;
            }
            return VersionIdChangeReason.Other;
        };
        this._register(editor.onDidChangeModelContent((e) => transaction(tx => 
        /** @description InlineCompletionsController.onDidChangeModelContent */
        this.updateObservables(tx, getReason(e)))));
        this._register(editor.onDidChangeCursorPosition(e => transaction(tx => {
            /** @description InlineCompletionsController.onDidChangeCursorPosition */
            this.updateObservables(tx, VersionIdChangeReason.Other);
            if (e.reason === 3 /* CursorChangeReason.Explicit */ || e.source === 'api') {
                this.model.get()?.stop(tx);
            }
        })));
        this._register(editor.onDidType(() => transaction(tx => {
            /** @description InlineCompletionsController.onDidType */
            this.updateObservables(tx, VersionIdChangeReason.Other);
            if (this._enabled.get()) {
                this.model.get()?.trigger(tx);
            }
        })));
        this._register(this._commandService.onDidExecuteCommand((e) => {
            // These commands don't trigger onDidType.
            const commands = new Set([
                CoreEditingCommands.Tab.id,
                CoreEditingCommands.DeleteLeft.id,
                CoreEditingCommands.DeleteRight.id,
                inlineSuggestCommitId,
                'acceptSelectedSuggestion',
            ]);
            if (commands.has(e.commandId) && editor.hasTextFocus() && this._enabled.get()) {
                transaction(tx => {
                    /** @description onDidExecuteCommand */
                    this.model.get()?.trigger(tx);
                });
            }
        }));
        this._register(this.editor.onDidBlurEditorWidget(() => {
            // This is a hidden setting very useful for debugging
            if (this._contextKeyService.getContextKeyValue('accessibleViewIsShown') || this._configurationService.getValue('editor.inlineSuggest.keepOnBlur') ||
                editor.getOption(62 /* EditorOption.inlineSuggest */).keepOnBlur) {
                return;
            }
            if (InlineSuggestionHintsContentWidget.dropDownVisible) {
                return;
            }
            transaction(tx => {
                /** @description InlineCompletionsController.onDidBlurEditorWidget */
                this.model.get()?.stop(tx);
            });
        }));
        this._register(autorun(reader => {
            /** @description InlineCompletionsController.forceRenderingAbove */
            const state = this.model.read(reader)?.state.read(reader);
            if (state?.suggestItem) {
                if (state.ghostText.lineCount >= 2) {
                    this._suggestWidgetAdaptor.forceRenderingAbove();
                }
            }
            else {
                this._suggestWidgetAdaptor.stopForceRenderingAbove();
            }
        }));
        this._register(toDisposable(() => {
            this._suggestWidgetAdaptor.stopForceRenderingAbove();
        }));
        let lastInlineCompletionId = undefined;
        this._register(autorunHandleChanges({
            handleChange: (context, changeSummary) => {
                if (context.didChange(this._playAudioCueSignal)) {
                    lastInlineCompletionId = undefined;
                }
                return true;
            },
        }, async (reader) => {
            /** @description InlineCompletionsController.playAudioCueAndReadSuggestion */
            this._playAudioCueSignal.read(reader);
            const model = this.model.read(reader);
            const state = model?.state.read(reader);
            if (!model || !state || !state.inlineCompletion) {
                lastInlineCompletionId = undefined;
                return;
            }
            if (state.inlineCompletion.semanticId !== lastInlineCompletionId) {
                lastInlineCompletionId = state.inlineCompletion.semanticId;
                const lineText = model.textModel.getLineContent(state.ghostText.lineNumber);
                this._audioCueService.playAudioCue(AudioCue.inlineSuggestion).then(() => {
                    if (this.editor.getOption(8 /* EditorOption.screenReaderAnnounceInlineSuggestion */)) {
                        this.provideScreenReaderUpdate(state.ghostText.renderForScreenReader(lineText));
                    }
                });
            }
        }));
        this._register(new InlineCompletionsHintsWidget(this.editor, this.model, this._instantiationService));
        this._register(this._configurationService.onDidChangeConfiguration(e => {
            if (e.affectsConfiguration('accessibility.verbosity.inlineCompletions')) {
                this.editor.updateOptions({ inlineCompletionsAccessibilityVerbose: this._configurationService.getValue('accessibility.verbosity.inlineCompletions') });
            }
        }));
        this.editor.updateOptions({ inlineCompletionsAccessibilityVerbose: this._configurationService.getValue('accessibility.verbosity.inlineCompletions') });
    }
    playAudioCue(tx) {
        this._playAudioCueSignal.trigger(tx);
    }
    provideScreenReaderUpdate(content) {
        const accessibleViewShowing = this._contextKeyService.getContextKeyValue('accessibleViewIsShown');
        const accessibleViewKeybinding = this._keybindingService.lookupKeybinding('editor.action.accessibleView');
        let hint;
        if (!accessibleViewShowing && accessibleViewKeybinding && this.editor.getOption(147 /* EditorOption.inlineCompletionsAccessibilityVerbose */)) {
            hint = localizeWithPath('vs/editor/contrib/inlineCompletions/browser/inlineCompletionsController', 'showAccessibleViewHint', "Inspect this in the accessible view ({0})", accessibleViewKeybinding.getAriaLabel());
        }
        hint ? alert(content + ', ' + hint) : alert(content);
    }
    /**
     * Copies over the relevant state from the text model to observables.
     * This solves all kind of eventing issues, as we make sure we always operate on the latest state,
     * regardless of who calls into us.
     */
    updateObservables(tx, changeReason) {
        const newModel = this.editor.getModel();
        this._textModelVersionId.set(newModel?.getVersionId() ?? -1, tx, changeReason);
        this._cursorPosition.set(this.editor.getPosition() ?? new Position(1, 1), tx);
    }
    shouldShowHoverAt(range) {
        const ghostText = this.model.get()?.ghostText.get();
        if (ghostText) {
            return ghostText.parts.some(p => range.containsPosition(new Position(ghostText.lineNumber, p.column)));
        }
        return false;
    }
    shouldShowHoverAtViewZone(viewZoneId) {
        return this._ghostTextWidget.ownsViewZone(viewZoneId);
    }
    hide() {
        transaction(tx => {
            this.model.get()?.stop(tx);
        });
    }
};
InlineCompletionsController.ID = 'editor.contrib.inlineCompletionsController';
InlineCompletionsController = InlineCompletionsController_1 = __decorate([
    __param(1, IInstantiationService),
    __param(2, IContextKeyService),
    __param(3, IConfigurationService),
    __param(4, ICommandService),
    __param(5, ILanguageFeatureDebounceService),
    __param(6, ILanguageFeaturesService),
    __param(7, IAudioCueService),
    __param(8, IKeybindingService)
], InlineCompletionsController);
export { InlineCompletionsController };
