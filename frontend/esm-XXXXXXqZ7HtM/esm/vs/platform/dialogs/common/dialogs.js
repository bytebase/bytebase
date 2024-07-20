/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { basename } from '../../../base/common/resources.js';
import Severity from '../../../base/common/severity.js';
import { localizeWithPath } from '../../../nls.js';
import { createDecorator } from '../../instantiation/common/instantiation.js';
import { mnemonicButtonLabel } from '../../../base/common/labels.js';
import { isLinux, isMacintosh, isWindows } from '../../../base/common/platform.js';
import { deepClone } from '../../../base/common/objects.js';
export const IDialogService = createDecorator('dialogService');
var DialogKind;
(function (DialogKind) {
    DialogKind[DialogKind["Confirmation"] = 1] = "Confirmation";
    DialogKind[DialogKind["Prompt"] = 2] = "Prompt";
    DialogKind[DialogKind["Input"] = 3] = "Input";
})(DialogKind || (DialogKind = {}));
export class AbstractDialogHandler {
    getConfirmationButtons(dialog) {
        return this.getButtons(dialog, DialogKind.Confirmation);
    }
    getPromptButtons(dialog) {
        return this.getButtons(dialog, DialogKind.Prompt);
    }
    getInputButtons(dialog) {
        return this.getButtons(dialog, DialogKind.Input);
    }
    getButtons(dialog, kind) {
        // We put buttons in the order of "default" button first and "cancel"
        // button last. There maybe later processing when presenting the buttons
        // based on OS standards.
        const buttons = [];
        switch (kind) {
            case DialogKind.Confirmation: {
                const confirmationDialog = dialog;
                if (confirmationDialog.primaryButton) {
                    buttons.push(confirmationDialog.primaryButton);
                }
                else {
                    buttons.push(localizeWithPath('vs/platform/dialogs/common/dialogs', { key: 'yesButton', comment: ['&& denotes a mnemonic'] }, "&&Yes"));
                }
                if (confirmationDialog.cancelButton) {
                    buttons.push(confirmationDialog.cancelButton);
                }
                else {
                    buttons.push(localizeWithPath('vs/platform/dialogs/common/dialogs', 'cancelButton', "Cancel"));
                }
                break;
            }
            case DialogKind.Prompt: {
                const promptDialog = dialog;
                if (Array.isArray(promptDialog.buttons) && promptDialog.buttons.length > 0) {
                    buttons.push(...promptDialog.buttons.map(button => button.label));
                }
                if (promptDialog.cancelButton) {
                    if (promptDialog.cancelButton === true) {
                        buttons.push(localizeWithPath('vs/platform/dialogs/common/dialogs', 'cancelButton', "Cancel"));
                    }
                    else if (typeof promptDialog.cancelButton === 'string') {
                        buttons.push(promptDialog.cancelButton);
                    }
                    else {
                        if (promptDialog.cancelButton.label) {
                            buttons.push(promptDialog.cancelButton.label);
                        }
                        else {
                            buttons.push(localizeWithPath('vs/platform/dialogs/common/dialogs', 'cancelButton', "Cancel"));
                        }
                    }
                }
                if (buttons.length === 0) {
                    buttons.push(localizeWithPath('vs/platform/dialogs/common/dialogs', { key: 'okButton', comment: ['&& denotes a mnemonic'] }, "&&OK"));
                }
                break;
            }
            case DialogKind.Input: {
                const inputDialog = dialog;
                if (inputDialog.primaryButton) {
                    buttons.push(inputDialog.primaryButton);
                }
                else {
                    buttons.push(localizeWithPath('vs/platform/dialogs/common/dialogs', { key: 'okButton', comment: ['&& denotes a mnemonic'] }, "&&OK"));
                }
                if (inputDialog.cancelButton) {
                    buttons.push(inputDialog.cancelButton);
                }
                else {
                    buttons.push(localizeWithPath('vs/platform/dialogs/common/dialogs', 'cancelButton', "Cancel"));
                }
                break;
            }
        }
        return buttons;
    }
    getDialogType(type) {
        if (typeof type === 'string') {
            return type;
        }
        if (typeof type === 'number') {
            return (type === Severity.Info) ? 'info' : (type === Severity.Error) ? 'error' : (type === Severity.Warning) ? 'warning' : 'none';
        }
        return undefined;
    }
    getPromptResult(prompt, buttonIndex, checkboxChecked) {
        const promptButtons = [...(prompt.buttons ?? [])];
        if (prompt.cancelButton && typeof prompt.cancelButton !== 'string' && typeof prompt.cancelButton !== 'boolean') {
            promptButtons.push(prompt.cancelButton);
        }
        let result = promptButtons[buttonIndex]?.run({ checkboxChecked });
        if (!(result instanceof Promise)) {
            result = Promise.resolve(result);
        }
        return { result, checkboxChecked };
    }
}
export const IFileDialogService = createDecorator('fileDialogService');
const MAX_CONFIRM_FILES = 10;
export function getFileNamesMessage(fileNamesOrResources) {
    const message = [];
    message.push(...fileNamesOrResources.slice(0, MAX_CONFIRM_FILES).map(fileNameOrResource => typeof fileNameOrResource === 'string' ? fileNameOrResource : basename(fileNameOrResource)));
    if (fileNamesOrResources.length > MAX_CONFIRM_FILES) {
        if (fileNamesOrResources.length - MAX_CONFIRM_FILES === 1) {
            message.push(localizeWithPath('vs/platform/dialogs/common/dialogs', 'moreFile', "...1 additional file not shown"));
        }
        else {
            message.push(localizeWithPath('vs/platform/dialogs/common/dialogs', 'moreFiles', "...{0} additional files not shown", fileNamesOrResources.length - MAX_CONFIRM_FILES));
        }
    }
    message.push('');
    return message.join('\n');
}
/**
 * A utility method to ensure the options for the message box dialog
 * are using properties that are consistent across all platforms and
 * specific to the platform where necessary.
 */
