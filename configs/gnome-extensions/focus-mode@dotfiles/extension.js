// Focus Mode — fullscreen envia janela para workspace exclusivo.
//
// Quando uma janela entra em fullscreen (F11), ela é movida para um
// workspace novo e vazio. Quando sai do fullscreen, volta ao workspace
// original. Resultado: F11 = tela cheia + desktop exclusivo (como macOS).
//
// Requer: dynamic-workspaces = true (configurado pelo módulo gnome-focus-mode).

import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

export default class FocusModeExtension extends Extension {
    enable() {
        this._originalWorkspaces = new Map();
        this._windowSignals = new Map();

        // Rastreia janelas existentes
        for (const actor of global.get_window_actors()) {
            this._trackWindow(actor.meta_window);
        }

        // Rastreia novas janelas
        this._createdId = global.display.connect(
            'window-created',
            (_display, win) => this._trackWindow(win),
        );
    }

    _trackWindow(win) {
        if (this._windowSignals.has(win)) return;

        const id = win.connect('notify::fullscreen', () => {
            this._onFullscreenChanged(win);
        });
        this._windowSignals.set(win, id);

        // Limpa quando a janela fechar
        const destroyId = win.connect('unmanaged', () => {
            this._originalWorkspaces.delete(win);
            this._windowSignals.delete(win);
        });
        // Guarda ambos os sinais (simplificação: só desconecta o principal)
    }

    _onFullscreenChanged(win) {
        const wsManager = global.workspace_manager;

        if (win.is_fullscreen()) {
            // Salva workspace original
            this._originalWorkspaces.set(win, win.get_workspace().index());

            // Cria workspace novo e move a janela
            const newWs = wsManager.append_new_workspace(
                false, global.get_current_time(),
            );
            win.change_workspace(newWs);
            newWs.activate(global.get_current_time());
        } else {
            // Volta pro workspace original
            const origIdx = this._originalWorkspaces.get(win);
            if (origIdx === undefined) return;

            const nWorkspaces = wsManager.get_n_workspaces();
            const targetIdx = Math.min(origIdx, nWorkspaces - 1);
            const origWs = wsManager.get_workspace_by_index(targetIdx);

            win.change_workspace(origWs);
            origWs.activate(global.get_current_time());
            this._originalWorkspaces.delete(win);
        }
    }

    disable() {
        if (this._createdId) {
            global.display.disconnect(this._createdId);
            this._createdId = null;
        }

        for (const [win, id] of this._windowSignals) {
            try { win.disconnect(id); } catch (_) { /* janela já fechada */ }
        }
        this._windowSignals.clear();
        this._originalWorkspaces.clear();
    }
}
