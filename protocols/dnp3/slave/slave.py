#!/usr/bin/env python3

"""
DNP3 Outstation Implementation with YAML Configuration
Requires: dnp3-python >= 2.13.6, pyyaml
"""

import logging
import time
import sys
import random
import yaml
from typing import Dict, Any, Optional

try:
    from dnp3_python.dnp3station.outstation import MyOutStationNew
    from pydnp3 import opendnp3
    DNP3_AVAILABLE = True
    _IMPORT_ERROR = None
except ImportError as e:
    MyOutStationNew = None
    opendnp3 = None
    DNP3_AVAILABLE = False
    _IMPORT_ERROR = e

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)

logger = logging.getLogger(__name__)


class DNP3Outstation:
    """DNP3 Outstation with YAML configuration support."""

    def __init__(self, config_file: Optional[str] = "/app/config/slave.yaml"):
        if not DNP3_AVAILABLE:
            raise RuntimeError(
                f"dnp3-python not available: {_IMPORT_ERROR}. "
                "Install with: pip install dnp3-python"
            )
        self.config = (
            self._load_config(config_file) if config_file else self._default_config()
        )
        self.config = self._normalize_config(self.config)
        self.ip = self.config["ip"]
        self.port = self.config["port"]
        self.outstation_id = self.config["outstation_id"]
        self.master_id = self.config["master_id"]
        self.outstation = None
        logger.info("DNP3 Outstation initialized")
        logger.info(f"  Listening on: {self.ip}:{self.port}")

    def _load_config(self, config_file: str) -> Dict[str, Any]:
        try:
            with open(config_file, "r") as f:
                data = yaml.safe_load(f) or {}
                if isinstance(data, dict) and "node" in data and "protocol" in data:
                    return data.get("node", {})
                return data
        except Exception as e:
            logger.error(f"Failed to load config file: {e}")
            raise

    def _default_config(self) -> Dict[str, Any]:
        return {
            "ip": "0.0.0.0",
            "port": 20000,
            "outstation_id": 1,
            "master_id": 2,
            "analog_inputs": {"count": 20, "initial_values": []},
            "binary_inputs": {"count": 20, "initial_values": []},
            "analog_output_status": {"count": 10, "initial_values": []},
            "binary_output_status": {"count": 10, "initial_values": []},
            "simulation": {"enabled": False, "interval": 5000},
        }

    def _normalize_point_config(self, raw: Any, default_count: int) -> Dict[str, Any]:
        if raw is None:
            return {"count": default_count, "initial_values": []}

        if isinstance(raw, dict) and ("count" in raw or "initial_values" in raw):
            count = int(raw.get("count", default_count) or default_count)
            initial_values = raw.get("initial_values", []) or []
            return {"count": count, "initial_values": initial_values}

        if isinstance(raw, dict):
            initial_values = []
            max_index = -1
            for key, value in raw.items():
                try:
                    index = int(key)
                except (TypeError, ValueError):
                    continue
                max_index = max(max_index, index)
                if isinstance(value, dict):
                    val = value.get("value", 0)
                else:
                    val = value
                initial_values.append({"index": index, "value": val})
            if max_index >= 0:
                count = max_index + 1
            else:
                count = default_count
            return {"count": count, "initial_values": initial_values}

        return {"count": default_count, "initial_values": []}

    def _normalize_config(self, config: Dict[str, Any]) -> Dict[str, Any]:
        normalized = dict(config)
        normalized.setdefault("ip", "0.0.0.0")
        normalized["port"] = int(normalized.get("port", 20000))
        normalized["outstation_id"] = int(normalized.get("outstation_id", 1))
        normalized["master_id"] = int(normalized.get("master_id", 2))
        normalized["analog_inputs"] = self._normalize_point_config(
            normalized.get("analog_inputs"), 20
        )
        normalized["binary_inputs"] = self._normalize_point_config(
            normalized.get("binary_inputs"), 20
        )
        normalized["analog_output_status"] = self._normalize_point_config(
            normalized.get("analog_output_status"), 10
        )
        normalized["binary_output_status"] = self._normalize_point_config(
            normalized.get("binary_output_status"), 10
        )
        sim = normalized.get("simulation") or {}
        sim.setdefault("enabled", False)
        sim.setdefault("interval", 5000)
        normalized["simulation"] = sim
        return normalized

    def start(self):
        try:
            logger.info("Starting DNP3 outstation...")
            db_sizes = {
                "analog": int(self.config["analog_inputs"].get("count", 0)),
                "binary": int(self.config["binary_inputs"].get("count", 0)),
                "analog_output_status": int(self.config["analog_output_status"].get("count", 0)),
                "binary_output_status": int(self.config["binary_output_status"].get("count", 0)),
            }
            self.outstation = MyOutStationNew(
                outstation_ip=self.ip,
                port=self.port,
                master_id=self.master_id,
                outstation_id=self.outstation_id,
                db_sizes=db_sizes,
            )
            self.outstation.start()
            self._initialize_points()
            logger.info("DNP3 outstation started successfully")
        except Exception as e:
            logger.error(f"Failed to start outstation: {e}")
            raise

    def _initialize_points(self):
        logger.info("Initializing DNP3 points...")
        # Analog Inputs
        ai_config = self.config.get("analog_inputs", {})
        ai_initial = {
            item["index"]: item["value"] for item in ai_config.get("initial_values", [])
        }
        for i in range(ai_config.get("count", 20)):
            self.update_analog_input(i, ai_initial.get(i, 0.0))

        # Binary Inputs
        bi_config = self.config.get("binary_inputs", {})
        bi_initial = {
            item["index"]: item["value"] for item in bi_config.get("initial_values", [])
        }
        for i in range(bi_config.get("count", 20)):
            self.update_binary_input(i, bi_initial.get(i, False))

        # Analog Output Status
        aos_config = self.config.get("analog_output_status", {})
        aos_initial = {
            item["index"]: item["value"]
            for item in aos_config.get("initial_values", [])
        }
        for i in range(aos_config.get("count", 10)):
            self.update_analog_output_status(i, aos_initial.get(i, 0.0))

        # Binary Output Status
        bos_config = self.config.get("binary_output_status", {})
        bos_initial = {
            item["index"]: item["value"]
            for item in bos_config.get("initial_values", [])
        }
        for i in range(bos_config.get("count", 10)):
            self.update_binary_output_status(i, bos_initial.get(i, False))

    def update_analog_input(self, index: int, value: float) -> bool:
        """Update an analog input point."""
        try:
            measurement = opendnp3.Analog(float(value))
            self.outstation.apply_update(measurement, index)
            logger.debug(f"Updated AI[{index}] = {value:.2f}")
            return True
        except Exception as e:
            logger.error(f"Error updating AI[{index}]: {e}")
            return False

    def update_binary_input(self, index: int, value: bool) -> bool:
        """Update a binary input point."""
        try:
            measurement = opendnp3.Binary(bool(value))
            self.outstation.apply_update(measurement, index)
            logger.debug(f"Updated BI[{index}] = {value}")
            return True
        except Exception as e:
            logger.error(f"Error updating BI[{index}]: {e}")
            return False

    def update_analog_output_status(self, index: int, value: float) -> bool:
        """Update an analog output status point."""
        try:
            measurement = opendnp3.AnalogOutputStatus(float(value))
            self.outstation.apply_update(measurement, index)
            logger.debug(f"Updated AOS[{index}] = {value:.2f}")
            return True
        except Exception as e:
            logger.error(f"Error updating AOS[{index}]: {e}")
            return False

    def update_binary_output_status(self, index: int, value: bool) -> bool:
        """Update a binary output status point."""
        try:
            measurement = opendnp3.BinaryOutputStatus(bool(value))
            self.outstation.apply_update(measurement, index)
            logger.debug(f"Updated BOS[{index}] = {value}")
            return True
        except Exception as e:
            logger.error(f"Error updating BOS[{index}]: {e}")
            return False

    def simulate_scada_data(self):
        """Simulate realistic SCADA data."""
        sim_config = self.config.get("simulation", {})
        variation_range = sim_config.get("analog_variation_range", 2.0)
        ai_config = self.config.get("analog_inputs", {})
        for item in ai_config.get("initial_values", []):
            variation = random.uniform(-variation_range, variation_range)
            self.update_analog_input(item["index"], item["value"] + variation)

    def get_database(self) -> Dict:
        try:
            if hasattr(self.outstation, "db_handler"):
                return self.outstation.db_handler.db
            return {}
        except Exception as e:
            logger.error(f"Error accessing database: {e}")
            return {}

    def display_statistics(self):
        db = self.get_database()
        logger.info("=" * 60)
        logger.info("OUTSTATION STATISTICS")
        for point_type, values in db.items():
            if values:
                logger.info(f"  {point_type:25s}: {len(values)} points")
        logger.info("=" * 60)

    def run(self):
        sim_config = self.config.get("simulation", {})
        enable_simulation = sim_config.get("enabled", False)
        interval_ms = sim_config.get("interval", 5000)
        interval = max(float(interval_ms) / 1000.0, 0.1)
        logger.info("DNP3 OUTSTATION RUNNING")
        try:
            counter = 0
            while True:
                time.sleep(interval)
                counter += 1
                if enable_simulation:
                    self.simulate_scada_data()
                if counter % 12 == 0:
                    self.display_statistics()
        except KeyboardInterrupt:
            self.shutdown()

    def shutdown(self):
        try:
            if self.outstation:
                self.outstation.shutdown()
            logger.info("Outstation shutdown complete")
        except Exception as e:
            logger.error(f"Error during shutdown: {e}")


def main():
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--config",
        type=str,
        default="/app/config/slave.yaml",
        help="Path to YAML configuration file",
    )
    args = parser.parse_args()

    outstation = DNP3Outstation(config_file=args.config)
    outstation.start()
    outstation.run()


if __name__ == "__main__":
    main()
