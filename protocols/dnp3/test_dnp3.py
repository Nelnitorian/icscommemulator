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

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)

DNP3_AVAILABLE = True
from slave import DNP3Outstation
from master import DNP3Master


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
def master_csv_config(test_port, tmp_path_factory):
    csv_content = f"""timestamp,ip,port,operation_type,group,variation,index,master_id,outstation_id,recurrent,interval,value
0,127.0.0.1,{test_port},poll_analog_inputs,30,6,,20,10,false,0,
1,127.0.0.1,{test_port},poll_binary_inputs,1,2,,20,10,false,0,
2,127.0.0.1,{test_port},poll_analog_output_status,40,2,,20,10,false,0,
3,127.0.0.1,{test_port},poll_binary_output_status,10,2,,20,10,false,0,
4,127.0.0.1,{test_port},poll_analog_inputs,30,6,,20,10,true,2,
"""
    config_dir = tmp_path_factory.mktemp("config")
    csv_file = config_dir / "master.csv"
    csv_file.write_text(csv_content)
    yield str(csv_file)


@pytest.fixture(scope="module")
def outstation(test_port, slave_yaml_config):
    if not DNP3_AVAILABLE:
        pytest.skip("dnp3-python not available")
    station = DNP3Outstation(config_file=slave_yaml_config)
    station.start()
    time.sleep(1)
    yield station
    station.shutdown()


@pytest.fixture(scope="module")
def master(test_port, master_csv_config, outstation):
    if not DNP3_AVAILABLE:
        pytest.skip("dnp3-python not available")
    controller = DNP3Master(
        outstation_ip="127.0.0.1",
        outstation_port=test_port,
        master_id=20,
        outstation_id=10,
        config_file=master_csv_config,
    )
    assert controller.connect()
    time.sleep(2)
    yield controller
    controller.shutdown()


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

    def test_csv_operations_loaded(self, master):
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

        master.execute_all_operations()
        time.sleep(3)

        assert master.stats["operations_executed"] > initial_ops
        assert master.stats["polls_sent"] > initial_polls
        values = master.get_all_values()
        assert len(values) > 0

    def test_operation_statistics(self, master):
        stats = master.stats
        assert stats["polls_sent"] >= 4  # At least 4 poll operations

    def test_recurrent_operations(self, master):
        initial_ops = master.stats["operations_executed"]
        time.sleep(5)  # Wait for recurrent operations
        assert master.stats["operations_executed"] > initial_ops + 1


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


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-m", "integration", "--tb=short"])
