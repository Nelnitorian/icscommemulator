window.App = window.App || {};

(function () {
    'use strict';
    const CONSTANTS = App.CONSTANTS;
    let cy, DOM, popupTracker;

    function init() {
        initCytoscape();
        initDOM();
        initEventHandlers();

        if (cy.nodes().every(n => n.position().x === 0 && n.position().y === 0)) {
            cy.layout({ name: 'concentric', spacingFactor: 2 }).run();
        }

        window.save = save;
        window.run = run;
        window.cancelRun = cancelRun;
        window.executeRun = executeRun;
        window.saveConfig = saveConfig;
    }

    function initCytoscape() {
        cy = cytoscape({
            container: document.getElementById('cy'),
            elements: { nodes: networkData.nodes || [], edges: networkData.edges || [] },
            style: App.CYTOSCAPE_STYLE,
            layout: { name: 'preset' }
        });
        window.cy = cy;
    }

    function initDOM() {
        DOM = new App.DOMManager();
        popupTracker = App.PopupTracker;

        window.DOM = DOM;
        App.DOM = DOM;
    }

    function initEventHandlers() {
        cy.on('taphold', evt => {
            const el = evt.target;
            if (el === cy) createNode(evt);
            else {
                App.State.selectElement(el);
                if (el.isNode()) showNodeForm(el);
                else showEdgeForm(el);
            }
        });

        cy.on('tap', evt => {
            const el = evt.target;
            if (el === cy) App.State.unselectElement();
            else if (el.isNode()) handleNodeTap(el);
        });

        document.addEventListener('keydown', evt => {
            if (evt.key === CONSTANTS.KEYS.DELETE && App.State.selectedElement) {
                App.State.selectedElement.remove();
                App.State.selectedElement = null;
                DOM.hideAllPopups();
            }
        });
    }

    function handleNodeTap(node) {
        if (App.State.selectedElement && App.State.selectedElement.isNode() && App.State.selectedElement !== node) {
            const src = App.State.selectedElement;
            cy.add({ group: 'edges', data: { source: src.id(), target: node.id() } });
            App.State.unselectElement();
        } else {
            App.State.selectElement(node);
        }
    }

    function createNode(evt) {
        const id = `node${App.State.incrementNodeCount()}`;
        const tmpl = App.ProtocolConfig.getNodeTemplate();
        cy.add({
            group: 'nodes',
            data: { id: id, name: id, ...tmpl, ip: App.IPManager.assignIP('', '') },
            position: evt.position
        });
    }

    function showNodeForm(node) {
        const data = node.data();
        DOM.fields.name.value = data.name;
        DOM.showNodeConfig();
        popupTracker.attachToElement(DOM.nodeConfigPopup, node);
    }

    function showEdgeForm(edge) {
        const msgs = edge.data('messages') || [];
        App.messagesUI.init(document.getElementById('messages-container'));
        App.messagesUI.render(msgs);
        DOM.showEdgeConfig();
        popupTracker.attachToElement(DOM.edgeConfigPopup, edge, 'center');
    }

    function saveConfig() {
        if (!App.State.selectedElement) return;
        const data = App.State.selectedElement.data();
        data.name = DOM.fields.name.value;

        if (DOM.registers && DOM.registers.discreteInputsType) {
            const val = App.Validators.parseRegisterValues(
                DOM.registers.discreteInputsType.value,
                DOM.registers.discreteInputs.value
            );
            if (val !== null && val !== -1) data.discrete_inputs.values = val;
        }

        App.State.selectedElement.data(data);
        DOM.hideAllPopups();
    }

    function save() {
        const valid = App.ClientValidator.validateScenario();
        if (!App.ClientValidator.showValidationResults(valid.errors, valid.warnings)) return;

        const json = App.APITransformer.cytoscapeToAPI(cy.json());
        fetch(`/api/networks/${App.State.scenarioId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(json)
        }).then(r => r.json()).then(res => {
            App.UIUtils.showError('Saved!');
        });
    }

    function run() { DOM.run.settings.style.display = 'block'; }
    function cancelRun() { DOM.run.settings.style.display = 'none'; }
    function executeRun() {
        DOM.run.settings.style.display = 'none';
        alert('Simulation started (Mock)');
    }

    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
    else init();

}());
