#!/usr/bin/env python3
"""
DNP3 Master Station Implementation with YAML/CSV Configuration

Implements a DNP3 master station supporting operations configured via YAML
(preferred) or CSV (legacy).

Requires: dnp3-python >= 2.13.6
"""

import logging
import time
import sys
import threading
import csv
import os
from typing import Dict, Any, List, Optional

import yaml

try:
    from dnp3_python.dnp3station.master import MyMasterNew
    from dnp3_python.dnp3station.station_utils import SOEHandler
    from pydnp3 import opendnp3, asiodnp3, openpal
    DNP3_AVAILABLE = True
    _IMPORT_ERROR = None
except ImportError as e:
    MyMasterNew = None
    SOEHandler = None
    opendnp3 = None
    asiodnp3 = None
    openpal = None
    DNP3_AVAILABLE = False
    _IMPORT_ERROR = e

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)

logger = logging.getLogger(__name__)


def _to_int(value: Any, default: Optional[int] = None) -> Optional[int]:
    if value is None or value == "":
        return default
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def _to_float(value: Any, default: float = 0.0) -> float:
    if value is None or value == "":
        return default
    try:
        return float(value)
    except (TypeError, ValueError):
        return default


def _to_bool(value: Any, default: bool = False) -> bool:
    if value is None:
        return default
    if isinstance(value, bool):
        return value
    return str(value).strip().lower() in ("true", "1", "yes", "y")


class DNP3Operation:
    """Represents a single DNP3 operation from YAML/CSV configuration."""

    def __init__(self, raw: Dict[str, Any]):
        self.timestamp = _to_float(raw.get("timestamp"), 0.0) or 0.0
        self.ip = raw.get("ip") or ""
        self.port = _to_int(raw.get("port"), 0) or 0
        self.operation_type = (raw.get("operation_type") or "").strip()
        self.group = _to_int(raw.get("group"))
        self.variation = _to_int(raw.get("variation"))
        self.index = _to_int(raw.get("index"))
        self.master_id = _to_int(raw.get("master_id"), 0) or 0
        self.outstation_id = _to_int(raw.get("outstation_id"), 0) or 0
        self.recurrent = _to_bool(raw.get("recurrent"), False)
        self.interval = _to_float(raw.get("interval"), 0.0)
        self.value = "" if raw.get("value") is None else str(raw.get("value"))

        if self.operation_type in (
            "send_binary_command",
            "send_analog_command_float32",
            "send_analog_command_int16",
            "send_analog_command_int32",
            "send_analog_command_double64",
            "poll_group_variation_index",
        ) and self.index is None:
            self.index = 0


class DNP3Connection:
    def __init__(self, ip: str, port: int, master_id: int, outstation_id: int):
        if not DNP3_AVAILABLE:
            raise RuntimeError(
                f"dnp3-python not available: {_IMPORT_ERROR}. "
                "Install with: pip install dnp3-python"
            )
        self.ip = ip
        self.port = port
        self.master_id = master_id
        self.outstation_id = outstation_id
        self.soe_handler = SOEHandler(soehandler_log_level=logging.WARNING)
        stack_config = asiodnp3.MasterStackConfig()
        stack_config.master.responseTimeout = openpal.TimeDuration().Seconds(2)
        none_class = getattr(opendnp3.ClassField, "None")()
        stack_config.master.startupIntegrityClassMask = none_class
        stack_config.master.unsolClassMask = none_class
        stack_config.master.eventScanOnEventsAvailableClassMask = none_class
        stack_config.link.RemoteAddr = self.outstation_id
        stack_config.link.LocalAddr = self.master_id

        self.master = MyMasterNew(
            master_ip="0.0.0.0",
            outstation_ip=self.ip,
            port=self.port,
            master_id=self.master_id,
            outstation_id=self.outstation_id,
            soe_handler=self.soe_handler,
            stack_config=stack_config,
            enable_scans=False,
        )
        self.master.start()


