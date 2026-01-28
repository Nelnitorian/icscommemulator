(() => {
    'use strict';

    const PROTOCOLS = {
        modbus: {
            name: 'Modbus',
            defaultPort: '502',
            nodeTemplate: {
                role: 'slave',
                port: '502',
                slave_id: '1',
                holding_registers: {},
                coils: {},
                discrete_inputs: {},
                input_registers: {},
                identity: {
                    vendor_name: '',
                    product_code: '',
                    major_minor_revision: '',
                    vendor_url: '',
                    product_name: '',
                    model_name: '',
                    user_application_name: ''
                },
                comment: ''
            }
        },
        dnp3: {
            name: 'DNP3',
            defaultPort: '20000',
            nodeTemplate: {
                role: 'slave',
                port: '20000',
                outstation_id: '1',
                master_id: '2',
                analog_inputs: {},
                binary_inputs: {},
                analog_output_status: {},
                binary_output_status: {},
                simulation: { enabled: false, interval: 1000 },
                comment: ''
            }
        },
        iec104: {
            name: 'IEC 104',
            defaultPort: '2404',
            nodeTemplate: {
                role: 'slave',
                port: '2404',
                common_address: 1,
                tick_rate_ms: 100,
                select_timeout_ms: 10000,
                max_connections: 5,
                authorized_masters: [],
                k: 12,
                w: 8,
                t1: 15,
                t2: 10,
                t3: 20,
                single_points: {},
                measured_short: {},
                comment: ''
            }
        }
    };

    const DOM = {
        scenarioName: document.getElementById('scenarioName'),
        scenarioProtocol: document.getElementById('scenarioProtocol'),
        scenarioNetwork: document.getElementById('scenarioNetwork'),
        nodesCount: document.getElementById('nodesCount'),
        edgesCount: document.getElementById('edgesCount'),
        nodePanel: document.getElementById('nodePanel'),
        edgePanel: document.getElementById('edgePanel'),
        emptyState: document.getElementById('emptyState'),
        toast: document.getElementById('toast'),
        btnAddNode: document.getElementById('btnAddNode'),
        btnLayout: document.getElementById('btnLayout'),
        btnSave: document.getElementById('btnSave'),
        btnRun: document.getElementById('btnRun'),
        cancelNode: document.getElementById('cancelNode'),
        addMessage: document.getElementById('addMessage'),
        saveEdge: document.getElementById('saveEdge'),
        runModal: document.getElementById('runModal'),
        simulationTime: document.getElementById('simulationTime'),
        cancelRun: document.getElementById('cancelRun'),
        confirmRun: document.getElementById('confirmRun'),
        runOverlay: document.getElementById('runOverlay'),
        progressTime: document.getElementById('progressTime'),
        progressPercent: document.getElementById('progressPercent'),
        progressPcap: document.getElementById('progressPcap'),
        stopRun: document.getElementById('stopRun'),
        edgeSource: document.getElementById('edgeSource'),
        edgeTarget: document.getElementById('edgeTarget'),
        attacksPanel: document.getElementById('attacksPanel'),
        attacksList: document.getElementById('attacksList'),
        jumpAttacks: document.getElementById('jumpAttacks'),
        expandAttacks: document.getElementById('expandAttacks'),
        collapseAttacks: document.getElementById('collapseAttacks'),
        attacksMasterSelect: document.getElementById('attacksMasterSelect'),
        saveAttacks: document.getElementById('saveAttacks'),
        messagesContainer: document.getElementById('messagesContainer'),
        nodeForm: document.getElementById('nodeForm'),
        nodeName: document.getElementById('nodeName'),
        nodeRole: document.getElementById('nodeRole'),
        nodeIp: document.getElementById('nodeIp'),
        nodePort: document.getElementById('nodePort'),
        nodeComment: document.getElementById('nodeComment'),
        nodeSlaveId: document.getElementById('nodeSlaveId'),
        holdingRegisters: document.getElementById('holdingRegisters'),
        inputRegisters: document.getElementById('inputRegisters'),
        coils: document.getElementById('coils'),
        discreteInputs: document.getElementById('discreteInputs'),
        vendorName: document.getElementById('vendorName'),
        productCode: document.getElementById('productCode'),
        productName: document.getElementById('productName'),
        modelName: document.getElementById('modelName'),
        majorMinorRevision: document.getElementById('majorMinorRevision'),
        vendorUrl: document.getElementById('vendorUrl'),
        userApplicationName: document.getElementById('userApplicationName'),
        dnp3Block: document.getElementById('dnp3Block'),
        dnp3MasterFields: document.getElementById('dnp3MasterFields'),
        dnp3SlaveFields: document.getElementById('dnp3SlaveFields'),
        modbusBlock: document.getElementById('modbusBlock'),
        iec104Block: document.getElementById('iec104Block'),
        outstationId: document.getElementById('outstationId'),
        masterId: document.getElementById('masterId'),
        analogInputs: document.getElementById('analogInputs'),
        binaryInputs: document.getElementById('binaryInputs'),
        analogOutputStatus: document.getElementById('analogOutputStatus'),
        binaryOutputStatus: document.getElementById('binaryOutputStatus'),
        dnp3Simulation: document.getElementById('dnp3Simulation'),
        dnp3Interval: document.getElementById('dnp3Interval'),
        commonAddress: document.getElementById('commonAddress'),
        tickRateMs: document.getElementById('tickRateMs'),
        selectTimeoutMs: document.getElementById('selectTimeoutMs'),
        maxConnections: document.getElementById('maxConnections'),
        authorizedMasters: document.getElementById('authorizedMasters'),
        iecK: document.getElementById('iecK'),
        iecW: document.getElementById('iecW'),
        iecT1: document.getElementById('iecT1'),
        iecT2: document.getElementById('iecT2'),
        iecT3: document.getElementById('iecT3'),
        iecSinglePoints: document.getElementById('iecSinglePoints'),
        iecMeasuredShort: document.getElementById('iecMeasuredShort')
    };

    const state = {
        cy: null,
        selectedElement: null,
        protocol: (networkData?.protocol || 'modbus').toLowerCase(),
        scenarioId: getScenarioId(),
        pollInterval: null,
        attacksMasterId: ''
    };

    let attackCatalog = [];
    let attackCatalogLoaded = false;

    function showToast(message, isError = false) {
        DOM.toast.textContent = message;
        DOM.toast.style.background = isError ? '#b91c1c' : '#111827';
        DOM.toast.classList.add('show');
        setTimeout(() => DOM.toast.classList.remove('show'), 4000);
    }

    function getScenarioId() {
        const parts = window.location.pathname.split('/').filter(Boolean);
        return parts[parts.length - 1] || '';
    }

    function getProtocolConfig() {
        return PROTOCOLS[state.protocol] || PROTOCOLS.modbus;
    }

    function getAttackCatalog() {
        if (!attackCatalogLoaded) {
            return [];
        }
        return attackCatalog.filter(attack => attack.protocol === state.protocol);
    }

    async function loadAttackCatalog() {
        try {
            const response = await fetch('/static/attacks/attacks.json', { cache: 'no-store' });
            if (!response.ok) throw new Error('Attack catalog not available');
            const payload = await response.json();
            attackCatalog = Array.isArray(payload.attacks) ? payload.attacks : [];
            attackCatalogLoaded = true;
        } catch (err) {
            attackCatalog = [];
            attackCatalogLoaded = false;
            showToast('No se pudo cargar el catálogo de ataques', true);
        }
    }

    function initHeader() {
        DOM.scenarioName.textContent = state.scenarioId || 'Escenario';
        DOM.scenarioProtocol.textContent = getProtocolConfig().name;
        DOM.scenarioProtocol.classList.remove('protocol-modbus', 'protocol-dnp3', 'protocol-iec104');
        DOM.scenarioProtocol.classList.add(`protocol-${state.protocol}`);
        DOM.scenarioNetwork.textContent = networkData?.ip_network || '-';
    }

    function normalizeNode(node) {
        const data = node.data || node;
        const position = node.position || data.position;
        const classes = node.classes || data.role || '';
        if (!data.name) data.name = data.id;
        return { data, position, classes };
    }

    function normalizeEdge(edge) {
        const data = edge.data || edge;
        return { data };
    }

    function initCytoscape() {
        const elements = {
            nodes: (networkData?.nodes || []).map(normalizeNode),
            edges: (networkData?.edges || []).map(normalizeEdge)
        };

        state.cy = cytoscape({
            container: document.getElementById('cy'),
            elements,
            layout: { name: 'preset' },
            style: [
                { selector: 'node', style: { 'background-color': '#64748b', 'label': 'data(name)' } },
                { selector: 'node.master', style: { 'background-color': '#2563eb' } },
                { selector: 'node.slave', style: { 'background-color': '#f97316' } },
                { selector: 'node.selected', style: { 'border-width': 3, 'border-color': '#facc15' } },
                { selector: 'edge', style: { 'width': 2, 'line-color': '#94a3b8', 'target-arrow-color': '#94a3b8', 'target-arrow-shape': 'triangle' } },
                { selector: 'edge.selected', style: { 'line-color': '#facc15', 'target-arrow-color': '#facc15' } }
            ]
        });

        if (state.cy.nodes().length === 0) {
            DOM.emptyState.style.display = 'flex';
        }

        state.cy.nodes().forEach(node => applyRoleClass(node));
    }

    function applyRoleClass(node) {
        node.removeClass('master');
        node.removeClass('slave');
        node.addClass(node.data('role'));
    }

    function updateCounts() {
        DOM.nodesCount.textContent = state.cy.nodes().length;
        DOM.edgesCount.textContent = state.cy.edges().length;
    }

    function deselectElement() {
        if (state.selectedElement) {
            state.selectedElement.removeClass('selected');
        }
        state.selectedElement = null;
        if (DOM.nodePanel) DOM.nodePanel.classList.add('hidden');
        if (DOM.edgePanel) DOM.edgePanel.classList.add('hidden');
        if (DOM.attacksPanel) DOM.attacksPanel.classList.add('hidden');
        if (DOM.edgeSource && DOM.edgeTarget) {
            DOM.edgeSource.textContent = '-';
            DOM.edgeTarget.textContent = '-';
        }
    }

    function selectElement(el) {
        deselectElement();
        state.selectedElement = el;
        el.addClass('selected');
        if (el.isNode()) {
            DOM.nodePanel.classList.remove('hidden');
            populateNodeForm(el.data());
        } else {
            DOM.edgePanel.classList.remove('hidden');
            populateEdgeForm(el);
        }
    }

    function nextNodeId(role) {
        const prefix = role === 'master' ? 'master' : 'slave';
        let i = 0;
        while (state.cy.$id(`${prefix}_${i}`).length) i++;
        return `${prefix}_${i}`;
    }

    function assignNextIp() {
        if (!networkData?.ip_network) return '';
        const [base, prefix] = ipaddr.parseCIDR(networkData.ip_network);
        const existing = new Set(state.cy.nodes().map(node => node.data('ip')));
        let candidate = base;
        for (let i = 0; i < 512; i++) {
            candidate = incrementIp(candidate);
            const candidateStr = candidate.toString();
            if (!existing.has(candidateStr) && candidate.match([base, prefix])) {
                return candidateStr;
            }
        }
        return base.toString();
    }

    function incrementIp(ip) {
        const bytes = ip.toByteArray();
        for (let i = bytes.length - 1; i >= 0; i--) {
            if (bytes[i] < 255) {
                bytes[i] += 1;
                break;
            }
            bytes[i] = 0;
        }
        return ipaddr.fromByteArray(bytes);
    }

    function getViewportCenter() {
        const pan = state.cy.pan();
        const zoom = state.cy.zoom();
        const width = state.cy.width();
        const height = state.cy.height();
        return {
            x: (width / 2 - pan.x) / zoom,
            y: (height / 2 - pan.y) / zoom
        };
    }

    function addNode(position = null) {
        const id = nextNodeId('slave');
        const template = JSON.parse(JSON.stringify(getProtocolConfig().nodeTemplate));
        const node = state.cy.add({
            group: 'nodes',
            data: {
                id,
                name: id,
                role: template.role,
                ip: assignNextIp(),
                ...template
            },
            position: position || getViewportCenter()
        });

        applyRoleClass(node);
        updateCounts();
        DOM.emptyState.style.display = 'none';
    }

    function addEdge(source, target) {
        const id = `edge_${Date.now()}`;
        state.cy.add({
            group: 'edges',
            data: { id, source, target, messages: [] }
        });
        updateCounts();
    }

    function getEdgeContext(edge) {
        if (!edge || !edge.source || !edge.target) return null;
        const source = edge.source();
        const target = edge.target();
        if (!source || !target) return null;
        const master = source.data('role') === 'master' ? source : target;
        const slave = source.data('role') === 'slave' ? source : target;
        return { source, target, master, slave };
    }

    function getNodeLabel(node) {
        if (!node) return '-';
        const data = node.data();
        return data.name || data.id || '-';
    }

    function buildMessageDefaults(context) {
        if (!context) return {};
        const targetData = context.slave?.data() || {};
        const masterData = context.master?.data() || {};
        return {
            ip: targetData.ip || '',
            port: toInt(targetData.port, 0),
            slave_id: toInt(targetData.slave_id, 0),
            master_id: toInt(masterData.master_id, 0),
            outstation_id: toInt(targetData.outstation_id, 0),
            common_address: toInt(targetData.common_address, 0)
        };
    }

    function applyMessageDefaults(message, defaults) {
        const msg = { ...message };
        if (!msg.ip && defaults.ip) msg.ip = defaults.ip;
        if (!msg.port && defaults.port) msg.port = defaults.port;
        if (state.protocol === 'modbus' && !msg.slave_id && defaults.slave_id) {
            msg.slave_id = defaults.slave_id;
        }
        if (state.protocol === 'dnp3') {
            if (!msg.master_id && defaults.master_id) msg.master_id = defaults.master_id;
            if (!msg.outstation_id && defaults.outstation_id) msg.outstation_id = defaults.outstation_id;
        }
        if (state.protocol === 'iec104' && !msg.common_address && defaults.common_address) {
            msg.common_address = defaults.common_address;
        }
        return msg;
    }

    function parseKeyValueMap(text) {
        if (!text.trim()) return {};
        const entries = text.split(/[\n,]+/).map(entry => entry.trim()).filter(Boolean);
        const hasExplicitKey = entries.some(entry => entry.includes(':') || entry.includes('='));
        const map = {};

        if (!hasExplicitKey) {
            entries.forEach((value, index) => {
                const num = Number(value);
                if (!Number.isNaN(num)) {
                    map[String(index)] = num;
                }
            });
            return map;
        }

        entries.forEach(entry => {
            const parts = entry.split(/[:=]/);
            if (parts.length < 2) return;
            const key = parts[0].trim();
            const value = Number(parts.slice(1).join(':').trim());
            if (!Number.isNaN(value) && key !== '') {
                map[key] = value;
            }
        });
        return map;
    }

    function mapToText(map) {
        if (!map) return '';
        if (Array.isArray(map)) {
            return map.map((value, index) => `${index}:${value}`).join('\n');
        }
        if (typeof map === 'object' && map.values) {
            return mapToText(map.values);
        }
        if (typeof map !== 'object') return '';
        return Object.entries(map).map(([k, v]) => `${k}:${v}`).join('\n');
    }

    function parseJSONField(value) {
        if (!value.trim()) return {};
        try {
            return JSON.parse(value);
        } catch (err) {
            showToast('JSON inválido en campos de protocolo', true);
            return {};
        }
    }

    function toInt(value, fallback) {
        const parsed = Number(value);
        if (Number.isNaN(parsed)) {
            return fallback;
        }
        return Math.trunc(parsed);
    }

    function normalizeIecPoints(raw) {
        if (!raw) return {};
        const normalized = {};

        if (Array.isArray(raw)) {
            raw.forEach((entry, index) => {
                if (entry && typeof entry === 'object') {
                    const ioa = toInt(entry.ioa ?? index, index);
                    normalized[String(ioa)] = {
                        ioa,
                        value: entry.value ?? entry,
                        report_ms: toInt(entry.report_ms, 0)
                    };
                }
            });
            return normalized;
        }

        Object.entries(raw).forEach(([key, value]) => {
            const ioa = toInt(key, 0);
            if (value && typeof value === 'object' && !Array.isArray(value)) {
                normalized[String(ioa)] = {
                    ioa: toInt(value.ioa, ioa),
                    value: value.value,
                    report_ms: toInt(value.report_ms, 0)
                };
                return;
            }
            normalized[String(ioa)] = { ioa, value, report_ms: 0 };
        });

        return normalized;
    }

    function updateProtocolVisibility() {
        const protocol = state.protocol;
        const isSlave = DOM.nodeRole.value === 'slave';
        const isMaster = DOM.nodeRole.value === 'master';
        DOM.modbusBlock.style.display = protocol === 'modbus' && isSlave ? 'block' : 'none';
        DOM.dnp3Block.style.display = protocol === 'dnp3' ? 'block' : 'none';
        if (protocol === 'dnp3') {
            DOM.dnp3MasterFields.style.display = isMaster ? 'block' : 'none';
            DOM.dnp3SlaveFields.style.display = isSlave ? 'block' : 'none';
        }
        DOM.iec104Block.style.display = protocol === 'iec104' && isSlave ? 'block' : 'none';
    }

    function populateNodeForm(data) {
        const protocol = state.protocol;
        DOM.nodeName.value = data.name || '';
        DOM.nodeRole.value = data.role || 'slave';
        DOM.nodeIp.value = data.ip || '';
        DOM.nodePort.value = data.port || getProtocolConfig().defaultPort;
        DOM.nodeComment.value = data.comment || '';

        updateProtocolVisibility();
        renderAttacksPanel(data);

        if (protocol === 'modbus') {
            DOM.nodeSlaveId.value = data.slave_id || '';
            DOM.holdingRegisters.value = mapToText(data.holding_registers);
            DOM.inputRegisters.value = mapToText(data.input_registers);
            DOM.coils.value = mapToText(data.coils);
            DOM.discreteInputs.value = mapToText(data.discrete_inputs);
            const identity = data.identity || {};
            DOM.vendorName.value = identity.vendor_name || '';
            DOM.productCode.value = identity.product_code || '';
            DOM.productName.value = identity.product_name || '';
            DOM.modelName.value = identity.model_name || '';
            DOM.majorMinorRevision.value = identity.major_minor_revision || '';
            DOM.vendorUrl.value = identity.vendor_url || '';
            DOM.userApplicationName.value = identity.user_application_name || '';
        }

        if (protocol === 'dnp3') {
            DOM.outstationId.value = data.outstation_id || '';
            DOM.masterId.value = data.master_id || '';
            DOM.analogInputs.value = JSON.stringify(data.analog_inputs || {}, null, 2);
            DOM.binaryInputs.value = JSON.stringify(data.binary_inputs || {}, null, 2);
            DOM.analogOutputStatus.value = JSON.stringify(data.analog_output_status || {}, null, 2);
            DOM.binaryOutputStatus.value = JSON.stringify(data.binary_output_status || {}, null, 2);
            DOM.dnp3Simulation.checked = data.simulation?.enabled || false;
            DOM.dnp3Interval.value = data.simulation?.interval || 1000;
        }

        if (protocol === 'iec104') {
            DOM.commonAddress.value = data.common_address || 1;
            DOM.tickRateMs.value = data.tick_rate_ms || 100;
            DOM.selectTimeoutMs.value = data.select_timeout_ms || 10000;
            DOM.maxConnections.value = data.max_connections || 5;
            DOM.authorizedMasters.value = Array.isArray(data.authorized_masters)
                ? data.authorized_masters.join(',')
                : '';
            DOM.iecK.value = data.k || 12;
            DOM.iecW.value = data.w || 8;
            DOM.iecT1.value = data.t1 || 15;
            DOM.iecT2.value = data.t2 || 10;
            DOM.iecT3.value = data.t3 || 20;
            DOM.iecSinglePoints.value = JSON.stringify(data.single_points || {}, null, 2);
            DOM.iecMeasuredShort.value = JSON.stringify(data.measured_short || {}, null, 2);
        }
    }

    function saveNode(event) {
        event.preventDefault();
        if (!state.selectedElement || !state.selectedElement.isNode()) return;
        const data = state.selectedElement.data();

        const ipValue = DOM.nodeIp.value.trim();
        if (!ipaddr.isValid(ipValue)) {
            showToast('IP inválida', true);
            return;
        }
        if (networkData?.ip_network) {
            const ip = ipaddr.parse(ipValue);
            const subnet = ipaddr.parseCIDR(networkData.ip_network);
            if (!ip.match(subnet)) {
                showToast('IP fuera de la subred', true);
                return;
            }
        }

        data.name = DOM.nodeName.value.trim();
        data.role = DOM.nodeRole.value;
        data.ip = ipValue;
        data.port = toInt(DOM.nodePort.value.trim(), toInt(getProtocolConfig().defaultPort, 0));
        data.comment = DOM.nodeComment.value.trim();

        if (state.protocol === 'modbus' && data.role === 'slave') {
            data.slave_id = toInt(DOM.nodeSlaveId.value, 1);
            data.holding_registers = parseKeyValueMap(DOM.holdingRegisters.value);
            data.input_registers = parseKeyValueMap(DOM.inputRegisters.value);
            data.coils = parseKeyValueMap(DOM.coils.value);
            data.discrete_inputs = parseKeyValueMap(DOM.discreteInputs.value);
            data.identity = {
                vendor_name: DOM.vendorName.value.trim(),
                product_code: DOM.productCode.value.trim(),
                major_minor_revision: DOM.majorMinorRevision.value.trim(),
                vendor_url: DOM.vendorUrl.value.trim(),
                product_name: DOM.productName.value.trim(),
                model_name: DOM.modelName.value.trim(),
                user_application_name: DOM.userApplicationName.value.trim()
            };
        }

        if (state.protocol === 'dnp3') {
            if (data.role === 'master') {
                data.master_id = toInt(DOM.masterId.value, 2);
            }
            if (data.role === 'slave') {
                data.outstation_id = toInt(DOM.outstationId.value, 1);
                data.master_id = toInt(DOM.masterId.value, 2);
                data.analog_inputs = parseJSONField(DOM.analogInputs.value);
                data.binary_inputs = parseJSONField(DOM.binaryInputs.value);
                data.analog_output_status = parseJSONField(DOM.analogOutputStatus.value);
                data.binary_output_status = parseJSONField(DOM.binaryOutputStatus.value);
                data.simulation = {
                    enabled: DOM.dnp3Simulation.checked,
                    interval: Number(DOM.dnp3Interval.value) || 1000
                };
            }
        }

        if (state.protocol === 'iec104' && data.role === 'slave') {
            data.common_address = toInt(DOM.commonAddress.value, 1);
            data.tick_rate_ms = Number(DOM.tickRateMs.value) || 100;
            data.select_timeout_ms = Number(DOM.selectTimeoutMs.value) || 10000;
            data.max_connections = Number(DOM.maxConnections.value) || 5;
            data.authorized_masters = DOM.authorizedMasters.value
                .split(',')
                .map(v => Number(v.trim()))
                .filter(v => !Number.isNaN(v) && v > 0);
            data.k = Number(DOM.iecK.value) || 12;
            data.w = Number(DOM.iecW.value) || 8;
            data.t1 = Number(DOM.iecT1.value) || 15;
            data.t2 = Number(DOM.iecT2.value) || 10;
            data.t3 = Number(DOM.iecT3.value) || 20;
            data.single_points = normalizeIecPoints(parseJSONField(DOM.iecSinglePoints.value));
            data.measured_short = normalizeIecPoints(parseJSONField(DOM.iecMeasuredShort.value));
        }

        if (data.role === 'master') {
            data.attacks = collectAttacksFromDOM();
        } else if (data.attacks) {
            delete data.attacks;
        }

        state.selectedElement.data(data);
        applyRoleClass(state.selectedElement);
        showToast('Nodo actualizado');
    }

    function populateEdgeForm(edge) {
        const data = edge.data();
        const context = getEdgeContext(edge);
        const defaults = buildMessageDefaults(context);
        const messages = (data.messages || []).map(msg => applyMessageDefaults(msg, defaults));
        if (DOM.edgeSource && DOM.edgeTarget) {
            DOM.edgeSource.textContent = getNodeLabel(context?.master);
            DOM.edgeTarget.textContent = getNodeLabel(context?.slave);
        }
        renderMessages(messages.length ? messages : [applyMessageDefaults(getEmptyMessage(), defaults)]);
    }

    function getEmptyMessage() {
        if (state.protocol === 'dnp3') {
            return {
                timestamp: 0,
                recurrent: false,
                interval: 0,
                ip: '',
                port: 0,
                operation_type: 'poll_analog_inputs',
                group: 30,
                variation: 6,
                index: 0,
                master_id: 0,
                outstation_id: 0,
                value: ''
            };
        }
        if (state.protocol === 'iec104') {
            return {
                timestamp: 0,
                recurrent: false,
                interval: 0,
                ip: '',
                port: 0,
                type_id: 0,
                common_address: 0,
                ioa: 1,
                cot: 0,
                value: ''
            };
        }
        return {
            timestamp: 0,
            recurrent: false,
            interval: 0,
            ip: '',
            port: 0,
            slave_id: 0,
            function_code: 3,
            start_address: 0,
            count: 1,
            values: []
        };
    }

    function renderMessages(messages) {
        DOM.messagesContainer.innerHTML = '';
        if (!messages || messages.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'message-empty';
            empty.textContent = 'No hay mensajes definidos para este enlace.';
            DOM.messagesContainer.appendChild(empty);
            return;
        }
        const list = document.createElement('div');
        list.className = 'messages-list';
        messages.forEach((msg, index) => {
            list.appendChild(buildMessageCard(msg, index));
        });
        DOM.messagesContainer.appendChild(list);
    }

    function buildMessageCard(msg, index) {
        const card = document.createElement('div');
        card.className = 'message-card';
        card.dataset.index = index;

        const summary = document.createElement('div');
        summary.className = 'message-summary';
        summary.innerHTML = buildMessageSummary(msg);

        const details = document.createElement('details');
        const summaryToggle = document.createElement('summary');
        summaryToggle.innerHTML = `
            <span>Editar mensaje</span>
            <span class="message-actions">
                <button type="button" class="button button-secondary" data-action="remove-message">Eliminar</button>
            </span>
        `;

        const body = document.createElement('div');
        body.className = 'message-card-body';
        body.innerHTML = buildMessageFields(msg, index);

        details.appendChild(summaryToggle);
        details.appendChild(body);
        card.appendChild(summary);
        card.appendChild(details);
        return card;
    }

    function buildMessageSummary(msg) {
        if (state.protocol === 'dnp3') {
            return `
                <div>
                    <strong>${msg.operation_type || 'Operación DNP3'}</strong>
                    <div class="message-meta">
                        <span class="message-badge">Group ${msg.group || 0}</span>
                        <span class="message-badge">Var ${msg.variation || 0}</span>
                        <span class="message-badge">Index ${msg.index || 0}</span>
                    </div>
                </div>
                <div class="message-meta">
                    <span class="message-badge">${msg.ip || 'IP destino'}</span>
                    <span class="message-badge">Port ${msg.port || 0}</span>
                    <span class="message-badge">${msg.recurrent ? `Recurrente cada ${msg.interval || 0}s` : `T+${msg.timestamp || 0}s`}</span>
                </div>
            `;
        }
        if (state.protocol === 'iec104') {
            return `
                <div>
                    <strong>Type ID ${msg.type_id || 0}</strong>
                    <div class="message-meta">
                        <span class="message-badge">IOA ${msg.ioa || 0}</span>
                        <span class="message-badge">COT ${msg.cot || 0}</span>
                    </div>
                </div>
                <div class="message-meta">
                    <span class="message-badge">${msg.ip || 'IP destino'}</span>
                    <span class="message-badge">Port ${msg.port || 0}</span>
                    <span class="message-badge">${msg.recurrent ? `Recurrente cada ${msg.interval || 0}s` : `T+${msg.timestamp || 0}s`}</span>
                </div>
            `;
        }
        return `
            <div>
                <strong>FC ${msg.function_code || 0}</strong>
                <div class="message-meta">
                    <span class="message-badge">Addr ${msg.start_address || 0}</span>
                    <span class="message-badge">Count ${msg.count || 0}</span>
                    <span class="message-badge">Slave ${msg.slave_id || 0}</span>
                </div>
            </div>
            <div class="message-meta">
                <span class="message-badge">${msg.ip || 'IP destino'}</span>
                <span class="message-badge">Port ${msg.port || 0}</span>
                <span class="message-badge">${msg.recurrent ? `Recurrente cada ${msg.interval || 0}s` : `T+${msg.timestamp || 0}s`}</span>
            </div>
        `;
    }

    function buildMessageFields(msg, index) {
        const baseFields = `
            <div class="form-row">
                <div class="form-group">
                    <label>Timestamp</label>
                    <input type="number" data-field="timestamp" data-index="${index}" value="${msg.timestamp || 0}">
                </div>
                <div class="form-group">
                    <label>Recurrente</label>
                    <select data-field="recurrent" data-index="${index}">
                        <option value="false"${msg.recurrent ? '' : ' selected'}>No</option>
                        <option value="true"${msg.recurrent ? ' selected' : ''}>Sí</option>
                    </select>
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>Intervalo (s)</label>
                    <input type="number" data-field="interval" data-index="${index}" value="${msg.interval || 0}">
                </div>
                <div class="form-group">
                    <label>IP destino</label>
                    <input type="text" data-field="ip" data-index="${index}" value="${msg.ip || ''}" placeholder="192.168.1.10">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>Puerto destino</label>
                    <input type="number" data-field="port" data-index="${index}" value="${msg.port || 0}">
                </div>
            </div>
        `;

        if (state.protocol === 'dnp3') {
            const datalist = index === 0 ? `
                <datalist id="dnp3OperationTypes">
                    <option value="poll_analog_inputs"></option>
                    <option value="poll_binary_inputs"></option>
                    <option value="poll_analog_output_status"></option>
                    <option value="poll_binary_output_status"></option>
                    <option value="poll_group_variation"></option>
                    <option value="poll_group_variation_index"></option>
                    <option value="poll_all"></option>
                    <option value="send_binary_command"></option>
                    <option value="send_analog_command_float32"></option>
                    <option value="send_analog_command_int16"></option>
                    <option value="send_analog_command_int32"></option>
                    <option value="send_analog_command_double64"></option>
                </datalist>
            ` : '';
            return baseFields + `
                <div class="form-row">
                    <div class="form-group">
                        <label>Operation type</label>
                        <input type="text" list="dnp3OperationTypes" data-field="operation_type" data-index="${index}" value="${msg.operation_type || ''}">
                    </div>
                    <div class="form-group">
                        <label>Group</label>
                        <input type="number" data-field="group" data-index="${index}" value="${msg.group || 0}">
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label>Variation</label>
                        <input type="number" data-field="variation" data-index="${index}" value="${msg.variation || 0}">
                    </div>
                    <div class="form-group">
                        <label>Index</label>
                        <input type="number" data-field="index" data-index="${index}" value="${msg.index || 0}">
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label>Master ID</label>
                        <input type="number" data-field="master_id" data-index="${index}" value="${msg.master_id || 1}">
                    </div>
                    <div class="form-group">
                        <label>Outstation ID</label>
                        <input type="number" data-field="outstation_id" data-index="${index}" value="${msg.outstation_id || 1}">
                    </div>
                </div>
                <div class="form-group">
                    <label>Value</label>
                    <input type="text" data-field="value" data-index="${index}" value="${msg.value || ''}">
                </div>
                ${datalist}
            `;
        }

        if (state.protocol === 'iec104') {
            return baseFields + `
                <div class="form-row">
                    <div class="form-group">
                        <label>Type ID</label>
                        <input type="number" data-field="type_id" data-index="${index}" value="${msg.type_id || 0}">
                    </div>
                    <div class="form-group">
                        <label>Common address</label>
                        <input type="number" data-field="common_address" data-index="${index}" value="${msg.common_address || 1}">
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label>IOA</label>
                        <input type="number" data-field="ioa" data-index="${index}" value="${msg.ioa || 1}">
                    </div>
                    <div class="form-group">
                        <label>COT</label>
                        <input type="number" data-field="cot" data-index="${index}" value="${msg.cot || 0}">
                    </div>
                </div>
                <div class="form-group">
                    <label>Value</label>
                    <input type="text" data-field="value" data-index="${index}" value="${msg.value || ''}">
                </div>
            `;
        }

        return baseFields + `
            <div class="form-row">
                <div class="form-group">
                    <label>Slave ID</label>
                    <input type="number" data-field="slave_id" data-index="${index}" value="${msg.slave_id || 0}">
                </div>
                <div class="form-group">
                    <label>Function code</label>
                    <input type="number" data-field="function_code" data-index="${index}" value="${msg.function_code || 3}">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>Start address</label>
                    <input type="text" data-field="start_address" data-index="${index}" value="${msg.start_address || 0}">
                </div>
                <div class="form-group">
                    <label>Count</label>
                    <input type="number" data-field="count" data-index="${index}" value="${msg.count || 1}">
                </div>
            </div>
            <div class="form-group">
                <label>Values</label>
                <input type="text" data-field="values" data-index="${index}" value="${Array.isArray(msg.values) ? msg.values.join(',') : ''}" placeholder="10,12,14">
            </div>
        `;
    }

    function addMessageCard() {
        const current = collectMessagesFromDOM();
        current.push(getEmptyMessage());
        renderMessages(current);
    }

    function collectMessagesFromDOM() {
        const cards = DOM.messagesContainer.querySelectorAll('.message-card');
        const messages = [];
        cards.forEach(card => {
            const message = {};
            card.querySelectorAll('[data-field]').forEach(input => {
                const field = input.dataset.field;
                const value = input.value;
                if (field === 'recurrent') {
                    message.recurrent = value === 'true';
                    return;
                }
                if (field === 'ip') {
                    message.ip = value.trim();
                    return;
                }
                if (['timestamp', 'interval', 'port', 'slave_id', 'group', 'variation', 'index', 'master_id', 'outstation_id', 'type_id', 'common_address', 'ioa', 'cot', 'function_code', 'count'].includes(field)) {
                    message[field] = Number(value) || 0;
                    return;
                }
                if (field === 'values') {
                    message.values = value
                        .split(',')
                        .map(v => v.trim())
                        .filter(v => v !== '')
                        .map(v => (Number.isNaN(Number(v)) ? v : Number(v)));
                    return;
                }
                message[field] = value;
            });
            messages.push(message);
        });
        return messages;
    }

    function getSlaveOptions() {
        return state.cy
            .nodes()
            .filter(node => node.data('role') === 'slave')
            .map(node => ({
                id: node.data('id'),
                label: node.data('name') || node.data('id')
            }));
    }

    function renderAttacksPanel(nodeData) {
        if (!DOM.attacksPanel || !DOM.attacksList) return;
        const masterOptions = getMasterOptions();
        if (masterOptions.length === 0) {
            DOM.attacksPanel.classList.remove('hidden');
            DOM.attacksList.innerHTML = '<div class="message-empty">No hay masters configurados.</div>';
            return;
        }

        if (!attackCatalogLoaded) {
            DOM.attacksPanel.classList.remove('hidden');
            DOM.attacksList.innerHTML = '<div class="message-empty">Cargando catálogo de ataques...</div>';
            return;
        }

        const catalog = getAttackCatalog();
        const masterNode = resolveAttackMasterNode(nodeData);
        const existing = masterNode ? (Array.isArray(masterNode.attacks) ? masterNode.attacks : []) : [];
        const slaveOptions = getSlaveOptions();
        DOM.attacksPanel.classList.remove('hidden');
        DOM.attacksList.innerHTML = '';
        renderAttackMasterSelect(masterOptions, masterNode);

        if (slaveOptions.length === 0) {
            DOM.attacksList.innerHTML = '<div class="message-empty">Agrega al menos un slave para asignar ataques.</div>';
            return;
        }

        if (catalog.length === 0) {
            DOM.attacksList.innerHTML = '<div class="message-empty">No hay ataques disponibles para este protocolo.</div>';
            return;
        }

        catalog.forEach(attack => {
            const current = existing.find(item => item.id === attack.id) || null;
            DOM.attacksList.appendChild(buildAttackCard(attack, current, slaveOptions));
        });
    }

    function getMasterOptions() {
        return state.cy
            .nodes()
            .filter(node => node.data('role') === 'master')
            .map(node => ({
                id: node.data('id'),
                label: node.data('name') || node.data('id')
            }));
    }

    function resolveAttackMasterNode(nodeData) {
        const masterOptions = getMasterOptions();
        if (nodeData && nodeData.role === 'master') {
            state.attacksMasterId = nodeData.id;
            return nodeData;
        }
        if (!state.attacksMasterId && masterOptions.length > 0) {
            state.attacksMasterId = masterOptions[0].id;
        }
        if (!state.attacksMasterId) {
            return null;
        }
        const node = state.cy.$id(state.attacksMasterId);
        return node.length ? node.data() : null;
    }

    function renderAttackMasterSelect(masterOptions, masterNode) {
        if (!DOM.attacksMasterSelect) return;
        const currentId = masterNode?.id || state.attacksMasterId;
        DOM.attacksMasterSelect.innerHTML = masterOptions
            .map(opt => `<option value="${opt.id}" ${opt.id === currentId ? 'selected' : ''}>${opt.label}</option>`)
            .join('');
    }

    function buildAttackCard(attack, current, slaveOptions) {
        const card = document.createElement('div');
        card.className = 'attack-card';
        card.dataset.attackId = attack.id;

        const enabled = current?.enabled ?? false;
        const targetId = current?.target_id || '';
        const scheduleDefaults = attack.schedule_defaults || {};

        const startTime = current?.start_time ?? scheduleDefaults.start_time ?? 0;
        const interval = current?.interval ?? scheduleDefaults.interval ?? 0;
        const count = current?.count ?? scheduleDefaults.count ?? 1;

        const detailsOpen = enabled ? 'open' : '';
        card.innerHTML = `
            <div class="attack-header">
                <div>
                    <div class="attack-title">${attack.name}</div>
                    <div class="attack-technique">${attack.technique_id || 'MITRE ICS'} · ${attack.description || ''}</div>
                </div>
                <label class="attack-toggle">
                    <input type="checkbox" data-attack-field="enabled" ${enabled ? 'checked' : ''}>
                    Activar
                </label>
            </div>
            <details class="attack-details" ${detailsOpen}>
                <summary>Configurar ataque</summary>
                <div class="attack-section">
                    <h4>Destino</h4>
                    <div class="attack-body">
                        <div class="form-group">
                            <label>Target (slave)</label>
                            <select data-attack-field="target_id">
                                <option value="">Selecciona nodo</option>
                                ${slaveOptions.map(opt => `<option value="${opt.id}" ${opt.id === targetId ? 'selected' : ''}>${opt.label}</option>`).join('')}
                            </select>
                        </div>
                    </div>
                </div>
                <div class="attack-section">
                    <h4>Planificación</h4>
                    <div class="attack-body">
                        <div class="form-group">
                            <label>Inicio (s)</label>
                            <input type="number" min="0" data-attack-field="start_time" value="${startTime}">
                        </div>
                        <div class="form-group">
                            <label>Intervalo (s)</label>
                            <input type="number" min="0" data-attack-field="interval" value="${interval}">
                        </div>
                        <div class="form-group">
                            <label>Repeticiones</label>
                            <input type="number" min="1" data-attack-field="count" value="${count}">
                        </div>
                    </div>
                </div>
                <div class="attack-section">
                    <h4>Parámetros</h4>
                    ${buildAttackParams(attack, current?.parameters || {})}
                </div>
            </details>
        `;

        return card;
    }

    function buildAttackParams(attack, params) {
        if (!Array.isArray(attack.parameters) || attack.parameters.length === 0) {
            return '<div class="message-empty">Sin parámetros adicionales.</div>';
        }
        const fields = attack.parameters.map(param => buildAttackParamField(param, params)).join('');
        return `<div class="attack-body">${fields}</div>`;
    }

    function buildAttackParamField(param, params) {
        const current = params?.[param.name];
        const value = current !== undefined && current !== null ? current : (param.default ?? '');
        const label = param.label || param.name;
        const description = param.description ? `<span class="hint">${param.description}</span>` : '';
        const type = param.type || 'string';

        if (type === 'bool') {
            return `
                <div class="form-group">
                    <label>${label}</label>
                    <div class="checkbox-row">
                        <input type="checkbox" data-attack-param="${param.name}" ${value ? 'checked' : ''}>
                        ${description}
                    </div>
                </div>
            `;
        }

        const inputType = type.startsWith('int') || type.startsWith('float') ? 'number' : 'text';
        const placeholder = param.placeholder || (type.startsWith('list') ? '10,20,30' : '');
        return `
            <div class="form-group">
                <label>${label}</label>
                <input type="${inputType}" data-attack-param="${param.name}" data-attack-param-type="${type}" value="${value}" placeholder="${placeholder}">
                ${description}
            </div>
        `;
    }

    function collectAttacksFromDOM() {
        if (!DOM.attacksList) return [];
        const cards = DOM.attacksList.querySelectorAll('.attack-card');
        const attacks = [];
        cards.forEach(card => {
            const attackId = card.dataset.attackId;
            const fields = card.querySelectorAll('[data-attack-field]');
            const data = { id: attackId };
            fields.forEach(field => {
                const key = field.dataset.attackField;
                if (key === 'enabled') {
                    data.enabled = field.checked;
                    return;
                }
                if (['start_time', 'interval', 'count'].includes(key)) {
                    data[key] = Number(field.value) || 0;
                    return;
                }
                if (key === 'target_id') {
                    data.target_id = field.value;
                    return;
                }
                data[key] = field.value;
            });

            const catalog = getAttackCatalog();
            const definition = catalog.find(item => item.id === attackId);
            if (definition) {
                data.technique_id = definition.technique_id;
                data.name = definition.name;
                data.description = definition.description;
            }

            data.parameters = {};
            const paramInputs = card.querySelectorAll('[data-attack-param]');
            paramInputs.forEach(input => {
                const name = input.dataset.attackParam;
                const paramType = input.dataset.attackParamType || 'string';
                if (!name) return;
                if (input.type === 'checkbox') {
                    data.parameters[name] = input.checked;
                    return;
                }
                data.parameters[name] = parseParamInputValue(paramType, input.value);
            });

            attacks.push(data);
        });
        return attacks;
    }

    function saveAttacksToSelectedMaster() {
        const masterId = DOM.attacksMasterSelect?.value || state.attacksMasterId;
        if (!masterId) {
            showToast('Selecciona un master para guardar ataques', true);
            return;
        }
        const node = state.cy.$id(masterId);
        if (!node.length) {
            showToast('Master no encontrado', true);
            return;
        }
        const data = node.data();
        data.attacks = collectAttacksFromDOM();
        node.data(data);
        showToast('Ataques guardados en el master');
    }

    function parseParamInputValue(paramType, value) {
        const type = (paramType || 'string').toLowerCase();
        if (type === 'int') {
            return Number(value) || 0;
        }
        if (type === 'float') {
            return Number(value) || 0;
        }
        if (type === 'bool') {
            return value === 'true' || value === '1';
        }
        if (type.startsWith('list')) {
            return value
                .split(',')
                .map(entry => entry.trim())
                .filter(entry => entry !== '')
                .map(entry => (type === 'list_int' || type === 'list_float') ? Number(entry) || 0 : entry);
        }
        return value;
    }

    function toggleAttackDetails(open) {
        if (!DOM.attacksList) return;
        DOM.attacksList.querySelectorAll('.attack-details').forEach(details => {
            details.open = open;
        });
    }

    function saveEdge() {
        if (!state.selectedElement || !state.selectedElement.isEdge()) return;
        const messages = collectMessagesFromDOM();
        const context = getEdgeContext(state.selectedElement);
        const defaults = buildMessageDefaults(context);
        const normalized = messages.map(message => applyMessageDefaults(message, defaults));
        state.selectedElement.data('messages', normalized);
        showToast('Enlace actualizado');
    }

    function handleCanvasTap(evt) {
        if (evt.target === state.cy) {
            deselectElement();
            return;
        }
        if (evt.target.isNode()) {
            if (state.selectedElement && state.selectedElement.isNode() && state.selectedElement.id() !== evt.target.id()) {
                const src = state.selectedElement;
                const dst = evt.target;
                if (src.data('role') === dst.data('role')) {
                    showToast('Los enlaces deben conectar roles distintos', true);
                    return;
                }
                const source = src.data('role') === 'master' ? src.id() : dst.id();
                const target = src.data('role') === 'master' ? dst.id() : src.id();
                addEdge(source, target);
                deselectElement();
            } else {
                selectElement(evt.target);
            }
            return;
        }

        if (evt.target.isEdge && evt.target.isEdge()) {
            selectElement(evt.target);
        }
    }

    function handleKeydown(evt) {
        if (!state.selectedElement) return;
        if (evt.key === 'Delete' || evt.key === 'Supr') {
            state.selectedElement.remove();
            state.selectedElement = null;
            updateCounts();
            DOM.nodePanel.classList.add('hidden');
            DOM.edgePanel.classList.add('hidden');
            if (state.cy.nodes().length === 0) {
                DOM.emptyState.style.display = 'flex';
            }
        }
        if (evt.key === 'Escape') {
            deselectElement();
        }
    }

    function layoutGraph() {
        state.cy.layout({ name: 'cose', animate: true, padding: 80 }).run();
    }

    function openRunModal() {
        DOM.runModal.style.display = 'flex';
        DOM.runModal.setAttribute('aria-hidden', 'false');
    }

    function closeRunModal() {
        DOM.runModal.style.display = 'none';
        DOM.runModal.setAttribute('aria-hidden', 'true');
    }

    async function saveScenario() {
        const payload = exportScenario();
        try {
            await fetchJSON(`/api/networks/${encodeURIComponent(state.scenarioId)}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            showToast('Escenario guardado');
        } catch (err) {
            showToast(err.message, true);
        }
    }

    function exportScenario() {
        const nodes = state.cy.nodes().map(node => ({
            data: node.data(),
            classes: node.classes(),
            position: node.position()
        }));
        const edges = state.cy.edges().map(edge => ({
            data: edge.data()
        }));
        return {
            protocol: state.protocol,
            ip_network: networkData?.ip_network || '',
            nodes,
            edges
        };
    }

    async function runScenario() {
        const simulationTime = Number(DOM.simulationTime.value) || 60;
        const payload = exportScenario();
        payload.simulation_time = simulationTime;

        closeRunModal();
        try {
            await fetchJSON('/api/run', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            openRunOverlay();
            startPolling();
        } catch (err) {
            showToast(err.message, true);
        }
    }

    function openRunOverlay() {
        DOM.runOverlay.style.display = 'flex';
        DOM.runOverlay.setAttribute('aria-hidden', 'false');
        DOM.progressTime.textContent = '0';
        DOM.progressPercent.textContent = '0%';
        DOM.progressPcap.textContent = '0 KB';
    }

    function closeRunOverlay() {
        DOM.runOverlay.style.display = 'none';
        DOM.runOverlay.setAttribute('aria-hidden', 'true');
    }

    async function pollRunStatus() {
        try {
            const status = await fetchJSON('/api/run');
            if (status.total_seconds) {
                const percent = Math.min(100, Math.floor((status.elapsed_seconds / status.total_seconds) * 100));
                DOM.progressTime.textContent = status.elapsed_seconds;
                DOM.progressPercent.textContent = `${percent}%`;
                DOM.progressPcap.textContent = `${status.pcap_size || 0} KB`;
                if (!status.running || status.elapsed_seconds >= status.total_seconds) {
                    stopPolling();
                    setTimeout(() => {
                        closeRunOverlay();
                        showToast('Simulación completada');
                    }, 800);
                }
            }
        } catch (err) {
            stopPolling();
            closeRunOverlay();
            showToast(err.message, true);
        }
    }

    function startPolling() {
        stopPolling();
        state.pollInterval = setInterval(pollRunStatus, 1000);
    }

    function stopPolling() {
        if (state.pollInterval) {
            clearInterval(state.pollInterval);
            state.pollInterval = null;
        }
    }

    async function stopRun() {
        try {
            await fetchJSON('/api/run', { method: 'DELETE' });
            showToast('Simulación detenida');
        } catch (err) {
            showToast(err.message, true);
        } finally {
            stopPolling();
            closeRunOverlay();
        }
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

    function bindEvents() {
        state.cy.on('tap', handleCanvasTap);
        state.cy.on('taphold', evt => {
            if (evt.target === state.cy) {
                addNode(evt.position);
            }
        });
        document.addEventListener('keydown', handleKeydown);

        DOM.btnAddNode.addEventListener('click', () => addNode());
        DOM.btnLayout.addEventListener('click', layoutGraph);
        DOM.btnSave.addEventListener('click', saveScenario);
        DOM.btnRun.addEventListener('click', openRunModal);

        DOM.nodeForm.addEventListener('submit', saveNode);
        DOM.nodeRole.addEventListener('change', updateProtocolVisibility);
        DOM.cancelNode.addEventListener('click', deselectElement);
        DOM.nodeRole.addEventListener('change', () => {
            if (!state.selectedElement || !state.selectedElement.isNode()) return;
            renderAttacksPanel(state.selectedElement.data());
        });
        if (DOM.jumpAttacks) {
            DOM.jumpAttacks.addEventListener('click', () => {
                if (!DOM.attacksPanel) return;
                DOM.attacksPanel.classList.remove('hidden');
                DOM.attacksPanel.scrollIntoView({ behavior: 'smooth', block: 'start' });
            });
        }
        if (DOM.expandAttacks) {
            DOM.expandAttacks.addEventListener('click', () => toggleAttackDetails(true));
        }
        if (DOM.collapseAttacks) {
            DOM.collapseAttacks.addEventListener('click', () => toggleAttackDetails(false));
        }
        if (DOM.attacksMasterSelect) {
            DOM.attacksMasterSelect.addEventListener('change', event => {
                state.attacksMasterId = event.target.value;
                renderAttacksPanel();
            });
        }
        if (DOM.saveAttacks) {
            DOM.saveAttacks.addEventListener('click', saveAttacksToSelectedMaster);
        }

        DOM.addMessage.addEventListener('click', addMessageCard);
        DOM.saveEdge.addEventListener('click', saveEdge);

        DOM.messagesContainer.addEventListener('click', event => {
            if (event.target.dataset.action === 'remove-message') {
                const current = collectMessagesFromDOM();
                const index = Number(event.target.closest('.message-card')?.dataset.index);
                if (Number.isNaN(index)) return;
                current.splice(index, 1);
                renderMessages(current.length ? current : [getEmptyMessage()]);
            }
        });

        DOM.cancelRun.addEventListener('click', closeRunModal);
        DOM.confirmRun.addEventListener('click', runScenario);
        DOM.stopRun.addEventListener('click', stopRun);
    }

    function init() {
        initHeader();
        initCytoscape();
        updateCounts();
        bindEvents();
        loadAttackCatalog().then(() => {
            if (state.selectedElement && state.selectedElement.isNode()) {
                renderAttacksPanel(state.selectedElement.data());
            }
        });
    }

    init();
})();
