(() => {
    'use strict';

    const DOM = {
        showCreate: document.getElementById('showCreate'),
        showList: document.getElementById('showList'),
        langSelect: document.getElementById('langSelect'),
        createPanel: document.getElementById('createPanel'),
        importPanel: document.getElementById('importPanel'),
        listPanel: document.getElementById('listPanel'),
        form: document.getElementById('scenarioForm'),
        projectName: document.getElementById('projectName'),
        ipSubrange: document.getElementById('ipSubrange'),
        protocol: document.getElementById('protocol'),
        masterNodes: document.getElementById('masterNodes'),
        slaveNodes: document.getElementById('slaveNodes'),
        scenarioList: document.getElementById('scenarioList'),
        emptyList: document.getElementById('emptyList'),
        refreshList: document.getElementById('refreshList'),
        toast: document.getElementById('toast'),
        importForm: document.getElementById('importForm'),
        importFile: document.getElementById('importFile'),
        importName: document.getElementById('importName')
    };

    function showPanel(panel) {
        const isCreate = panel === 'create';
        DOM.createPanel.classList.toggle('active', isCreate);
        if (DOM.importPanel) DOM.importPanel.classList.toggle('active', isCreate);
        DOM.listPanel.classList.toggle('active', !isCreate);
        DOM.showCreate.setAttribute('aria-pressed', isCreate ? 'true' : 'false');
        DOM.showList.setAttribute('aria-pressed', isCreate ? 'false' : 'true');
    }

    function showToast(message, isError = false) {
        DOM.toast.textContent = message;
        DOM.toast.style.background = isError ? '#b91c1c' : '#111827';
        DOM.toast.classList.add('show');
        setTimeout(() => DOM.toast.classList.remove('show'), 3500);
    }

    function t(key, vars) {
        if (window.I18N) return window.I18N.t(key, vars);
        return key;
    }

    async function fetchJSON(url, options = {}) {
        const response = await fetch(url, options);
        let data = {};
        try {
            data = await response.json();
        } catch (err) {
            data = {};
        }
        if (!response.ok) {
            const message = data.error || data.message || `Error ${response.status}`;
            throw new Error(message);
        }
        return data;
    }

    async function loadScenarios() {
        DOM.scenarioList.innerHTML = '';
        DOM.emptyList.style.display = 'none';

        try {
            const scenarios = await fetchJSON('/api/networks/');
            if (!Array.isArray(scenarios) || scenarios.length === 0) {
                DOM.emptyList.style.display = 'block';
                return;
            }

            scenarios.forEach(name => {
                const card = document.createElement('div');
                card.className = 'scenario-item';
                card.innerHTML = `
                    <h3>${name}</h3>
                    <div class="scenario-actions">
                        <button class="button button-primary" data-action="open">${t('common.open')}</button>
                        <button class="button button-secondary" data-action="delete">${t('common.delete')}</button>
                    </div>
                `;
                card.dataset.name = name;
                DOM.scenarioList.appendChild(card);
            });
        } catch (err) {
            showToast(err.message, true);
        }
    }

    async function handleCreate(event) {
        event.preventDefault();
        const payload = {
            projectName: DOM.projectName.value.trim(),
            ipSubrange: DOM.ipSubrange.value.trim(),
            protocol: DOM.protocol.value,
            masterNodes: Number(DOM.masterNodes.value),
            slaveNodes: Number(DOM.slaveNodes.value)
        };

        try {
            await fetchJSON('/api/networks/', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            showToast(t('index.createSuccess'));
            window.location.href = `/networks/${encodeURIComponent(payload.projectName)}`;
        } catch (err) {
            showToast(err.message, true);
        }
    }

    async function handleImport(event) {
        event.preventDefault();
        const file = DOM.importFile.files[0];
        if (!file) {
            showToast(t('index.importMissingFile'), true);
            return;
        }

        const formData = new FormData();
        formData.append('file', file);
        const name = DOM.importName.value.trim();
        if (name) {
            formData.append('name', name);
        }

        try {
            await fetchJSON('/api/networks/import', {
                method: 'POST',
                body: formData
            });
            showToast(t('index.importSuccess'));
            DOM.importForm.reset();
            loadScenarios();
        } catch (err) {
            showToast(err.message || t('index.importError'), true);
        }
    }

    async function handleScenarioAction(event) {
        const action = event.target.dataset.action;
        if (!action) return;
        const card = event.target.closest('.scenario-item');
        if (!card) return;
        const name = card.dataset.name;

        if (action === 'open') {
            window.location.href = `/networks/${encodeURIComponent(name)}`;
            return;
        }

        if (action === 'delete') {
            if (!confirm(t('index.deleteConfirm', { name }))) return;
            try {
                await fetchJSON(`/api/networks/${encodeURIComponent(name)}`, { method: 'DELETE' });
                showToast(t('index.deleteSuccess'));
                loadScenarios();
            } catch (err) {
                showToast(err.message, true);
            }
        }
    }

    function bindEvents() {
        DOM.showCreate.addEventListener('click', () => showPanel('create'));
        DOM.showList.addEventListener('click', () => {
            showPanel('list');
            loadScenarios();
        });
        DOM.refreshList.addEventListener('click', loadScenarios);
        DOM.form.addEventListener('submit', handleCreate);
        DOM.scenarioList.addEventListener('click', handleScenarioAction);
        DOM.importForm.addEventListener('submit', handleImport);
        if (DOM.langSelect && window.I18N) {
            DOM.langSelect.value = window.I18N.getLang();
            DOM.langSelect.addEventListener('change', (event) => {
                window.I18N.setLang(event.target.value);
            });
            window.I18N.applyTranslations(document);
            window.I18N.onChange(() => {
                window.I18N.applyTranslations(document);
                loadScenarios();
            });
        }
    }

    function init() {
        bindEvents();
        showPanel('create');
    }

    init();
})();
