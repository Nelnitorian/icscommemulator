#!/usr/bin/env python3
"""
DNP3 Master Station Implementation with CSV Configuration

Implements a DNP3 master station supporting operations configured via CSV file.

Requires: dnp3-python >= 2.13.6
"""

import logging
import time
import sys
import threading
import csv
from typing import Dict, Any, List, Optional

try:
    from dnp3_python.dnp3station.master_new import MyMasterNew
    from dnp3_python.dnp3station.station_utils import SOEHandler
    from pydnp3 import opendnp3
except ImportError as e:
    print(f"Error importing dnp3-python: {e}")
    print("Install with: pip install dnp3-python")
    sys.exit(1)

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)

logger = logging.getLogger(__name__)


class DNP3Operation:
    """Represents a single DNP3 operation from CSV configuration."""

    def __init__(self, row: Dict[str, str]):
        self.timestamp = int(row["timestamp"]) if row["timestamp"] else 0
        self.ip = row["ip"]
        self.port = int(row["port"])
        self.operation_type = row["operation_type"]
        self.group = int(row["group"]) if row["group"] else None
        self.variation = int(row["variation"]) if row["variation"] else None
        self.index = int(row["index"]) if row["index"] else None
        self.master_id = int(row["master_id"])
        self.outstation_id = int(row["outstation_id"])
        self.recurrent = row["recurrent"].lower() == "true"
        self.interval = float(row["interval"]) if row["interval"] else 0
        self.value = row["value"]


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
        config_file: Optional[str] = None,
    ):
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
        self.soe_handler = SOEHandler(soehandler_log_level=logging.WARNING)
        self.master = None
        self.operations: List[DNP3Operation] = []
        self.recurrent_threads: List[threading.Thread] = []
        self.running = False

        self.stats = {
            "polls_sent": 0,
            "commands_sent": 0,
            "operations_executed": 0,
            "start_time": time.time(),
        }

        if config_file:
            self._load_operations(config_file)

        logger.info("DNP3 Master initialized")
        logger.info(f"  Target outstation: {outstation_ip}:{outstation_port}")
        logger.info(f"  Master ID: {master_id}")
        logger.info(f"  Outstation ID: {outstation_id}")
        if config_file:
            logger.info(f"  Operations loaded: {len(self.operations)}")

    def _load_operations(self, config_file: str):
        """Load operations from CSV file."""
        try:
            with open(config_file, "r") as f:
                reader = csv.DictReader(f)
                self.operations = [DNP3Operation(row) for row in reader]
            logger.info(f"Loaded {len(self.operations)} operations from {config_file}")
        except Exception as e:
            logger.error(f"Failed to load operations: {e}")
            raise

    def connect(self) -> bool:
        """Establish connection to the outstation."""
        try:
            logger.info("Connecting to outstation...")
            self.master = MyMasterNew(
                master_ip="0.0.0.0",
                outstation_ip=self.outstation_ip,
                port=self.outstation_port,
                master_id=self.master_id,
                outstation_id=self.outstation_id,
                soe_handler=self.soe_handler,
            )
            self.master.start()
            time.sleep(2)
            logger.info("Connection established")
            self.stats["start_time"] = time.time()
            return True
        except Exception as e:
            logger.error(f"Connection failed: {e}")
            return False

    def execute_operation(self, operation: DNP3Operation):
        """Execute a single operation based on its type."""
        try:
            op_type = operation.operation_type

            if op_type == "poll_analog_inputs":
                self.poll_analog_inputs(operation.group, operation.variation)
            elif op_type == "poll_binary_inputs":
                self.poll_binary_inputs(operation.group, operation.variation)
            elif op_type == "poll_analog_output_status":
                self.poll_analog_output_status(operation.group, operation.variation)
            elif op_type == "poll_binary_output_status":
                self.poll_binary_output_status(operation.group, operation.variation)
            elif op_type == "send_binary_command":
                value = operation.value == "1" or operation.value.lower() == "true"
                self.send_binary_command(operation.index, value)
            elif op_type == "send_analog_command_float32":
                self.send_analog_command_float32(
                    operation.index, float(operation.value)
                )
            elif op_type == "send_analog_command_int16":
                self.send_analog_command_int16(
                    operation.index, int(float(operation.value))
                )
            elif op_type == "send_analog_command_int32":
                self.send_analog_command_int32(
                    operation.index, int(float(operation.value))
                )
            elif op_type == "send_analog_command_double64":
                self.send_analog_command_double64(
                    operation.index, float(operation.value)
                )
            else:
                logger.warning(f"Unknown operation type: {op_type}")
                return

            self.stats["operations_executed"] += 1
            logger.info(f"Executed: {op_type} (timestamp: {operation.timestamp})")

        except Exception as e:
            logger.error(f"Error executing operation {operation.operation_type}: {e}")

    def poll_analog_inputs(self, group: int = 30, variation: int = 6):
        """Poll analog inputs."""
        try:
            logger.debug(f"Polling analog inputs (Group {group}, Var {variation})...")
            result = self.master.get_db_by_group_variation(
                group=group, variation=variation
            )
            self.stats["polls_sent"] += 1
            return result
        except Exception as e:
            logger.error(f"Error polling analog inputs: {e}")
            return None

    def poll_binary_inputs(self, group: int = 1, variation: int = 2):
        """Poll binary inputs."""
        try:
            logger.debug(f"Polling binary inputs (Group {group}, Var {variation})...")
            result = self.master.get_db_by_group_variation(
                group=group, variation=variation
            )
            self.stats["polls_sent"] += 1
            return result
        except Exception as e:
            logger.error(f"Error polling binary inputs: {e}")
            return None

    def poll_analog_output_status(self, group: int = 40, variation: int = 2):
        """Poll analog output status."""
        try:
            logger.debug(
                f"Polling analog output status (Group {group}, Var {variation})..."
            )
            result = self.master.get_db_by_group_variation(
                group=group, variation=variation
            )
            self.stats["polls_sent"] += 1
            return result
        except Exception as e:
            logger.error(f"Error polling analog output status: {e}")
            return None

    def poll_binary_output_status(self, group: int = 10, variation: int = 2):
        """Poll binary output status."""
        try:
            logger.debug(
                f"Polling binary output status (Group {group}, Var {variation})..."
            )
            result = self.master.get_db_by_group_variation(
                group=group, variation=variation
            )
            self.stats["polls_sent"] += 1
            return result
        except Exception as e:
            logger.error(f"Error polling binary output status: {e}")
            return None

    def send_binary_command(self, index: int, state: bool) -> bool:
        """Send binary control command."""
        try:
            logger.info(
                f"Sending binary command: BO[{index}] = {'ON' if state else 'OFF'}"
            )
            # CORRECCIÓN: Usar ControlRelayOutputBlock con ControlCode
            if state:
                command = opendnp3.ControlRelayOutputBlock(
                    opendnp3.ControlCode.LATCH_ON
                )
            else:
                command = opendnp3.ControlRelayOutputBlock(
                    opendnp3.ControlCode.LATCH_OFF
                )

            self.master.send_direct_operate_command(command, index)
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def send_analog_command_float32(self, index: int, value: float) -> bool:
        """Send analog control command (Float32)."""
        try:
            logger.info(f"Sending analog command (Float32): AO[{index}] = {value}")
            command = opendnp3.AnalogOutputFloat32(float(value))
            self.master.send_direct_operate_command(command, index)
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def send_analog_command_int16(self, index: int, value: int) -> bool:
        """Send analog control command (Int16)."""
        try:
            logger.info(f"Sending analog command (Int16): AO[{index}] = {value}")
            self.master.send_direct_point_command(
                group=41, variation=2, index=index, val_to_set=int(value)
            )
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def send_analog_command_int32(self, index: int, value: int) -> bool:
        """Send analog control command (Int32)."""
        try:
            logger.info(f"Sending analog command (Int32): AO[{index}] = {value}")
            self.master.send_direct_point_command(
                group=41, variation=1, index=index, val_to_set=int(value)
            )
            self.stats["commands_sent"] += 1
            time.sleep(0.5)
            logger.info("  Command sent")
            return True
        except Exception as e:
            logger.error(f"  Command failed: {e}")
            return False

    def send_analog_command_double64(self, index: int, value: float) -> bool:
        """Send analog control command (Double64)."""
        try:
            logger.info(f"Sending analog command (Double64): AO[{index}] = {value}")
            self.master.send_direct_point_command(
                group=41, variation=4, index=index, val_to_set=float(value)
            )
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
        db = self.soe_handler.db
        logger.info("\nCURRENT VALUES:")

        point_types = [
            ("Analog", "Analog Inputs"),
            ("Binary", "Binary Inputs"),
            ("AnalogOutputStatus", "Analog Output Status"),
            ("BinaryOutputStatus", "Binary Output Status"),
        ]

        for db_key, display_name in point_types:
            values = db.get(db_key, {})
            if values:
                logger.info(f"\n  {display_name} ({len(values)} points):")
                for idx, val in sorted(values.items())[:5]:
                    logger.info(f"    [{idx}] = {val}")
                if len(values) > 5:
                    logger.info(f"    ... and {len(values)-5} more")

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
        return dict(self.soe_handler.db)

    def shutdown(self):
        """Shutdown the master station gracefully."""
        try:
            self.running = False
            if self.master:
                self.master.shutdown()
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
        "--config", type=str, required=True, help="Path to CSV configuration file"
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
