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
import { mapFindFirst } from '../../../../base/common/arraysFind.js';
import { BugIndicatingError, onUnexpectedExternalError } from '../../../../base/common/errors.js';
import { Disposable } from '../../../../base/common/lifecycle.js';
import { autorun, derived, derivedHandleChanges, derivedOpts, recomputeInitiallyAndOnChange, observableSignal, observableValue, subtransaction, transaction } from '../../../../base/common/observable.js';
import { isDefined } from '../../../../base/common/types.js';
import { EditOperation } from '../../../common/core/editOperation.js';
import { Position } from '../../../common/core/position.js';
import { Range } from '../../../common/core/range.js';
import { InlineCompletionTriggerKind } from '../../../common/languages.js';
import { ILanguageConfigurationService } from '../../../common/languages/languageConfigurationRegistry.js';
import { GhostText, ghostTextOrReplacementEquals } from './ghostText.js';
import { InlineCompletionsSource } from './inlineCompletionsSource.js';
import { addPositions, lengthOfText } from './utils.js';
import { SnippetController2 } from '../../snippet/browser/snippetController2.js';
import { ICommandService } from '../../../../platform/commands/common/commands.js';
import { IInstantiationService } from '../../../../platform/instantiation/common/instantiation.js';
export var VersionIdChangeReason;
(function (VersionIdChangeReason) {
    VersionIdChangeReason[VersionIdChangeReason["Undo"] = 0] = "Undo";
    VersionIdChangeReason[VersionIdChangeReason["Redo"] = 1] = "Redo";
    VersionIdChangeReason[VersionIdChangeReason["AcceptWord"] = 2] = "AcceptWord";
    VersionIdChangeReason[VersionIdChangeReason["Other"] = 3] = "Other";
})(VersionIdChangeReason || (VersionIdChangeReason = {}));
let InlineCompletionsModel = class InlineCompletionsModel extends Disposable {
    get isAcceptingPartially() { return this._isAcceptingPartially; }
    constructor(textModel, selectedSuggestItem, cursorPosition, textModelVersionId, _debounceValue, _suggestPreviewEnabled, _suggestPreviewMode, _inlineSuggestMode, _enabled, _instantiationService, _commandService, _languageConfigurationService) {
        super();
        this.textModel = textModel;
        this.selectedSuggestItem = selectedSuggestItem;
        this.cursorPosition = cursorPosition;
        this.textModelVersionId = textModelVersionId;
        this._debounceValue = _debounceValue;
        this._suggestPreviewEnabled = _suggestPreviewEnabled;
        this._suggestPreviewMode = _suggestPreviewMode;
        this._inlineSuggestMode = _inlineSuggestMode;
        this._enabled = _enabled;
        this._instantiationService = _instantiationService;
        this._commandService = _commandService;
        this._languageConfigurationService = _languageConfigurationService;
        this._source = this._register(this._instantiationService.createInstance(InlineCompletionsSource, this.textModel, this.textModelVersionId, this._debounceValue));
        this._isActive = observableValue(this, false);
        this._forceUpdateSignal = observableSignal('forceUpdate');
        // We use a semantic id to keep the same inline completion selected even if the provider reorders the completions.
        this._selectedInlineCompletionId = observableValue(this, undefined);
        this._isAcceptingPartially = false;
        this._preserveCurrentCompletionReasons = new Set([
            VersionIdChangeReason.Redo,
            VersionIdChangeReason.Undo,
            VersionIdChangeReason.AcceptWord,
        ]);
        this._fetchInlineCompletions = derivedHandleChanges({
            owner: this,
            createEmptyChangeSummary: () => ({
                preserveCurrentCompletion: false,
                inlineCompletionTriggerKind: InlineCompletionTriggerKind.Automatic
            }),
            handleChange: (ctx, changeSummary) => {
                /** @description fetch inline completions */
                if (ctx.didChange(this.textModelVersionId) && this._preserveCurrentCompletionReasons.has(ctx.change)) {
                    changeSummary.preserveCurrentCompletion = true;
                }
                else if (ctx.didChange(this._forceUpdateSignal)) {
                    changeSummary.inlineCompletionTriggerKind = ctx.change;
                }
                return true;
            },
        }, (reader, changeSummary) => {
            this._forceUpdateSignal.read(reader);
            const shouldUpdate = (this._enabled.read(reader) && this.selectedSuggestItem.read(reader)) || this._isActive.read(reader);
            if (!shouldUpdate) {
                this._source.cancelUpdate();
                return undefined;
            }
            this.textModelVersionId.read(reader); // Refetch on text change
            const itemToPreserveCandidate = this.selectedInlineCompletion.get();
            const itemToPreserve = changeSummary.preserveCurrentCompletion || itemToPreserveCandidate?.forwardStable
                ? itemToPreserveCandidate : undefined;
            const suggestWidgetInlineCompletions = this._source.suggestWidgetInlineCompletions.get();
            const suggestItem = this.selectedSuggestItem.read(reader);
            if (suggestWidgetInlineCompletions && !suggestItem) {
                const inlineCompletions = this._source.inlineCompletions.get();
                transaction(tx => {
                    /** @description Seed inline completions with (newer) suggest widget inline completions */
                    if (!inlineCompletions || suggestWidgetInlineCompletions.request.versionId > inlineCompletions.request.versionId) {
                        this._source.inlineCompletions.set(suggestWidgetInlineCompletions.clone(), tx);
                    }
                    this._source.clearSuggestWidgetInlineCompletions(tx);
                });
            }
            const cursorPosition = this.cursorPosition.read(reader);
            const context = {
                triggerKind: changeSummary.inlineCompletionTriggerKind,
                selectedSuggestionInfo: suggestItem?.toSelectedSuggestionInfo(),
            };
            return this._source.fetch(cursorPosition, context, itemToPreserve);
        });
        this._filteredInlineCompletionItems = derived(this, reader => {
            const c = this._source.inlineCompletions.read(reader);
            if (!c) {
                return [];
            }
            const cursorPosition = this.cursorPosition.read(reader);
            const filteredCompletions = c.inlineCompletions.filter(c => c.isVisible(this.textModel, cursorPosition, reader));
            return filteredCompletions;
        });
        this.selectedInlineCompletionIndex = derived(this, (reader) => {
            const selectedInlineCompletionId = this._selectedInlineCompletionId.read(reader);
            const filteredCompletions = this._filteredInlineCompletionItems.read(reader);
            const idx = this._selectedInlineCompletionId === undefined ? -1
                : filteredCompletions.findIndex(v => v.semanticId === selectedInlineCompletionId);
            if (idx === -1) {
                // Reset the selection so that the selection does not jump back when it appears again
                this._selectedInlineCompletionId.set(undefined, undefined);
                return 0;
            }
            return idx;
        });
        this.selectedInlineCompletion = derived(this, (reader) => {
            const filteredCompletions = this._filteredInlineCompletionItems.read(reader);
            const idx = this.selectedInlineCompletionIndex.read(reader);
            return filteredCompletions[idx];
        });
        this.lastTriggerKind = this._source.inlineCompletions.map(this, v => v?.request.context.triggerKind);
        this.inlineCompletionsCount = derived(this, reader => {
            if (this.lastTriggerKind.read(reader) === InlineCompletionTriggerKind.Explicit) {
                return this._filteredInlineCompletionItems.read(reader).length;
            }
            else {
                return undefined;
            }
        });
        this.state = derivedOpts({
            owner: this,
            equalityComparer: (a, b) => {
                if (!a || !b) {
                    return a === b;
                }
                return ghostTextOrReplacementEquals(a.ghostText, b.ghostText)
                    && a.inlineCompletion === b.inlineCompletion
                    && a.suggestItem === b.suggestItem;
            }
        }, (reader) => {
            const model = this.textModel;
            const suggestItem = this.selectedSuggestItem.read(reader);
            if (suggestItem) {
                const suggestCompletion = suggestItem.toSingleTextEdit().removeCommonPrefix(model);
                const augmentedCompletion = this._computeAugmentedCompletion(suggestCompletion, reader);
                const isSuggestionPreviewEnabled = this._suggestPreviewEnabled.read(reader);
                if (!isSuggestionPreviewEnabled && !augmentedCompletion) {
                    return undefined;
                }
                const edit = augmentedCompletion?.edit ?? suggestCompletion;
                const editPreviewLength = augmentedCompletion ? augmentedCompletion.edit.text.length - suggestCompletion.text.length : 0;
                const mode = this._suggestPreviewMode.read(reader);
                const cursor = this.cursorPosition.read(reader);
                const newGhostText = edit.computeGhostText(model, mode, cursor, editPreviewLength);
                // Show an invisible ghost text to reserve space
                const ghostText = newGhostText ?? new GhostText(edit.range.endLineNumber, []);
                return { ghostText, inlineCompletion: augmentedCompletion?.completion, suggestItem };
            }
            else {
                if (!this._isActive.read(reader)) {
                    return undefined;
                }
                const item = this.selectedInlineCompletion.read(reader);
                if (!item) {
                    return undefined;
                }
                const replacement = item.toSingleTextEdit(reader);
                const mode = this._inlineSuggestMode.read(reader);
                const cursor = this.cursorPosition.read(reader);
                const ghostText = replacement.computeGhostText(model, mode, cursor);
                return ghostText ? { ghostText, inlineCompletion: item, suggestItem: undefined } : undefined;
            }
        });
        this.ghostText = derivedOpts({
            owner: this,
            equalityComparer: ghostTextOrReplacementEquals
        }, reader => {
            const v = this.state.read(reader);
            if (!v) {
                return undefined;
            }
            return v.ghostText;
        });
        this._register(recomputeInitiallyAndOnChange(this._fetchInlineCompletions));
        let lastItem = undefined;
        this._register(autorun(reader => {
            /** @description call handleItemDidShow */
            const item = this.state.read(reader);
            const completion = item?.inlineCompletion;
            if (completion?.semanticId !== lastItem?.semanticId) {
                lastItem = completion;
                if (completion) {
                    const i = completion.inlineCompletion;
                    const src = i.source;
                    src.provider.handleItemDidShow?.(src.inlineCompletions, i.sourceInlineCompletion, i.insertText);
                }
            }
        }));
    }
    async trigger(tx) {
        this._isActive.set(true, tx);
        await this._fetchInlineCompletions.get();
    }
    async triggerExplicitly(tx) {
        subtransaction(tx, tx => {
            this._isActive.set(true, tx);
            this._forceUpdateSignal.trigger(tx, InlineCompletionTriggerKind.Explicit);
        });
        await this._fetchInlineCompletions.get();
    }
    stop(tx) {
        subtransaction(tx, tx => {
            this._isActive.set(false, tx);
            this._source.clear(tx);
        });
    }
    _computeAugmentedCompletion(suggestCompletion, reader) {
        const model = this.textModel;
        const suggestWidgetInlineCompletions = this._source.suggestWidgetInlineCompletions.read(reader);
        const candidateInlineCompletions = suggestWidgetInlineCompletions
            ? suggestWidgetInlineCompletions.inlineCompletions
            : [this.selectedInlineCompletion.read(reader)].filter(isDefined);
        const augmentedCompletion = mapFindFirst(candidateInlineCompletions, completion => {
            let r = completion.toSingleTextEdit(reader);
            r = r.removeCommonPrefix(model, Range.fromPositions(r.range.getStartPosition(), suggestCompletion.range.getEndPosition()));
            return r.augments(suggestCompletion) ? { edit: r, completion } : undefined;
        });
        return augmentedCompletion;
    }
    async _deltaSelectedInlineCompletionIndex(delta) {
        await this.triggerExplicitly();
        const completions = this._filteredInlineCompletionItems.get() || [];
        if (completions.length > 0) {
            const newIdx = (this.selectedInlineCompletionIndex.get() + delta + completions.length) % completions.length;
            this._selectedInlineCompletionId.set(completions[newIdx].semanticId, undefined);
        }
        else {
            this._selectedInlineCompletionId.set(undefined, undefined);
        }
    }
    async next() {
        await this._deltaSelectedInlineCompletionIndex(1);
    }
    async previous() {
        await this._deltaSelectedInlineCompletionIndex(-1);
    }
    async accept(editor) {
        if (editor.getModel() !== this.textModel) {
            throw new BugIndicatingError();
        }
        const state = this.state.get();
        if (!state || state.ghostText.isEmpty() || !state.inlineCompletion) {
            return;
        }
        const completion = state.inlineCompletion.toInlineCompletion(undefined);
        editor.pushUndoStop();
        if (completion.snippetInfo) {
            editor.executeEdits('inlineSuggestion.accept', [
                EditOperation.replaceMove(completion.range, ''),
                ...completion.additionalTextEdits
            ]);
            editor.setPosition(completion.snippetInfo.range.getStartPosition());
            SnippetController2.get(editor)?.insert(completion.snippetInfo.snippet, { undoStopBefore: false });
        }
        else {
            editor.executeEdits('inlineSuggestion.accept', [
                EditOperation.replaceMove(completion.range, completion.insertText),
                ...completion.additionalTextEdits
            ]);
        }
        if (completion.command) {
            // Make sure the completion list will not be disposed.
            completion.source.addRef();
        }
        // Reset before invoking the command, since the command might cause a follow up trigger.
        transaction(tx => {
            this._source.clear(tx);
            // Potentially, isActive will get set back to true by the typing or accept inline suggest event
            // if automatic inline suggestions are enabled.
            this._isActive.set(false, tx);
        });
        if (completion.command) {
            await this._commandService
                .executeCommand(completion.command.id, ...(completion.command.arguments || []))
                .then(undefined, onUnexpectedExternalError);
            completion.source.removeRef();
        }
    }
    async acceptNextWord(editor) {
        await this._acceptNext(editor, (pos, text) => {
            const langId = this.textModel.getLanguageIdAtPosition(pos.lineNumber, pos.column);
            const config = this._languageConfigurationService.getLanguageConfiguration(langId);
            const wordRegExp = new RegExp(config.wordDefinition.source, config.wordDefinition.flags.replace('g', ''));
            const m1 = text.match(wordRegExp);
            let acceptUntilIndexExclusive = 0;
            if (m1 && m1.index !== undefined) {
                if (m1.index === 0) {
                    acceptUntilIndexExclusive = m1[0].length;
                }
                else {
                    acceptUntilIndexExclusive = m1.index;
                }
            }
            else {
                acceptUntilIndexExclusive = text.length;
            }
            const wsRegExp = /\s+/g;
            const m2 = wsRegExp.exec(text);
            if (m2 && m2.index !== undefined) {
                if (m2.index + m2[0].length < acceptUntilIndexExclusive) {
                    acceptUntilIndexExclusive = m2.index + m2[0].length;
                }
            }
            return acceptUntilIndexExclusive;
        });
    }
    async acceptNextLine(editor) {
        await this._acceptNext(editor, (pos, text) => {
            const m = text.match(/\n/);
            if (m && m.index !== undefined) {
                return m.index + 1;
            }
            return text.length;
        });
    }
    async _acceptNext(editor, getAcceptUntilIndex) {
        if (editor.getModel() !== this.textModel) {
            throw new BugIndicatingError();
        }
        const state = this.state.get();
        if (!state || state.ghostText.isEmpty() || !state.inlineCompletion) {
            return;
        }
        const ghostText = state.ghostText;
        const completion = state.inlineCompletion.toInlineCompletion(undefined);
        if (completion.snippetInfo || completion.filterText !== completion.insertText) {
            // not in WYSIWYG mode, partial commit might change completion, thus it is not supported
            await this.accept(editor);
            return;
        }
        const firstPart = ghostText.parts[0];
        const position = new Position(ghostText.lineNumber, firstPart.column);
        const line = firstPart.lines.join('\n');
        const acceptUntilIndexExclusive = getAcceptUntilIndex(position, line);
        if (acceptUntilIndexExclusive === line.length && ghostText.parts.length === 1) {
            this.accept(editor);
            return;
        }
        const partialText = line.substring(0, acceptUntilIndexExclusive);
        // Executing the edit might free the completion, so we have to hold a reference on it.
        completion.source.addRef();
        try {
            this._isAcceptingPartially = true;
            try {
                editor.pushUndoStop();
                editor.executeEdits('inlineSuggestion.accept', [
                    EditOperation.replace(Range.fromPositions(position), partialText),
                ]);
                const length = lengthOfText(partialText);
                editor.setPosition(addPositions(position, length));
            }
            finally {
                this._isAcceptingPartially = false;
            }
            if (completion.source.provider.handlePartialAccept) {
                const acceptedRange = Range.fromPositions(completion.range.getStartPosition(), addPositions(position, lengthOfText(partialText)));
                // This assumes that the inline completion and the model use the same EOL style.
                const text = editor.getModel().getValueInRange(acceptedRange, 1 /* EndOfLinePreference.LF */);
                completion.source.provider.handlePartialAccept(completion.source.inlineCompletions, completion.sourceInlineCompletion, text.length);
            }
        }
        finally {
            completion.source.removeRef();
        }
    }
    handleSuggestAccepted(item) {
        const itemEdit = item.toSingleTextEdit().removeCommonPrefix(this.textModel);
        const augmentedCompletion = this._computeAugmentedCompletion(itemEdit, undefined);
        if (!augmentedCompletion) {
            return;
        }
        const inlineCompletion = augmentedCompletion.completion.inlineCompletion;
        inlineCompletion.source.provider.handlePartialAccept?.(inlineCompletion.source.inlineCompletions, inlineCompletion.sourceInlineCompletion, itemEdit.text.length);
    }
};
InlineCompletionsModel = __decorate([
    __param(9, IInstantiationService),
    __param(10, ICommandService),
    __param(11, ILanguageConfigurationService)
], InlineCompletionsModel);
export { InlineCompletionsModel };
