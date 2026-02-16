window.App = window.App || {};

(function () {
    'use strict';
    const CONSTANTS = App.CONSTANTS;
    const t = (key, vars) => (window.I18N ? window.I18N.t(key, vars) : key);

    const ProtocolConfig = {
        MODBUS: {
            name: 'modbus',
            defaultPort: '502',
            nodeTemplate: {
                role: 'slave', port: '502', slave_id: '1',
                holding_registers: { type: 'sequential', values: '' },
                coils: { type: 'sequential', values: '' },
                discrete_inputs: { type: 'sequential', values: '' },
                input_registers: { type: 'sequential', values: '' },
                identity: { vendor_name: '' }, comment: ''
            },
            masterFields: ['port'], slaveFields: ['port', 'slave_id'],
            hasRegisters: true, hasIdentity: true
        },
        DNP3: {
            name: 'dnp3',
            defaultPort: '20000',
            nodeTemplate: {
                role: 'slave', port: '20000', outstation_id: '1', master_id: '2',
                analog_inputs: { count: 20, initial_values: [] },
                binary_inputs: { count: 20, initial_values: [] },
                analog_output_status: { count: 10, initial_values: [] },
                binary_output_status: { count: 10, initial_values: [] },
                simulation: { enabled: false, interval: 1000 }, comment: ''
            },
            masterFields: ['master_id'], slaveFields: ['port', 'outstation_id'],
            hasRegisters: false, hasIdentity: false
        },
        IEC104: {
            name: 'iec104',
            defaultPort: '2404',
            nodeTemplate: {
                role: 'slave', port: '2404', common_address: 1, tick_rate_ms: 100,
                select_timeout_ms: 10000, max_connections: 5, authorized_masters: [],
                k: 12, w: 8, t1: 15, t2: 10, t3: 20,
                single_points: {}, measured_short: {}, comment: ''
            },
            masterFields: ['port'],
            slaveFields: ['port', 'common_address', 'tick_rate_ms', 'max_connections', 'k', 'w', 't1', 't2', 't3'],
            hasRegisters: false, hasIdentity: false, hasIEC104Points: true
        },
        getCurrentProtocol() {
            const p = (networkData?.protocol || 'modbus').toLowerCase();
            return this[p.toUpperCase()] || this.MODBUS;
        },
        getNodeTemplate() { return JSON.parse(JSON.stringify(this.getCurrentProtocol().nodeTemplate)); },
        getDefaultPort() { return this.getCurrentProtocol().defaultPort; }
    };

    class Validators {
        static validateIP(ip) { return ipaddr.isValid(ip); }
        static validateMAC(mac) { return /^(?:[0-9A-Fa-f]{2}([:\-.]?)){3,5}[0-9A-Fa-f]{2}$/.test(mac); }
        static parseMACToColonFormat(mac, fallback) {
            if (!this.validateMAC(mac)) return fallback;
            const clean = mac.replace(/[^0-9A-Fa-f]/g, '');
            if (![8, 10, 12].includes(clean.length)) return fallback;
            return clean.match(/.{2}/g).join(':').toUpperCase();
        }
        static validateMessageFields(msg) {
            if (msg.timestamp < 0) return false;
            if (msg.recurrent && msg.interval <= 0) return false;
            return true;
        }
        static checkCharacters(str, includeColon) {
            const regex = includeColon ? /[^0-9:\s,]/ : /[^0-9\s,]/;
            return !regex.test(str);
        }
        static parseRegisterValues(type, values) {
            if (!values) return -1;
            const isSparse = type === CONSTANTS.REGISTER_TYPES.SPARSE;
            if (!this.checkCharacters(values, isSparse)) {
                if (App.UIUtils) App.UIUtils.showError(t('protocols.registerFormatError'));
                return null;
            }
            if (!isSparse) return values.split(',').map(Number);

            const result = values.split(',').reduce((acc, pair) => {
                const [k, v] = pair.split(':').map(i => i.trim());
                acc[k] = parseInt(v, 10);
                return acc;
            }, {});
            return result;
        }
    }

    class MessageProcessor {
        static processEdgeMessages(edge, messages) {
            const protocol = (networkData?.protocol || 'modbus').toLowerCase();
            if (protocol === 'dnp3') return this.processDNP3Messages(edge, messages);
            if (protocol === 'iec104') return this.processIEC104Messages(edge, messages);
            return messages;
        }

        static processDNP3Messages(edge, messages) {
            const src = edge.source();
            const tgt = edge.target();
            const master = src.data('role') === CONSTANTS.ROLES.MASTER ? src : tgt;
            const slave = src.data('role') === CONSTANTS.ROLES.SLAVE ? src : tgt;
            return messages.map(msg => ({
                ...msg,
                master_id: master.data('master_id') || 1,
                outstation_id: slave.data('outstation_id') || 1
            }));
        }

        static processIEC104Messages(edge, messages) {
            const tgt = edge.target();
            return messages.map(msg => ({
                ...msg,
                common_address: tgt.data('common_address') || 1
            }));
        }
    }

    class ClientValidator {
        static validateScenario() {
            if (!window.cy) return { errors: ['System error: Cytoscape not initialized'], warnings: [] };

            const errors = [];
            const warnings = [];
            const nodes = window.cy.nodes();
            const edges = window.cy.edges();

            if (nodes.length === 0) errors.push('The network must have at least one node');

            const nodeIds = new Set();
            nodes.forEach(node => {
                const id = node.data('id');
                if (nodeIds.has(id)) errors.push(`Duplicate ID: ${id}`);
                nodeIds.add(id);

                const ip = node.data('ip');
                if (!ip) errors.push(`Node ${node.data('name')} has no IP`);
                else if (!Validators.validateIP(ip)) errors.push(`Node ${node.data('name')} invalid IP`);
            });

            return { errors, warnings };
        }

        static showValidationResults(errors, warnings) {
            if (errors.length > 0) {
                alert('Validation errors:\n\n' + errors.join('\n'));
                return false;
            }
            if (warnings.length > 0) {
                return confirm('Validation warnings:\n\n' + warnings.join('\n') + '\n\nContinue?');
            }
            return true;
        }
    }

    App.ProtocolConfig = ProtocolConfig;
    App.Validators = Validators;
    App.MessageProcessor = MessageProcessor;
    App.ClientValidator = ClientValidator;
}());
