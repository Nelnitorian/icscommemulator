// ============================================================================
// MODULE: Constants & Configuration
// ============================================================================

const CONSTANTS = {
    KEYS: {
        DELETE: 'Delete',
        SUPR: 'Supr',
        ESCAPE: 'Escape',
        Z: 'z',
        Y: 'y'
    },
    ACTIONS: {
        ADD: 'add',
        DELETE: 'delete'
    },
    ROLES: {
        MASTER: 'master',
        SLAVE: 'slave'
    },
    EDGE: {
        ARROW_SHAPE: 'triangle'
    },
    LOG_LEVEL: {
        ERROR: 1,
        WARNING: 2
    },
    REGISTER_TYPES: {
        SEQUENTIAL: 'sequential',
        SPARSE: 'sparse'
    },
    FUNCTION_CODES: [1, 2, 3, 4, 5, 6, 15, 16, 43],
    TAPHOLD_DELAY: 500
};

const CYTOSCAPE_STYLE = [
    {
        selector: 'node',
        style: { 'background-color': '#666', 'label': 'data(name)' }
    },
    {
        selector: 'node.master',
        style: { 'background-color': '#3A86FF' }
    },
    {
        selector: 'node.slave',
        style: { 'background-color': '#FF6D00' }
    },
    {
        selector: 'node.selected',
        style: { 'background-color': '#FFA500' }
    },
    {
        selector: 'edge',
        style: {
            'width': 3,
            'line-color': '#ccc',
            'curve-style': 'bezier',
            'target-arrow-color': '#ccc',
            'target-arrow-shape': 'triangle',
            'arrow-scale': '1'
        }
    },
    {
        selector: 'edge.selected',
        style: {
            'line-color': '#FFA500',
            'target-arrow-color': '#FFA500'
        }
    }
];

const DEFAULT_NODE_TEMPLATE = {
    role: CONSTANTS.ROLES.SLAVE,
    holding_registers: { type: 'sequential', values: '' },
    coils: { type: 'sequential', values: '' },
    discrete_inputs: { type: 'sequential', values: '' },
    input_registers: { type: 'sequential', values: '' },
    comment: '',
    port: '502',
    slave_id: '1',
    identity: {
        major_minor_revision: "",
        model_name: "",
        product_code: "",
        product_name: "",
        user_application_name: "",
        vendor_name: "",
        vendor_url: ""
    }
};