class DNP3Master:
    """
    DNP3 Master Station with CSV-based operation configuration.

    Supports:
    - Polling operations from CSV
    - Control commands from CSV
    - Recurrent operations scheduling
    """

    def __init__(
        self,
        outstation_ip="127.0.0.1",
        outstation_port=20000,
        master_id=2,
        outstation_id=1,
        config_file: Optional[str] = "/app/config/master.yaml",
    ):
        if not DNP3_AVAILABLE:
            raise RuntimeError(
                f"dnp3-python not available: {_IMPORT_ERROR}. "
                "Install with: pip install dnp3-python"
            )
        """
        Initialize DNP3 master station.

        Args:
            outstation_ip: IP address of remote outstation
            outstation_port: TCP port of remote outstation
            master_id: DNP3 address of this master
            outstation_id: DNP3 address of remote outstation
            config_file: Path to CSV configuration file
        """
        self.outstation_ip = outstation_ip
        self.outstation_port = outstation_port
        self.master_id = master_id
        self.outstation_id = outstation_id
        self.operations: List[DNP3Operation] = []
        self.recurrent_threads: List[threading.Thread] = []
        self.running = False
        self.connections: Dict[str, DNP3Connection] = {}
        self.master: Optional[MyMasterNew] = None

        self.stats = {
            "polls_sent": 0,
            "commands_sent": 0,
            "operations_executed": 0,
            "start_time": time.time(),
        }

        if config_file:
            self._load_operations(config_file)

        logger.info("DNP3 Master initialized")
        logger.info(f"  Default outstation: {outstation_ip}:{outstation_port}")
        logger.info(f"  Default Master ID: {master_id}")
        logger.info(f"  Default Outstation ID: {outstation_id}")
        if config_file:
            logger.info(f"  Operations loaded: {len(self.operations)}")

    def _load_operations(self, config_file: str):
        """Load operations from YAML (preferred) or CSV (legacy)."""
        try:
            with open(config_file, "r") as f:
                content = f.read()

            is_csv = config_file.endswith(".csv") or content.lstrip().startswith("timestamp,")
            if is_csv:
                reader = csv.DictReader(content.splitlines())
                self.operations = [DNP3Operation(row) for row in reader]
            else:
                raw = yaml.safe_load(content) or {}
                messages = raw.get("messages") if isinstance(raw, dict) else raw
                if not isinstance(messages, list):
                    raise ValueError("YAML config must define a list of messages")
                self.operations = [DNP3Operation(msg) for msg in messages]

            logger.info(f"Loaded {len(self.operations)} operations from {config_file}")
        except Exception as e:
            logger.error(f"Failed to load operations: {e}")
            raise

    def _connection_key(self, ip: str, port: int, master_id: int, outstation_id: int) -> str:
        return f"{ip}:{port}|{master_id}->{outstation_id}"

    def _resolve_connection_fields(self, operation: Optional[DNP3Operation]) -> Dict[str, int]:
        ip = (operation.ip if operation and operation.ip else self.outstation_ip) or "127.0.0.1"
        port = (operation.port if operation and operation.port else self.outstation_port) or 20000
        master_id = (operation.master_id if operation and operation.master_id else self.master_id) or 2
        outstation_id = (operation.outstation_id if operation and operation.outstation_id else self.outstation_id) or 1
        return {"ip": ip, "port": port, "master_id": master_id, "outstation_id": outstation_id}

    def _ensure_connection(self, operation: Optional[DNP3Operation] = None) -> DNP3Connection:
        fields = self._resolve_connection_fields(operation)
        key = self._connection_key(fields["ip"], fields["port"], fields["master_id"], fields["outstation_id"])
        if key not in self.connections:
            logger.info("Connecting to outstation %s:%d (master=%d, outstation=%d)",
                        fields["ip"], fields["port"], fields["master_id"], fields["outstation_id"])
            self.connections[key] = DNP3Connection(
                ip=fields["ip"],
                port=fields["port"],
                master_id=fields["master_id"],
                outstation_id=fields["outstation_id"],
            )
            time.sleep(1.0)
        if self.master is None:
            self.master = self.connections[key].master
        return self.connections[key]

    def connect(self) -> bool:
        """Establish connections to all outstations referenced in the operations."""
        try:
            if not self.operations:
                self._ensure_connection()
            else:
                for op in self.operations:
                    self._ensure_connection(op)
            if self.master is None and self.connections:
                self.master = next(iter(self.connections.values())).master
            self.stats["start_time"] = time.time()
            return True
        except Exception as e:
            logger.error(f"Connection failed: {e}")
            return False

    def _get_default_master(self) -> Optional[MyMasterNew]:
        if self.master is not None:
            return self.master
        if self.connections:
            self.master = next(iter(self.connections.values())).master
            return self.master
        return self._ensure_connection().master

    def execute_operation(self, operation: DNP3Operation):
        """Execute a single operation based on its type."""
        try:
            op_type = operation.operation_type
            conn = self._ensure_connection(operation)
            master = conn.master

            if op_type == "poll_analog_inputs":
                group = operation.group if operation.group is not None else 30
                variation = operation.variation if operation.variation is not None else 6
                self.poll_analog_inputs(master, group, variation)
            elif op_type == "poll_binary_inputs":
                group = operation.group if operation.group is not None else 1
                variation = operation.variation if operation.variation is not None else 2
                self.poll_binary_inputs(master, group, variation)
            elif op_type == "poll_analog_output_status":
                group = operation.group if operation.group is not None else 40
                variation = operation.variation if operation.variation is not None else 2
                self.poll_analog_output_status(master, group, variation)
            elif op_type == "poll_binary_output_status":
                group = operation.group if operation.group is not None else 10
                variation = operation.variation if operation.variation is not None else 2
                self.poll_binary_output_status(master, group, variation)
            elif op_type == "poll_group_variation":
                if operation.group is None or operation.variation is None:
                    raise ValueError("poll_group_variation requires group and variation")
                master.get_db_by_group_variation(operation.group, operation.variation)
                self.stats["polls_sent"] += 1
            elif op_type == "poll_group_variation_index":
                if operation.group is None or operation.variation is None or operation.index is None:
                    raise ValueError("poll_group_variation_index requires group, variation, index")
                master.get_db_by_group_variation_index(operation.group, operation.variation, operation.index)
                self.stats["polls_sent"] += 1
            elif op_type == "poll_all":
                master.send_scan_all_request()
                self.stats["polls_sent"] += 1
            elif op_type == "send_binary_command":
                if operation.index is None:
                    raise ValueError("send_binary_command requires index")
                value = operation.value == "1" or operation.value.lower() == "true"
                self.send_binary_command(master, operation.index, value)
            elif op_type == "send_analog_command_float32":
                if operation.index is None:
                    raise ValueError("send_analog_command_float32 requires index")
                self.send_analog_command_float32(
                    master, operation.index, float(operation.value)
                )
            elif op_type == "send_analog_command_int16":
                if operation.index is None:
                    raise ValueError("send_analog_command_int16 requires index")
                self.send_analog_command_int16(
                    master, operation.index, int(float(operation.value))
                )
            elif op_type == "send_analog_command_int32":
                if operation.index is None:
                    raise ValueError("send_analog_command_int32 requires index")
                self.send_analog_command_int32(
                    master, operation.index, int(float(operation.value))
                )
            elif op_type == "send_analog_command_double64":
                if operation.index is None:
                    raise ValueError("send_analog_command_double64 requires index")
                self.send_analog_command_double64(
                    master, operation.index, float(operation.value)
                )
            else:
                logger.warning(f"Unknown operation type: {op_type}")
                return

            self.stats["operations_executed"] += 1
            logger.info(f"Executed: {op_type} (timestamp: {operation.timestamp})")

        except Exception as e:
            logger.error(f"Error executing operation {operation.operation_type}: {e}")

    def poll_analog_inputs(self, master: Optional[MyMasterNew] = None, group: int = 30, variation: int = 6):
        """Poll analog inputs."""
        try:
            if master is None:
                master = self._get_default_master()
            logger.debug(f"Polling analog inputs (Group {group}, Var {variation})...")
            result = master.get_db_by_group_variation(group=group, variation=variation)
            self.stats["polls_sent"] += 1
            return result
        except Exception as e:
            logger.error(f"Error polling analog inputs: {e}")
            return None

    def poll_binary_inputs(self, master: Optional[MyMasterNew] = None, group: int = 1, variation: int = 2):
        """Poll binary inputs."""
        try:
            if master is None:
                master = self._get_default_master()
            logger.debug(f"Polling binary inputs (Group {group}, Var {variation})...")
            result = master.get_db_by_group_variation(group=group, variation=variation)
            self.stats["polls_sent"] += 1
            return result
        except Exception as e:
            logger.error(f"Error polling binary inputs: {e}")
            return None

    def poll_analog_output_status(self, master: Optional[MyMasterNew] = None, group: int = 40, variation: int = 2):
        """Poll analog output status."""
        try:
            if master is None:
                master = self._get_default_master()
            logger.debug(
                f"Polling analog output status (Group {group}, Var {variation})..."
            )
            result = master.get_db_by_group_variation(group=group, variation=variation)
            self.stats["polls_sent"] += 1
            return result
        except Exception as e:
            logger.error(f"Error polling analog output status: {e}")
            return None

    def poll_binary_output_status(self, master: Optional[MyMasterNew] = None, group: int = 10, variation: int = 2):
        """Poll binary output status."""
        try:
            if master is None:
                master = self._get_default_master()
            logger.debug(
                f"Polling binary output status (Group {group}, Var {variation})..."
            )
            result = master.get_db_by_group_variation(group=group, variation=variation)
            self.stats["polls_sent"] += 1
            return result
        except Exception as e:
            logger.error(f"Error polling binary output status: {e}")
            return None

    def send_binary_command(self, master: MyMasterNew, index: int, state: bool) -> bool:
        """Send binary control command."""
        try:
            logger.info(
                f"Sending binary command: BO[{index}] = {'ON' if state else 'OFF'}"
            )
            # Use ControlCode to avoid rawCode mutation crashes in bindings.
            code = opendnp3.ControlCode.LATCH_ON if state else opendnp3.ControlCode.LATCH_OFF
            command = opendnp3.ControlRelayOutputBlock(code)
            if not self._send_direct_operate(master, command, index):
                return False
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def _send_direct_operate(self, master: MyMasterNew, command: Any, index: int) -> bool:
        """Send direct-operate command with compatibility fallback for broken bindings."""
        try:
            master.send_direct_operate_command(command, index)
            return True
        except Exception as e:
            err = str(e)
            if "default-holder" in err:
                logger.warning(
                    "  DirectOperate callback binding unavailable (%s). "
                    "Treating command as accepted for compatibility.",
                    err,
                )
                return True
            logger.error(f"  Command failed: {e}")
            return False

    def send_analog_command_float32(self, master: MyMasterNew, index: int, value: float) -> bool:
        """Send analog control command (Float32)."""
        try:
            logger.info(f"Sending analog command (Float32): AO[{index}] = {value}")
            command = opendnp3.AnalogOutputFloat32(float(value))
            if not self._send_direct_operate(master, command, index):
                return False
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def send_analog_command_int16(self, master: MyMasterNew, index: int, value: int) -> bool:
        """Send analog control command (Int16)."""
        try:
            logger.info(f"Sending analog command (Int16): AO[{index}] = {value}")
            command = opendnp3.AnalogOutputInt16(int(value))
            if not self._send_direct_operate(master, command, index):
                return False
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def send_analog_command_int32(self, master: MyMasterNew, index: int, value: int) -> bool:
        """Send analog control command (Int32)."""
        try:
            logger.info(f"Sending analog command (Int32): AO[{index}] = {value}")
            command = opendnp3.AnalogOutputInt32(int(value))
            if not self._send_direct_operate(master, command, index):
                return False
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def send_analog_command_double64(self, master: MyMasterNew, index: int, value: float) -> bool:
        """Send analog control command (Double64)."""
        try:
            logger.info(f"Sending analog command (Double64): AO[{index}] = {value}")
            command = opendnp3.AnalogOutputDouble64(float(value))
            if not self._send_direct_operate(master, command, index):
                return False
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def execute_all_operations(self):
        """Execute all loaded operations in sequence."""
        logger.info("=" * 60)
        logger.info("EXECUTING OPERATIONS FROM CSV")
        logger.info("=" * 60)

        # Sort operations by timestamp
        sorted_ops = sorted(self.operations, key=lambda op: op.timestamp)

        # Separate one-time and recurrent operations
        one_time_ops = [op for op in sorted_ops if not op.recurrent]
        recurrent_ops = [op for op in sorted_ops if op.recurrent]

        # Execute one-time operations
        for operation in one_time_ops:
            self.execute_operation(operation)
            time.sleep(1)  # Small delay between operations

        # Start recurrent operations in background threads
        self.running = True
        for operation in recurrent_ops:
            thread = threading.Thread(
                target=self._recurrent_operation_worker, args=(operation,), daemon=True
            )
            thread.start()
            self.recurrent_threads.append(thread)
            logger.info(
                f"Started recurrent operation: {operation.operation_type} "
                f"(interval: {operation.interval}s)"
            )

    def _recurrent_operation_worker(self, operation: DNP3Operation):
        """Worker for recurrent operations."""
        while self.running:
            self.execute_operation(operation)
            time.sleep(operation.interval)

    def display_current_values(self):
        """Display current values from all polled points."""
        logger.info("\nCURRENT VALUES:")
        point_types = [
            ("Analog", "Analog Inputs"),
            ("Binary", "Binary Inputs"),
            ("AnalogOutputStatus", "Analog Output Status"),
            ("BinaryOutputStatus", "Binary Output Status"),
        ]

        for key, conn in self.connections.items():
            db = conn.soe_handler.db
            logger.info(f"\n  Outstation {key}:")
            for db_key, display_name in point_types:
                values = db.get(db_key, {})
                if values:
                    logger.info(f"    {display_name} ({len(values)} points):")
                    for idx, val in sorted(values.items())[:5]:
                        logger.info(f"      [{idx}] = {val}")
                    if len(values) > 5:
                        logger.info(f"      ... and {len(values)-5} more")

    def display_statistics(self):
        """Display connection and operation statistics."""
        uptime = time.time() - self.stats["start_time"]
        logger.info("=" * 60)
        logger.info("MASTER STATISTICS")
        logger.info("=" * 60)
        logger.info(f"  Connection time: {uptime:.1f}s")
        logger.info(f"  Polls sent: {self.stats['polls_sent']}")
        logger.info(f"  Commands sent: {self.stats['commands_sent']}")
        logger.info(f"  Operations executed: {self.stats['operations_executed']}")
        if uptime > 0:
            logger.info(
                f"  Operation rate: {self.stats['operations_executed']/uptime:.2f} ops/s"
            )
        logger.info("=" * 60)

    def get_all_values(self) -> Dict[str, Any]:
        """Get all current point values."""
        if len(self.connections) == 1:
            return dict(next(iter(self.connections.values())).soe_handler.db)
        return {key: dict(conn.soe_handler.db) for key, conn in self.connections.items()}

    def shutdown(self):
        """Shutdown the master station gracefully."""
        try:
            self.running = False
            for conn in self.connections.values():
                conn.master.shutdown()
            logger.info("Master shutdown complete")
        except Exception as e:
            logger.error(f"Error during shutdown: {e}")


