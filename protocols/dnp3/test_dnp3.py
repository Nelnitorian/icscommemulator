#!/usr/bin/env python3

"""
Integration tests for DNP3 Master-Outstation communication with configuration files.
Tests only functionality supported by the dnp3-python library.
"""

import pytest
import time
import logging
import tempfile
import os
import sys

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)

# Ensure command callbacks don't crash under pytest runs.
os.environ.setdefault("DNP3_DISABLE_COMMAND_HANDLER", "1")

SKIP_COMMAND_TESTS = (
    sys.version_info >= (3, 12)
    or os.getenv("DNP3_SKIP_COMMAND_TESTS") == "1"
)

try:
    import slave.slave as dnp3_slave
    import master.master as dnp3_master
    DNP3Outstation = dnp3_slave.DNP3Outstation
    DNP3Master = dnp3_master.DNP3Master
    DNP3Operation = dnp3_master.DNP3Operation
    DNP3_AVAILABLE = bool(dnp3_slave.DNP3_AVAILABLE and dnp3_master.DNP3_AVAILABLE)
    _IMPORT_ERROR = dnp3_slave._IMPORT_ERROR or dnp3_master._IMPORT_ERROR
except Exception as e:
    DNP3Outstation = None
    DNP3Master = None
    DNP3Operation = None
    DNP3_AVAILABLE = False
    _IMPORT_ERROR = e


@pytest.fixture(scope="module")
def test_port():
    return 20099


@pytest.fixture(scope="module")
def slave_yaml_config(test_port, tmp_path_factory):
    yaml_content = f"""ip: 127.0.0.1
port: {test_port}
outstation_id: 10
master_id: 20
analog_inputs:
  count: 20
  initial_values:
    - index: 0
      value: 100.0
    - index: 1
      value: 200.5
    - index: 10
      value: -25.3
binary_inputs:
  count: 20
  initial_values:
    - index: 0
      value: true
    - index: 1
      value: false
analog_output_status:
  count: 10
  initial_values:
    - index: 0
      value: 50.0
binary_output_status:
  count: 10
  initial_values:
    - index: 0
      value: false
simulation:
  enabled: false
"""
    # Crear directorio temporal que simula /app/config
    config_dir = tmp_path_factory.mktemp("config")
    yaml_file = config_dir / "slave.yaml"
    yaml_file.write_text(yaml_content)
    yield str(yaml_file)


@pytest.fixture(scope="module")
def master_yaml_config(test_port, tmp_path_factory):
    yaml_content = f"""protocol: dnp3
messages:
  - timestamp: 0
    ip: 127.0.0.1
    port: {test_port}
    operation_type: poll_analog_inputs
    group: 30
    variation: 6
    master_id: 20
    outstation_id: 10
    recurrent: false
    interval: 0
  - timestamp: 1
    ip: 127.0.0.1
    port: {test_port}
    operation_type: poll_binary_inputs
    group: 1
    variation: 2
    master_id: 20
    outstation_id: 10
    recurrent: false
    interval: 0
  - timestamp: 2
    ip: 127.0.0.1
    port: {test_port}
    operation_type: poll_analog_output_status
    group: 40
    variation: 2
    master_id: 20
    outstation_id: 10
    recurrent: false
    interval: 0
  - timestamp: 3
    ip: 127.0.0.1
    port: {test_port}
    operation_type: poll_binary_output_status
    group: 10
    variation: 2
    master_id: 20
    outstation_id: 10
    recurrent: false
    interval: 0
  - timestamp: 4
    ip: 127.0.0.1
    port: {test_port}
    operation_type: poll_analog_inputs
    group: 30
    variation: 6
    master_id: 20
    outstation_id: 10
    recurrent: true
    interval: 2
"""
    config_dir = tmp_path_factory.mktemp("config")
    yaml_file = config_dir / "master.yaml"
    yaml_file.write_text(yaml_content)
    yield str(yaml_file)


@pytest.fixture(scope="module")
def outstation(test_port, slave_yaml_config):
    if not DNP3_AVAILABLE:
        pytest.skip(f"dnp3-python not available: {_IMPORT_ERROR}")
    station = DNP3Outstation(config_file=slave_yaml_config)
    station.start()
    time.sleep(1)
    yield station
    station.shutdown()


