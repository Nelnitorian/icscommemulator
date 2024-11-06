import csv
import sys
import time
from pymodbus.client import ModbusTcpClient
from pymodbus.constants import DeviceInformation


class ModbusMaster:
    def __init__(self, csv_file="master.csv"):
        self.csv_file = csv_file
        self.responses = []

    def _send_message(
        self, ip, port, function_code, start_address, slave_id, values=None, count=None
    ):
        client = ModbusTcpClient(ip, port=port)
        result = False
        client.connect()
        if function_code in [1, 2, 3, 4]:
            if function_code == 1:
                result = client.read_coils(start_address, count=count, slave=slave_id)
            elif function_code == 2:
                result = client.read_discrete_inputs(
                    start_address, count=count, slave=slave_id
                )
            elif function_code == 3:
                result = client.read_holding_registers(
                    start_address, count=count, slave=slave_id
                )
            elif function_code == 4:
                result = client.read_input_registers(
                    start_address, count=count, slave=slave_id
                )
        elif function_code in [5, 6, 15, 16]:
            if function_code == 5:
                result = client.write_coil(start_address, values[0], slave=slave_id)
            elif function_code == 6:
                result = client.write_register(start_address, values[0], slave=slave_id)
            elif function_code == 15:
                result = client.write_coils(start_address, values, slave=slave_id)
            elif function_code == 16:
                result = client.write_registers(start_address, values, slave=slave_id)
        elif function_code == 43:
            result = client.read_device_information(DeviceInformation.REGULAR, slave_id)
        client.close()
        return result

    def loop(self):
        rows = []
        with open(self.csv_file, "r") as file:
            reader = csv.DictReader(file)
            for row in reader:
                row["timestamp"] = float(row["timestamp"])
                row["recurrent"] = row["recurrent"].lower() == "true"
                if row["recurrent"]:
                    row["original_interval"] = float(row["interval"])
                rows.append(row)

        rows.sort(key=lambda x: x["timestamp"])

        current_time = 0
        while True:
            if not rows:
                break

            row = rows.pop(0)

            delay = max(0, row["timestamp"] - current_time)
            time.sleep(delay)
            current_time = row["timestamp"]

            ip = row["ip"]
            port = int(row["port"])
            function_code = int(row["function_code"])
            start_address = int(row["start_address"])
            slave_id = int(row["slave_id"])
            recurrent = bool(row["recurrent"])

            count = int(row["count"]) if function_code in [1, 2, 3, 4] else None
            values = (
                [int(v) for v in row["values"].split(",")]
                if function_code in [5, 6, 15, 16]
                else None
            )
            print(
                f"Sending msg to {ip}:{port} - {function_code} - {start_address} - {slave_id} - {values} - {count}"
            )

            self.responses.append(
                self._send_message(
                    ip, port, function_code, start_address, slave_id, values, count
                )
            )

            # If the row is recurrent, update its timestamp and reinsert it
            if recurrent:
                row["timestamp"] += row["original_interval"]
                rows.append(row)

            # Sort rows again to maintain order after reinserting recurrent rows
            rows.sort(key=lambda x: x["timestamp"])


if __name__ == "__main__":
    print("Starting ModbusMaster...")
    client = ModbusMaster("master.csv")
    try:
        client.loop()
    except Exception as e:
        # print whole modbus error
        print(f"Error during Modbus message loop: {e}")
        sys.exit(1)