def main():
    """Command-line interface for the DNP3 master."""
    import argparse

    parser = argparse.ArgumentParser(
        description="DNP3 Master Station with CSV Configuration",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument("--ip", default="127.0.0.1", help="Outstation IP address")
    parser.add_argument("--port", type=int, default=20000, help="Outstation TCP port")
    parser.add_argument("--master-id", type=int, default=2, help="DNP3 master address")
    parser.add_argument(
        "--outstation-id", type=int, default=1, help="DNP3 outstation address"
    )
    parser.add_argument(
        "--config",
        type=str,
        default=os.getenv("MASTER_CONFIG", "/app/config/master.yaml"),
        help="Path to YAML/CSV configuration file",
    )
    parser.add_argument(
        "--continuous",
        action="store_true",
        help="Keep running after executing operations",
    )

    args = parser.parse_args()

    master = DNP3Master(
        outstation_ip=args.ip,
        outstation_port=args.port,
        master_id=args.master_id,
        outstation_id=args.outstation_id,
        config_file=args.config,
    )

    if not master.connect():
        logger.error("Failed to connect to outstation")
        sys.exit(1)

    # Initial connection delay
    time.sleep(2)

    # Execute all operations from CSV
    master.execute_all_operations()

    if args.continuous:
        try:
            logger.info("\n" + "=" * 60)
            logger.info("DNP3 MASTER RUNNING (Continuous Mode)")
            logger.info("=" * 60)
            logger.info("Press Ctrl+C to stop")

            counter = 0
            while True:
                time.sleep(10)
                counter += 1

                if counter % 6 == 0:
                    master.display_statistics()
                    master.display_current_values()
        except KeyboardInterrupt:
            logger.info("\nShutting down...")
            master.display_statistics()
            master.shutdown()
    else:
        # Wait a bit for operations to complete
        time.sleep(5)
        master.display_statistics()
        master.display_current_values()
        master.shutdown()


if __name__ == "__main__":
    main()
