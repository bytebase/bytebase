/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import * as nls from '../../nls.js';
import { RawContextKey } from '../../platform/contextkey/common/contextkey.js';
export var EditorContextKeys;
(function (EditorContextKeys) {
    EditorContextKeys.editorSimpleInput = new RawContextKey('editorSimpleInput', false, true);
    /**
     * A context key that is set when the editor's text has focus (cursor is blinking).
     * Is false when focus is in simple editor widgets (repl input, scm commit input).
     */
    EditorContextKeys.editorTextFocus = new RawContextKey('editorTextFocus', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorTextFocus', "Whether the editor text has focus (cursor is blinking)"));
    /**
     * A context key that is set when the editor's text or an editor's widget has focus.
     */
    EditorContextKeys.focus = new RawContextKey('editorFocus', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorFocus', "Whether the editor or an editor widget has focus (e.g. focus is in the find widget)"));
    /**
     * A context key that is set when any editor input has focus (regular editor, repl input...).
     */
    EditorContextKeys.textInputFocus = new RawContextKey('textInputFocus', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'textInputFocus', "Whether an editor or a rich text input has focus (cursor is blinking)"));
    EditorContextKeys.readOnly = new RawContextKey('editorReadonly', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorReadonly', "Whether the editor is read-only"));
    EditorContextKeys.inDiffEditor = new RawContextKey('inDiffEditor', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'inDiffEditor', "Whether the context is a diff editor"));
    EditorContextKeys.isEmbeddedDiffEditor = new RawContextKey('isEmbeddedDiffEditor', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'isEmbeddedDiffEditor', "Whether the context is an embedded diff editor"));
    EditorContextKeys.inMultiDiffEditor = new RawContextKey('inMultiDiffEditor', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'inMultiDiffEditor', "Whether the context is a multi diff editor"));
    EditorContextKeys.multiDiffEditorAllCollapsed = new RawContextKey('multiDiffEditorAllCollapsed', undefined, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'multiDiffEditorAllCollapsed', "Whether all files in multi diff editor are collapsed"));
    EditorContextKeys.hasChanges = new RawContextKey('diffEditorHasChanges', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'diffEditorHasChanges', "Whether the diff editor has changes"));
    EditorContextKeys.comparingMovedCode = new RawContextKey('comparingMovedCode', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'comparingMovedCode', "Whether a moved code block is selected for comparison"));
    EditorContextKeys.accessibleDiffViewerVisible = new RawContextKey('accessibleDiffViewerVisible', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'accessibleDiffViewerVisible', "Whether the accessible diff viewer is visible"));
    EditorContextKeys.diffEditorRenderSideBySideInlineBreakpointReached = new RawContextKey('diffEditorRenderSideBySideInlineBreakpointReached', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'diffEditorRenderSideBySideInlineBreakpointReached', "Whether the diff editor render side by side inline breakpoint is reached"));
    EditorContextKeys.columnSelection = new RawContextKey('editorColumnSelection', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorColumnSelection', "Whether `editor.columnSelection` is enabled"));
    EditorContextKeys.writable = EditorContextKeys.readOnly.toNegated();
    EditorContextKeys.hasNonEmptySelection = new RawContextKey('editorHasSelection', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasSelection', "Whether the editor has text selected"));
    EditorContextKeys.hasOnlyEmptySelection = EditorContextKeys.hasNonEmptySelection.toNegated();
    EditorContextKeys.hasMultipleSelections = new RawContextKey('editorHasMultipleSelections', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasMultipleSelections', "Whether the editor has multiple selections"));
    EditorContextKeys.hasSingleSelection = EditorContextKeys.hasMultipleSelections.toNegated();
    EditorContextKeys.tabMovesFocus = new RawContextKey('editorTabMovesFocus', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorTabMovesFocus', "Whether `Tab` will move focus out of the editor"));
    EditorContextKeys.tabDoesNotMoveFocus = EditorContextKeys.tabMovesFocus.toNegated();
    EditorContextKeys.isInWalkThroughSnippet = new RawContextKey('isInEmbeddedEditor', false, true);
    EditorContextKeys.canUndo = new RawContextKey('canUndo', false, true);
    EditorContextKeys.canRedo = new RawContextKey('canRedo', false, true);
    EditorContextKeys.hoverVisible = new RawContextKey('editorHoverVisible', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHoverVisible', "Whether the editor hover is visible"));
    EditorContextKeys.hoverFocused = new RawContextKey('editorHoverFocused', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHoverFocused', "Whether the editor hover is focused"));
    EditorContextKeys.stickyScrollFocused = new RawContextKey('stickyScrollFocused', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'stickyScrollFocused', "Whether the sticky scroll is focused"));
    EditorContextKeys.stickyScrollVisible = new RawContextKey('stickyScrollVisible', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'stickyScrollVisible', "Whether the sticky scroll is visible"));
    EditorContextKeys.standaloneColorPickerVisible = new RawContextKey('standaloneColorPickerVisible', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'standaloneColorPickerVisible', "Whether the standalone color picker is visible"));
    EditorContextKeys.standaloneColorPickerFocused = new RawContextKey('standaloneColorPickerFocused', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'standaloneColorPickerFocused', "Whether the standalone color picker is focused"));
    /**
     * A context key that is set when an editor is part of a larger editor, like notebooks or
     * (future) a diff editor
     */
    EditorContextKeys.inCompositeEditor = new RawContextKey('inCompositeEditor', undefined, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'inCompositeEditor', "Whether the editor is part of a larger editor (e.g. notebooks)"));
    EditorContextKeys.notInCompositeEditor = EditorContextKeys.inCompositeEditor.toNegated();
    // -- mode context keys
    EditorContextKeys.languageId = new RawContextKey('editorLangId', '', nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorLangId', "The language identifier of the editor"));
    EditorContextKeys.hasCompletionItemProvider = new RawContextKey('editorHasCompletionItemProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasCompletionItemProvider', "Whether the editor has a completion item provider"));
    EditorContextKeys.hasCodeActionsProvider = new RawContextKey('editorHasCodeActionsProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasCodeActionsProvider', "Whether the editor has a code actions provider"));
    EditorContextKeys.hasCodeLensProvider = new RawContextKey('editorHasCodeLensProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasCodeLensProvider', "Whether the editor has a code lens provider"));
    EditorContextKeys.hasDefinitionProvider = new RawContextKey('editorHasDefinitionProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasDefinitionProvider', "Whether the editor has a definition provider"));
    EditorContextKeys.hasDeclarationProvider = new RawContextKey('editorHasDeclarationProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasDeclarationProvider', "Whether the editor has a declaration provider"));
    EditorContextKeys.hasImplementationProvider = new RawContextKey('editorHasImplementationProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasImplementationProvider', "Whether the editor has an implementation provider"));
    EditorContextKeys.hasTypeDefinitionProvider = new RawContextKey('editorHasTypeDefinitionProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasTypeDefinitionProvider', "Whether the editor has a type definition provider"));
    EditorContextKeys.hasHoverProvider = new RawContextKey('editorHasHoverProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasHoverProvider', "Whether the editor has a hover provider"));
    EditorContextKeys.hasDocumentHighlightProvider = new RawContextKey('editorHasDocumentHighlightProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasDocumentHighlightProvider', "Whether the editor has a document highlight provider"));
    EditorContextKeys.hasDocumentSymbolProvider = new RawContextKey('editorHasDocumentSymbolProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasDocumentSymbolProvider', "Whether the editor has a document symbol provider"));
    EditorContextKeys.hasReferenceProvider = new RawContextKey('editorHasReferenceProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasReferenceProvider', "Whether the editor has a reference provider"));
    EditorContextKeys.hasRenameProvider = new RawContextKey('editorHasRenameProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasRenameProvider', "Whether the editor has a rename provider"));
    EditorContextKeys.hasSignatureHelpProvider = new RawContextKey('editorHasSignatureHelpProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasSignatureHelpProvider', "Whether the editor has a signature help provider"));
    EditorContextKeys.hasInlayHintsProvider = new RawContextKey('editorHasInlayHintsProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasInlayHintsProvider', "Whether the editor has an inline hints provider"));
    // -- mode context keys: formatting
    EditorContextKeys.hasDocumentFormattingProvider = new RawContextKey('editorHasDocumentFormattingProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasDocumentFormattingProvider', "Whether the editor has a document formatting provider"));
    EditorContextKeys.hasDocumentSelectionFormattingProvider = new RawContextKey('editorHasDocumentSelectionFormattingProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasDocumentSelectionFormattingProvider', "Whether the editor has a document selection formatting provider"));
    EditorContextKeys.hasMultipleDocumentFormattingProvider = new RawContextKey('editorHasMultipleDocumentFormattingProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasMultipleDocumentFormattingProvider', "Whether the editor has multiple document formatting providers"));
    EditorContextKeys.hasMultipleDocumentSelectionFormattingProvider = new RawContextKey('editorHasMultipleDocumentSelectionFormattingProvider', false, nls.localizeWithPath('vs/editor/common/editorContextKeys', 'editorHasMultipleDocumentSelectionFormattingProvider', "Whether the editor has multiple document selection formatting providers"));
})(EditorContextKeys || (EditorContextKeys = {}));