@pytest.fixture(scope="module")
def master(test_port, master_yaml_config, outstation):
    if not DNP3_AVAILABLE:
        pytest.skip(f"dnp3-python not available: {_IMPORT_ERROR}")
    controller = DNP3Master(
        outstation_ip="127.0.0.1",
        outstation_port=test_port,
        master_id=20,
        outstation_id=10,
        config_file=master_yaml_config,
    )
    assert controller.connect()
    time.sleep(2)
    yield controller
    controller.shutdown()


def stop_recurrent(master):
    master.running = False
    for thread in list(master.recurrent_threads):
        if thread.is_alive():
            thread.join(timeout=1)
    master.recurrent_threads = []


# Connection Tests
@pytest.mark.integration
class TestConnection:
    """Test DNP3 connection establishment."""

    def test_outstation_started(self, outstation):
        assert outstation is not None
        assert outstation.outstation is not None
        assert outstation.ip == "127.0.0.1"

    def test_master_connected(self, master):
        assert master is not None
        assert master.master is not None

    def test_yaml_operations_loaded(self, master):
        assert len(master.operations) == 5
        recurrent_ops = [op for op in master.operations if op.recurrent]
        assert len(recurrent_ops) == 1

    def test_yaml_config_structure(self, outstation):
        config = outstation.config
        assert "analog_inputs" in config
        assert config["analog_inputs"]["count"] == 20


# Analog Input Tests
@pytest.mark.integration
class TestAnalogInputs:
    """Test analog input polling."""

    def test_poll_analog_inputs(self, master, outstation):
        test_values = {0: 42.5, 1: 99.9, 5: 777.123}
        for idx, val in test_values.items():
            assert outstation.update_analog_input(idx, val)
        time.sleep(0.5)

        master.poll_analog_inputs(group=30, variation=6)
        time.sleep(1)

        values = master.get_all_values()
        assert "Analog" in values
        assert len(values["Analog"]) > 0

    def test_yaml_initial_analog_values(self, outstation):
        config = outstation.config
        initial_values = {
            item["index"]: item["value"]
            for item in config["analog_inputs"]["initial_values"]
        }
        assert initial_values[0] == 100.0
        assert initial_values[1] == 200.5
        assert initial_values[10] == -25.3

    def test_analog_value_range(self, master, outstation):
        extreme_values = {0: 0.0, 1: -1000.5, 2: 9999.99, 3: 0.001}
        for idx, val in extreme_values.items():
            assert outstation.update_analog_input(idx, val)

        time.sleep(0.5)
        master.poll_analog_inputs()
        time.sleep(1)
        values = master.get_all_values()
        assert "Analog" in values


# Binary Input Tests
@pytest.mark.integration
class TestBinaryInputs:
    """Test binary input polling."""

    def test_poll_binary_inputs(self, master, outstation):
        test_values = {0: True, 1: False, 5: True}
        for idx, val in test_values.items():
            assert outstation.update_binary_input(idx, val)
        time.sleep(0.5)

        master.poll_binary_inputs(group=1, variation=2)
        time.sleep(1)

        values = master.get_all_values()
        assert "Binary" in values
        assert len(values["Binary"]) > 0

    def test_yaml_initial_binary_values(self, outstation):
        config = outstation.config
        initial_values = {
            item["index"]: item["value"]
            for item in config["binary_inputs"]["initial_values"]
        }
        assert initial_values[0] is True
        assert initial_values[1] is False


# Output Status Tests
@pytest.mark.integration
class TestOutputStatus:
    """Test analog and binary output status polling."""

    def test_poll_analog_output_status(self, master, outstation):
        test_values = {0: 123.45, 1: 678.90}
        for idx, val in test_values.items():
            assert outstation.update_analog_output_status(idx, val)
        time.sleep(0.5)

        master.poll_analog_output_status(group=40, variation=2)
        time.sleep(1)

        values = master.get_all_values()
        assert "AnalogOutputStatus" in values

    def test_poll_binary_output_status(self, master, outstation):
        test_values = {0: True, 1: False}
        for idx, val in test_values.items():
            assert outstation.update_binary_output_status(idx, val)
        time.sleep(0.5)

        master.poll_binary_output_status(group=10, variation=2)
        time.sleep(1)

        values = master.get_all_values()
        assert "BinaryOutputStatus" in values