export function massageMessageBoxOptions(options, productService) {
    const massagedOptions = deepClone(options);
    let buttons = (massagedOptions.buttons ?? []).map(button => mnemonicButtonLabel(button));
    let buttonIndeces = (options.buttons || []).map((button, index) => index);
    let defaultId = 0; // by default the first button is default button
    let cancelId = massagedOptions.cancelId ?? buttons.length - 1; // by default the last button is cancel button
    // Apply HIG per OS when more than one button is used
    if (buttons.length > 1) {
        const cancelButton = typeof cancelId === 'number' ? buttons[cancelId] : undefined;
        if (isLinux || isMacintosh) {
            // Linux: the GNOME HIG (https://developer.gnome.org/hig/patterns/feedback/dialogs.html?highlight=dialog)
            // recommend the following:
            // "Always ensure that the cancel button appears first, before the affirmative button. In left-to-right
            //  locales, this is on the left. This button order ensures that users become aware of, and are reminded
            //  of, the ability to cancel prior to encountering the affirmative button."
            //
            // Electron APIs do not reorder buttons for us, so we ensure a reverse order of buttons and a position
            // of the cancel button (if provided) that matches the HIG
            // macOS: the HIG (https://developer.apple.com/design/human-interface-guidelines/components/presentation/alerts)
            // recommend the following:
            // "Place buttons where people expect. In general, place the button people are most likely to choose on the trailing side in a
            //  row of buttons or at the top in a stack of buttons. Always place the default button on the trailing side of a row or at the
            //  top of a stack. Cancel buttons are typically on the leading side of a row or at the bottom of a stack."
            //
            // However: it seems that older macOS versions where 3 buttons were presented in a row differ from this
            // recommendation. In fact, cancel buttons were placed to the left of the default button and secondary
            // buttons on the far left. To support these older macOS versions we have to manually shuffle the cancel
            // button in the same way as we do on Linux. This will not have any impact on newer macOS versions where
            // shuffling is done for us.
            if (typeof cancelButton === 'string' && buttons.length > 1 && cancelId !== 1) {
                buttons.splice(cancelId, 1);
                buttons.splice(1, 0, cancelButton);
                const cancelButtonIndex = buttonIndeces[cancelId];
                buttonIndeces.splice(cancelId, 1);
                buttonIndeces.splice(1, 0, cancelButtonIndex);
                cancelId = 1;
            }
            if (isLinux && buttons.length > 1) {
                buttons = buttons.reverse();
                buttonIndeces = buttonIndeces.reverse();
                defaultId = buttons.length - 1;
                if (typeof cancelButton === 'string') {
                    cancelId = defaultId - 1;
                }
            }
        }
        else if (isWindows) {
            // Windows: the HIG (https://learn.microsoft.com/en-us/windows/win32/uxguide/win-dialog-box)
            // recommend the following:
            // "One of the following sets of concise commands: Yes/No, Yes/No/Cancel, [Do it]/Cancel,
            //  [Do it]/[Don't do it], [Do it]/[Don't do it]/Cancel."
            //
            // Electron APIs do not reorder buttons for us, so we ensure the position of the cancel button
            // (if provided) that matches the HIG
            if (typeof cancelButton === 'string' && buttons.length > 1 && cancelId !== buttons.length - 1 /* last action */) {
                buttons.splice(cancelId, 1);
                buttons.push(cancelButton);
                const buttonIndex = buttonIndeces[cancelId];
                buttonIndeces.splice(cancelId, 1);
                buttonIndeces.push(buttonIndex);
                cancelId = buttons.length - 1;
            }
        }
    }
    massagedOptions.buttons = buttons;
    massagedOptions.defaultId = defaultId;
    massagedOptions.cancelId = cancelId;
    massagedOptions.noLink = true;
    massagedOptions.title = massagedOptions.title || productService.nameLong;
    return {
        options: massagedOptions,
        buttonIndeces
    };
}
