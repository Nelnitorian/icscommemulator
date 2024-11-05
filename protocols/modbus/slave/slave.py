import yaml
import sys
from pymodbus.server import StartTcpServer, ServerStop
from pymodbus.device import ModbusDeviceIdentification
from pymodbus.datastore import (
    ModbusSequentialDataBlock,
    ModbusSlaveContext,
    ModbusServerContext,
    ModbusSparseDataBlock,
)
from pathlib import Path


class ModbusSlave:
    def __init__(self, config_file, sync_file="/app/app_running.lock"):
        with open(config_file, "r") as file:
            self.config = yaml.safe_load(file)
        self.sync_file = sync_file

    def create_data_block(self, config):
        if config["values"]:
            if config["type"] == "sequential":
                return ModbusSequentialDataBlock(0, config["values"])
            elif config["type"] == "sparse":
                return ModbusSparseDataBlock(config["values"])
        else:
            return ModbusSparseDataBlock()

    def touch_sync_file(self):
        Path(self.sync_file).touch()

    def start(self):
        try:
            di_block = self.create_data_block(self.config["discrete_inputs"])
            co_block = self.create_data_block(self.config["coils"])
            ir_block = self.create_data_block(self.config["input_registers"])
            hr_block = self.create_data_block(self.config["holding_registers"])

            store = ModbusSlaveContext(
                di=di_block, co=co_block, hr=hr_block, ir=ir_block
            )
            context = ModbusServerContext(slaves=store, single=True)

            identity = None
            identity_info = self.config.get("identity", {})
            if identity_info:
                identity = ModbusDeviceIdentification()
                identity.VendorName = identity_info.get("vendor_name", "Pymodbus")
                identity.ProductCode = identity_info.get("product_code", "PM")
                identity.VendorUrl = identity_info.get(
                    "vendor_url", "http://github.com/riptideio/pymodbus/"
                )
                identity.ProductName = identity_info.get(
                    "product_name", "Pymodbus Server"
                )
                identity.ModelName = identity_info.get("model_name", "Pymodbus Server")
                identity.MajorMinorRevision = identity_info.get(
                    "major_minor_revision", "1.0"
                )

            print(
                f"Modbus TCP Server starting on {self.config['ip']}:{self.config['port']}"
            )
            self.touch_sync_file()
            StartTcpServer(
                context=context,
                identity=identity,
                address=(self.config["ip"], self.config["port"]),
            )

        except Exception as e:
            print(f"Error starting Modbus TCP Server: {e}")
            sys.exit(1)

    def shutdown(self):
        ServerStop()


if __name__ == "__main__":
    print("Starting ModbusSlave...")
    slave = ModbusSlave(config_file="slave.yaml", sync_file="app_running.lock")
    try:
        slave.start()
    finally:
        slave.shutdown()
