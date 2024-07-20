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
import { ContextView } from '../../../base/browser/ui/contextview/contextview.js';
import { Disposable, toDisposable } from '../../../base/common/lifecycle.js';
import { ILayoutService } from '../../layout/browser/layoutService.js';
import { getWindow } from '../../../base/browser/dom.js';
let ContextViewService = class ContextViewService extends Disposable {
    constructor(layoutService) {
        super();
        this.layoutService = layoutService;
        this.currentViewDisposable = Disposable.None;
        this.contextView = this._register(new ContextView(this.layoutService.mainContainer, 1 /* ContextViewDOMPosition.ABSOLUTE */));
        this.layout();
        this._register(layoutService.onDidLayoutContainer(() => this.layout()));
    }
    // ContextView
    showContextView(delegate, container, shadowRoot) {
        let domPosition;
        if (container) {
            if (container === this.layoutService.getContainer(getWindow(container))) {
                domPosition = 1 /* ContextViewDOMPosition.ABSOLUTE */;
            }
            else if (shadowRoot) {
                domPosition = 3 /* ContextViewDOMPosition.FIXED_SHADOW */;
            }
            else {
                domPosition = 2 /* ContextViewDOMPosition.FIXED */;
            }
        }
        else {
            domPosition = 1 /* ContextViewDOMPosition.ABSOLUTE */;
        }
        this.contextView.setContainer(container ?? this.layoutService.activeContainer, domPosition);
        this.contextView.show(delegate);
        const disposable = toDisposable(() => {
            if (this.currentViewDisposable === disposable) {
                this.hideContextView();
            }
        });
        this.currentViewDisposable = disposable;
        return disposable;
    }
    getContextViewElement() {
        return this.contextView.getViewElement();
    }
    layout() {
        this.contextView.layout();
    }
    hideContextView(data) {
        this.contextView.hide(data);
    }
    dispose() {
        super.dispose();
        this.currentViewDisposable.dispose();
        this.currentViewDisposable = Disposable.None;
    }
};
ContextViewService = __decorate([
    __param(0, ILayoutService)
], ContextViewService);
export { ContextViewService };
