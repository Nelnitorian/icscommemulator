window.App = window.App || {};

(function () {
    'use strict';

    const CONSTANTS = {
        KEYS: { DELETE: 'Delete', SUPR: 'Supr', ESCAPE: 'Escape', Z: 'z', Y: 'y' },
        ACTIONS: { ADD: 'add', DELETE: 'delete' },
        ROLES: { MASTER: 'master', SLAVE: 'slave' },
        EDGE: { ARROW_SHAPE: 'triangle' },
        LOG_LEVEL: { ERROR: 1, WARNING: 2 },
        REGISTER_TYPES: { SEQUENTIAL: 'sequential', SPARSE: 'sparse' },
        FUNCTION_CODES: [1, 2, 3, 4, 5, 6, 15, 16, 43],
        TAPHOLD_DELAY: 500
    };

    const CYTOSCAPE_STYLE = [
        { selector: 'node', style: { 'background-color': '#666', 'label': 'data(name)' } },
        { selector: 'node.master', style: { 'background-color': '#3A86FF' } },
        { selector: 'node.slave', style: { 'background-color': '#FF6D00' } },
        { selector: 'node.selected', style: { 'background-color': '#FFA500' } },
        { selector: 'edge', style: { 'width': 3, 'line-color': '#ccc', 'curve-style': 'bezier', 'target-arrow-color': '#ccc', 'target-arrow-shape': 'triangle' } },
        { selector: 'edge.selected', style: { 'line-color': '#FFA500', 'target-arrow-color': '#FFA500' } }
    ];

    class StateManager {
        constructor() {
            this.scenarioId = this._getCurrentId();
            this.nodeCount = 3;
            this.edgeCount = 2;
            this.selectedElement = null;
            this.history = [];
            this.redoHistory = [];
            this.lastTaphold = 0;
            this.canvasDragHandler = null;
            this.intervalId = null;
            this.filePath = null;
            this.connection = { status: 'disconnected', lastError: null };
        }

        _getCurrentId() {
            const path = window.location.pathname;
            const parts = path.split('/');
            return parts[parts.length - 1] || parts[parts.length - 2];
        }

        incrementNodeCount() { return ++this.nodeCount; }
        incrementEdgeCount() { return ++this.edgeCount; }

        selectElement(element) {
            if (this.selectedElement) this.unselectElement();
            this.selectedElement = element;
            element.addClass('selected');
        }

        unselectElement() {
            if (this.selectedElement) {
                this.selectedElement.removeClass('selected');
                if (this.selectedElement.isNode()) {
                    this.selectedElement.removeClass(CONSTANTS.ROLES.MASTER);
                    this.selectedElement.removeClass(CONSTANTS.ROLES.SLAVE);
                    this.selectedElement.addClass(this.selectedElement.data('role'));
                }
            }
            this.selectedElement = null;
        }

        addToHistory(action, element, connectedEdges = null) {
            this.history.push({ action, element, connectedEdges });
            this.redoHistory = [];
        }

        undo(cy) {
            if (this.history.length === 0) return;
            const lastAction = this.history.pop();
            this.redoHistory.push(lastAction);
            if (lastAction.action === CONSTANTS.ACTIONS.ADD) {
                cy.$id(`${lastAction.element.data.id}`).remove();
            } else if (lastAction.action === CONSTANTS.ACTIONS.DELETE) {
                cy.add(lastAction.element);
                if (lastAction.connectedEdges) lastAction.connectedEdges.forEach(edge => cy.add(edge));
            }
        }

        redo(cy) {
            if (this.redoHistory.length === 0) return;
            const lastRedo = this.redoHistory.pop();
            this.history.push(lastRedo);
            if (lastRedo.action === CONSTANTS.ACTIONS.ADD) {
                cy.add(lastRedo.element);
            } else if (lastRedo.action === CONSTANTS.ACTIONS.DELETE) {
                cy.$id(`${lastRedo.element.data.id}`).remove();
            }
        }

        setLastTaphold() { this.lastTaphold = Date.now(); }

        isTapholdRecent() {
            return (Date.now() - this.lastTaphold) < CONSTANTS.TAPHOLD_DELAY;
        }

        removeCanvasListener(cy) {
            if (this.canvasDragHandler) {
                cy.off('drag', this.canvasDragHandler);
                this.canvasDragHandler = null;
            }
        }
    }

    class IPManager {
        static getNextIPInSubnet(ipAddress, subnet) {
            const ip = ipaddr.parse(ipAddress);
            const subnetParsed = ipaddr.parseCIDR(subnet);
            if (!ip.match(subnetParsed)) throw new Error('IP outside subnet.');

            const nextIp = ip.toByteArray();
            for (let i = nextIp.length - 1; i >= 0; i--) {
                if (nextIp[i] < 255) { nextIp[i]++; break; }
                nextIp[i] = 0;
            }
            return ipaddr.fromByteArray(nextIp).toString();
        }

        static getUniqueIP(existingIPs, subnet) {
            let newIP = subnet.split('/')[0];
            while (existingIPs.includes(newIP)) {
                newIP = this.getNextIPInSubnet(newIP, subnet);
            }
            return newIP;
        }

        static assignIP(ipCandidate, fallback) {
            const ipValue = (ipCandidate || '').trim();

            if (ipValue === '' && window.cy) {
                const existingIPs = window.cy.nodes().map(node => node.data('ip'));
                existingIPs.push(this.getUniqueIP([], networkData.ip_network));
                return this.getUniqueIP(existingIPs, networkData.ip_network);
            }

            if (App.Validators && App.Validators.validateIP(ipValue)) {
                return ipValue;
            } else {
                console.error('Invalid IP address');
                if (App.UIUtils) {
                    App.UIUtils.showError('Error: Invalid IP address');
                } else {
                    alert('Error: Invalid IP address');
                }
                return fallback;
            }
        }

        static generateMAC() {
            const bytes = [];
            for (let i = 0; i < 6; i++) {
                bytes.push(Math.floor(Math.random() * 256).toString(16).padStart(2, '0'));
            }
            return bytes.join(':').toUpperCase();
        }
    }

    class APITransformer {
        static cytoscapeToAPI(json) {
            const nodes = json.elements.nodes || [];
            const edges = json.elements.edges || [];
            return {
                protocol: networkData.protocol,
                ip_network: networkData.ip_network,
                nodes: nodes.map(n => ({ data: n.data, classes: n.classes, position: { x: n.position.x, y: n.position.y } })),
                edges: edges.map(e => ({ data: e.data }))
            };
        }
    }

    App.CONSTANTS = CONSTANTS;
    App.CYTOSCAPE_STYLE = CYTOSCAPE_STYLE;
    App.State = new StateManager();
    App.IPManager = IPManager;
    App.APITransformer = APITransformer;
}());
