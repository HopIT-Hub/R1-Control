// R1 Control Settings — client-side JavaScript

(function() {
    'use strict';

    const deviceStatus = document.getElementById('device-status');
    const currentHotkey = document.getElementById('current-hotkey');
    const currentSwipeHotkey = document.getElementById('current-swipe-hotkey');
    const recordBtn = document.getElementById('record-btn');
    const recordingOverlay = document.getElementById('recording-overlay');
    const cancelBtn = document.getElementById('cancel-btn');
    const preview = document.getElementById('preview');
    const previewHotkey = document.getElementById('preview-hotkey');
    const saveBtn = document.getElementById('save-btn');
    const discardBtn = document.getElementById('discard-btn');

    const swipeRecordBtn = document.getElementById('swipe-record-btn');
    const swipeRecordingOverlay = document.getElementById('swipe-recording-overlay');
    const swipeCancelBtn = document.getElementById('swipe-cancel-btn');
    const swipePreview = document.getElementById('swipe-preview');
    const swipePreviewHotkey = document.getElementById('swipe-preview-hotkey');
    const swipeSaveBtn = document.getElementById('swipe-save-btn');
    const swipeDiscardBtn = document.getElementById('swipe-discard-btn');
    const autostartToggle = document.getElementById('autostart-toggle');
    const keepawakeToggle = document.getElementById('keepawake-toggle');
    const sleepAfterSelect = document.getElementById('sleep-after-select');
    const sleepAfterRow = document.getElementById('sleep-after-row');
    const versionFooter = document.getElementById('version-footer');

    let pendingHotkey = null;
    let pendingSwipeHotkey = null;

    // --- Status polling ---
    async function pollStatus() {
        try {
            const res = await fetch('/status');
            const data = await res.json();

            // Update device status
            deviceStatus.textContent = formatState(data.state);
            deviceStatus.className = 'status ' + data.state;

            // Update hotkey displays
            currentHotkey.textContent = data.hotkey;
            if (currentSwipeHotkey) {
                currentSwipeHotkey.textContent = data.swipe_hotkey;
            }

            // Update autostart toggle
            if (autostartToggle && !autostartToggle._userChanging) {
                autostartToggle.checked = data.auto_start;
            }

            // Update keep-awake controls
            if (keepawakeToggle && !keepawakeToggle._userChanging) {
                keepawakeToggle.checked = data.keep_awake;
                updateSleepAfterVisibility(data.keep_awake);
            }
            if (sleepAfterSelect && !sleepAfterSelect._userChanging) {
                sleepAfterSelect.value = String(data.sleep_after_minutes);
            }

            // Update version footer (once)
            if (versionFooter && data.version && !versionFooter.textContent) {
                versionFooter.textContent = 'R1 Control v' + data.version.replace(/^v/, '');
            }
        } catch (e) {
            deviceStatus.textContent = 'Error';
            deviceStatus.className = 'status disconnected';
        }
    }

    function formatState(state) {
        switch (state) {
            case 'disconnected': return 'Disconnected';
            case 'connected': return 'Connected';
            case 'ptt_active': return 'PTT Active';
            default: return state;
        }
    }

    function updateSleepAfterVisibility(keepAwakeEnabled) {
        if (sleepAfterRow) {
            sleepAfterRow.style.opacity = keepAwakeEnabled ? '1' : '0.4';
            sleepAfterRow.style.pointerEvents = keepAwakeEnabled ? 'auto' : 'none';
        }
    }

    // Poll every 2 seconds
    pollStatus();
    setInterval(pollStatus, 2000);

    // --- Auto-start toggle ---
    if (autostartToggle) {
        autostartToggle.addEventListener('change', async function() {
            autostartToggle._userChanging = true;
            const enabled = autostartToggle.checked;

            try {
                const res = await fetch('/autostart', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ enabled: enabled })
                });

                const data = await res.json();

                if (data.error) {
                    showToast(data.error, true);
                    autostartToggle.checked = !enabled; // revert
                } else {
                    showToast(enabled ? 'Will start on login' : 'Will not start on login');
                }
            } catch (e) {
                showToast('Failed to update setting', true);
                autostartToggle.checked = !enabled; // revert
            }

            autostartToggle._userChanging = false;
        });
    }

    // --- Keep-awake toggle ---
    if (keepawakeToggle) {
        keepawakeToggle.addEventListener('change', async function() {
            keepawakeToggle._userChanging = true;
            const enabled = keepawakeToggle.checked;
            const sleepAfter = parseInt(sleepAfterSelect.value, 10);

            updateSleepAfterVisibility(enabled);

            try {
                const res = await fetch('/keepawake', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ enabled: enabled, sleep_after_minutes: sleepAfter })
                });

                const data = await res.json();

                if (data.error) {
                    showToast(data.error, true);
                    keepawakeToggle.checked = !enabled; // revert
                    updateSleepAfterVisibility(!enabled);
                } else {
                    showToast(enabled ? 'Keep awake enabled' : 'Keep awake disabled');
                }
            } catch (e) {
                showToast('Failed to update setting', true);
                keepawakeToggle.checked = !enabled; // revert
                updateSleepAfterVisibility(!enabled);
            }

            keepawakeToggle._userChanging = false;
        });
    }

    // --- Sleep-after dropdown ---
    if (sleepAfterSelect) {
        sleepAfterSelect.addEventListener('change', async function() {
            sleepAfterSelect._userChanging = true;
            const sleepAfter = parseInt(sleepAfterSelect.value, 10);
            const enabled = keepawakeToggle.checked;

            try {
                const res = await fetch('/keepawake', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ enabled: enabled, sleep_after_minutes: sleepAfter })
                });

                const data = await res.json();

                if (data.error) {
                    showToast(data.error, true);
                } else {
                    const label = sleepAfter === 0 ? 'Never' : formatMinutes(sleepAfter);
                    showToast('Sleep after idle: ' + label);
                }
            } catch (e) {
                showToast('Failed to update setting', true);
            }

            sleepAfterSelect._userChanging = false;
        });
    }

    function formatMinutes(mins) {
        if (mins < 60) return mins + ' min';
        const hrs = mins / 60;
        return hrs === 1 ? '1 hour' : hrs + ' hours';
    }

    // --- Hotkey recording ---
    recordBtn.addEventListener('click', startRecording);
    cancelBtn.addEventListener('click', stopRecording);
    saveBtn.addEventListener('click', saveHotkey);
    discardBtn.addEventListener('click', discardHotkey);

    function startRecording() {
        recordingOverlay.classList.remove('hidden');
        document.addEventListener('keydown', captureKey);
    }

    function stopRecording() {
        recordingOverlay.classList.add('hidden');
        document.removeEventListener('keydown', captureKey);
    }

    function captureKey(e) {
        e.preventDefault();
        e.stopPropagation();

        // Ignore bare modifier keys
        if (['Control', 'Shift', 'Alt', 'Meta'].includes(e.key)) {
            return;
        }

        const modifiers = [];
        if (e.ctrlKey) modifiers.push('ctrl');
        if (e.shiftKey) modifiers.push('shift');
        if (e.altKey) modifiers.push('alt');
        if (e.metaKey) modifiers.push('super');

        // Require at least one modifier
        if (modifiers.length === 0) {
            showToast('Please include at least one modifier (Ctrl, Shift, Alt)', true);
            return;
        }

        const keyCode = e.code;

        // Build display string
        const displayParts = modifiers.map(m =>
            m === 'ctrl' ? 'Ctrl' :
            m === 'shift' ? 'Shift' :
            m === 'alt' ? 'Alt' :
            m === 'super' ? 'Super' : m
        );

        // Clean up key name for display
        let keyDisplay = keyCode
            .replace('Key', '')
            .replace('Digit', '')
            .replace('Arrow', '');

        displayParts.push(keyDisplay);
        const displayStr = displayParts.join('+');

        pendingHotkey = {
            modifiers: modifiers,
            jsCode: keyCode,
            display: displayStr
        };

        stopRecording();

        // Show preview
        previewHotkey.textContent = displayStr;
        preview.classList.remove('hidden');
    }

    async function saveHotkey() {
        if (!pendingHotkey) return;

        try {
            const res = await fetch('/hotkey', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    modifiers: pendingHotkey.modifiers,
                    js_code: pendingHotkey.jsCode
                })
            });

            const data = await res.json();

            if (data.error) {
                showToast(data.error, true);
                return;
            }

            currentHotkey.textContent = data.hotkey;
            showToast('Hotkey saved!');
        } catch (e) {
            showToast('Failed to save hotkey: ' + e.message, true);
        }

        discardHotkey();
    }

    function discardHotkey() {
        pendingHotkey = null;
        preview.classList.add('hidden');
    }

    // --- Swipe hotkey recording ---
    swipeRecordBtn.addEventListener('click', startSwipeRecording);
    swipeCancelBtn.addEventListener('click', stopSwipeRecording);
    swipeSaveBtn.addEventListener('click', saveSwipeHotkey);
    swipeDiscardBtn.addEventListener('click', discardSwipeHotkey);

    function startSwipeRecording() {
        swipeRecordingOverlay.classList.remove('hidden');
        document.addEventListener('keydown', captureSwipeKey);
    }

    function stopSwipeRecording() {
        swipeRecordingOverlay.classList.add('hidden');
        document.removeEventListener('keydown', captureSwipeKey);
    }

    function captureSwipeKey(e) {
        e.preventDefault();
        e.stopPropagation();

        if (['Control', 'Shift', 'Alt', 'Meta'].includes(e.key)) {
            return;
        }

        const modifiers = [];
        if (e.ctrlKey) modifiers.push('ctrl');
        if (e.shiftKey) modifiers.push('shift');
        if (e.altKey) modifiers.push('alt');
        if (e.metaKey) modifiers.push('super');

        if (modifiers.length === 0) {
            showToast('Please include at least one modifier (Ctrl, Shift, Alt)', true);
            return;
        }

        const keyCode = e.code;

        const displayParts = modifiers.map(m =>
            m === 'ctrl' ? 'Ctrl' :
            m === 'shift' ? 'Shift' :
            m === 'alt' ? 'Alt' :
            m === 'super' ? 'Super' : m
        );

        let keyDisplay = keyCode
            .replace('Key', '')
            .replace('Digit', '')
            .replace('Arrow', '');

        displayParts.push(keyDisplay);
        const displayStr = displayParts.join('+');

        pendingSwipeHotkey = {
            modifiers: modifiers,
            jsCode: keyCode,
            display: displayStr
        };

        stopSwipeRecording();

        swipePreviewHotkey.textContent = displayStr;
        swipePreview.classList.remove('hidden');
    }

    async function saveSwipeHotkey() {
        if (!pendingSwipeHotkey) return;

        try {
            const res = await fetch('/swipe-hotkey', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    modifiers: pendingSwipeHotkey.modifiers,
                    js_code: pendingSwipeHotkey.jsCode
                })
            });

            const data = await res.json();

            if (data.error) {
                showToast(data.error, true);
                return;
            }

            currentSwipeHotkey.textContent = data.hotkey;
            showToast('Swipe hotkey saved!');
        } catch (e) {
            showToast('Failed to save hotkey: ' + e.message, true);
        }

        discardSwipeHotkey();
    }

    function discardSwipeHotkey() {
        pendingSwipeHotkey = null;
        swipePreview.classList.add('hidden');
    }

    function showToast(message, isError) {
        const toast = document.createElement('div');
        toast.className = 'toast' + (isError ? ' error' : '');
        toast.textContent = message;
        document.body.appendChild(toast);
        setTimeout(() => toast.remove(), 2500);
    }
})();