# CSV Operations Tests
@pytest.mark.integration
class TestCSVOperations:
    """Test execution of operations from CSV file."""

    def test_execute_csv_operations(self, master, outstation):
        outstation.update_analog_input(0, 42.5)
        time.sleep(0.5)

        initial_ops = master.stats["operations_executed"]
        initial_polls = master.stats["polls_sent"]

        try:
            master.execute_all_operations()
            time.sleep(3)

            assert master.stats["operations_executed"] > initial_ops
            assert master.stats["polls_sent"] > initial_polls
            values = master.get_all_values()
            assert len(values) > 0
        finally:
            stop_recurrent(master)

    def test_operation_statistics(self, master):
        stats = master.stats
        assert stats["polls_sent"] >= 4  # At least 4 poll operations

    def test_recurrent_operations(self, master):
        master.execute_all_operations()
        initial_ops = master.stats["operations_executed"]
        try:
            time.sleep(5)  # Wait for recurrent operations
            assert master.stats["operations_executed"] > initial_ops + 1
        finally:
            stop_recurrent(master)


# Bidirectional Communication Tests
@pytest.mark.integration
class TestBidirectionalCommunication:
    """Test bidirectional data flow."""

    def test_outstation_to_master_analog(self, outstation, master):
        test_index, test_value = 15, 888.777
        assert outstation.update_analog_input(test_index, test_value)
        time.sleep(0.5)

        master.poll_analog_inputs()
        time.sleep(1)

        values = master.get_all_values()
        assert "Analog" in values

    def test_multiple_point_types(self, outstation, master):
        outstation.update_analog_input(0, 111.1)
        outstation.update_binary_input(0, True)
        time.sleep(0.5)

        master.poll_analog_inputs()
        time.sleep(0.3)
        master.poll_binary_inputs()
        time.sleep(1)

        values = master.get_all_values()
        assert "Analog" in values
        assert "Binary" in values


# Command Tests
@pytest.mark.integration
@pytest.mark.skipif(
    SKIP_COMMAND_TESTS,
    reason="DirectOperate crashes under Python 3.12 for current pydnp3 build",
)
class TestCommands:
    """Test control commands."""

    def test_send_binary_command(self, master):
        stop_recurrent(master)
        master_connection = master.master
        assert master_connection is not None
        initial = master.stats["commands_sent"]
        assert master.send_binary_command(master_connection, 0, True)
        assert master.stats["commands_sent"] == initial + 1

    def test_send_analog_command_float32(self, master):
        stop_recurrent(master)
        master_connection = master.master
        assert master_connection is not None
        initial = master.stats["commands_sent"]
        assert master.send_analog_command_float32(master_connection, 0, 12.5)
        assert master.stats["commands_sent"] == initial + 1

    def test_send_analog_command_int16(self, master):
        stop_recurrent(master)
        master_connection = master.master
        assert master_connection is not None
        initial = master.stats["commands_sent"]
        assert master.send_analog_command_int16(master_connection, 0, 123)
        assert master.stats["commands_sent"] == initial + 1

    def test_send_analog_command_int32(self, master):
        stop_recurrent(master)
        master_connection = master.master
        assert master_connection is not None
        initial = master.stats["commands_sent"]
        assert master.send_analog_command_int32(master_connection, 0, 12345)
        assert master.stats["commands_sent"] == initial + 1

    def test_send_analog_command_double64(self, master):
        stop_recurrent(master)
        master_connection = master.master
        assert master_connection is not None
        initial = master.stats["commands_sent"]
        assert master.send_analog_command_double64(master_connection, 0, 123.456)
        assert master.stats["commands_sent"] == initial + 1


# Poll variants not covered by YAML
@pytest.mark.integration
class TestPollVariants:
    """Test poll variants not covered by base YAML."""

    def test_poll_group_variation(self, master):
        op = DNP3Operation({
            "operation_type": "poll_group_variation",
            "group": 30,
            "variation": 6,
            "master_id": master.master_id,
            "outstation_id": master.outstation_id,
        })
        initial = master.stats["polls_sent"]
        master.execute_operation(op)
        assert master.stats["polls_sent"] >= initial + 1

    def test_poll_group_variation_index(self, master):
        op = DNP3Operation({
            "operation_type": "poll_group_variation_index",
            "group": 30,
            "variation": 6,
            "index": 0,
            "master_id": master.master_id,
            "outstation_id": master.outstation_id,
        })
        initial = master.stats["polls_sent"]
        master.execute_operation(op)
        assert master.stats["polls_sent"] >= initial + 1

    def test_poll_all(self, master):
        op = DNP3Operation({
            "operation_type": "poll_all",
            "master_id": master.master_id,
            "outstation_id": master.outstation_id,
        })
        initial = master.stats["polls_sent"]
        master.execute_operation(op)
        assert master.stats["polls_sent"] >= initial + 1


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-m", "integration", "--tb=short"])
