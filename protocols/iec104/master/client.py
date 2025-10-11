#!/usr/bin/env python3

"""
Cliente IEC 104 - SCADA
Implementación de cliente IEC 60870-5-104 para sistemas de control y monitoreo.
"""

import c104
import time
import csv
import threading
from dataclasses import dataclass
from typing import List, Optional, Union, Set

COMMAND_IOAS: Set[int] = {1001, 1002, 1003, 1004, 10001, 10002, 10003, 10004}


@dataclass
class Command:
    """Representa un comando IEC 104 a ejecutar"""

    timestamp: int
    ip: str
    port: int
    type_id: int
    common_address: int
    recurrent: bool
    interval: int
    ioa: int
    cot: int
    value: Optional[str]

    @classmethod
    def from_csv_row(cls, row: dict):
        return cls(
            timestamp=int(row["timestamp"]),
            ip=row["ip"],
            port=int(row["port"]),
            type_id=int(row["type_id"]),
            common_address=int(row["common_address"]),
            recurrent=row["recurrent"].lower() == "true",
            interval=int(row["interval"]),
            ioa=int(row["ioa"]),
            cot=int(row["cot"]),
            value=row["value"] if row["value"] else None,
        )


class IEC104Client:
    def __init__(self, server_ip="127.0.0.1", server_port=2404, common_address=1):
        self.client = c104.Client(tick_rate_ms=100, command_timeout_ms=10000)
        self.common_address = common_address

        connection = self.client.add_connection(
            ip=server_ip, port=server_port, init=c104.Init.INTERROGATION
        )

        if connection is None:
            raise RuntimeError("No se pudo crear la conexión al servidor")

        self.connection: c104.Connection = connection

        station = self.connection.add_station(common_address=common_address)
        if station is None:
            raise RuntimeError("No se pudo crear la estación")

        self.station: c104.Station = station
        self.received_measurements: List[dict] = []
        self.connection_state = {"connected": False, "initialized": False}
        self.recurrent_commands: List[Command] = []
        self.recurrent_threads: List[threading.Thread] = []
        self.running = False
        self._registered_points = set()
        self._command_points = set()
        self._measurements_lock = threading.Lock()

        self._setup_callbacks()

    def _extract_value(
        self,
        raw_value: Union[
            bool,
            int,
            float,
            c104.Int16,
            c104.NormalizedFloat,
            c104.Int7,
            c104.Byte32,
            c104.Double,
            None,
        ],
    ) -> Union[bool, int, float, None]:
        if raw_value is None:
            return None

        if isinstance(raw_value, (bool, int, float)):
            return raw_value

        if isinstance(raw_value, (c104.Int16, c104.Int7, c104.Byte32)):
            return int(raw_value)
        elif isinstance(raw_value, (c104.NormalizedFloat, c104.Double)):
            return float(raw_value)

        try:
            return float(raw_value)
        except:
            return raw_value

    def _register_point_callback(self, point: c104.Point):
        if point.io_address in self._registered_points:
            return

        def on_point_receive(
            point: c104.Point,
            previous_info: c104.Information,
            message: c104.IncomingMessage,
        ) -> c104.ResponseState:
            actual_value = self._extract_value(point.value)

            with self._measurements_lock:
                self.received_measurements.append(
                    {
                        "io_address": point.io_address,
                        "type": point.type,
                        "value": actual_value,
                        "quality": point.quality,
                        "timestamp": time.time(),
                        "source": "on_receive_callback",
                    }
                )

            return c104.ResponseState.SUCCESS

        point.on_receive(callable=on_point_receive)
        self._registered_points.add(point.io_address)

    def _setup_callbacks(self):
        def on_connection_state(
            connection: c104.Connection, state: c104.ConnectionState
        ) -> None:
            self.connection_state["connected"] = state == c104.ConnectionState.OPEN
            if state == c104.ConnectionState.OPEN:
                self.connection_state["initialized"] = True

        def on_new_station(
            client: c104.Client, connection: c104.Connection, common_address: int
        ) -> None:
            pass

        def on_new_point(
            client: c104.Client,
            station: c104.Station,
            io_address: int,
            point_type: c104.Type,
        ) -> None:
            if io_address in COMMAND_IOAS:
                return

            point = station.add_point(io_address=io_address, type=point_type)

            if point:
                actual_value = self._extract_value(point.value)

                with self._measurements_lock:
                    self.received_measurements.append(
                        {
                            "io_address": io_address,
                            "type": point_type,
                            "value": actual_value,
                            "quality": point.quality,
                            "timestamp": time.time(),
                            "source": "on_new_point_callback",
                        }
                    )

                self._register_point_callback(point)

        def on_receive_raw(connection: c104.Connection, data: bytes) -> None:
            pass

        def on_send_raw(connection: c104.Connection, data: bytes) -> None:
            pass

        self.connection.on_state_change(callable=on_connection_state)
        self.connection.on_receive_raw(callable=on_receive_raw)
        self.connection.on_send_raw(callable=on_send_raw)
        self.client.on_new_station(callable=on_new_station)
        self.client.on_new_point(callable=on_new_point)

    def start(self):
        self.client.start()
        self.running = True

    def stop(self):
        self.running = False

        for thread in self.recurrent_threads:
            thread.join(timeout=1)

        self.client.stop()

    def wait_for_connection(self, timeout=10):
        start_time = time.time()

        while time.time() - start_time < timeout:
            if (
                self.connection_state["connected"]
                and self.connection_state["initialized"]
            ):
                time.sleep(1.5)
                return True
            time.sleep(0.5)

        return False

    def execute_command(self, cmd: Command):
        try:
            type_map = {
                45: c104.Type.C_SC_NA_1,
                46: c104.Type.C_DC_NA_1,
                48: c104.Type.C_SE_NB_1,
                49: c104.Type.C_SE_NC_1,
                100: c104.Type.C_IC_NA_1,
                102: c104.Type.C_RD_NA_1,
            }

            cot_map = {
                5: c104.Cot.REQUEST,
                6: c104.Cot.ACTIVATION,
                7: c104.Cot.ACTIVATION_CON,
                8: c104.Cot.DEACTIVATION,
            }

            point_type = type_map.get(cmd.type_id)
            cot = cot_map.get(cmd.cot, c104.Cot.ACTIVATION)

            if not point_type:
                return False

            if cmd.type_id == 100:
                success = self.connection.interrogation(
                    common_address=self.common_address,
                    cause=cot,
                    qualifier=c104.Qoi.STATION,
                )

                if success:
                    time.sleep(1.5)

                return success

            point = None

            if cmd.ioa in self._command_points:
                point = self.station.get_point(io_address=cmd.ioa)

            if not point:
                point = self.station.add_point(io_address=cmd.ioa, type=point_type)
                if point:
                    self._command_points.add(cmd.ioa)

            if not point:
                return False

            if cmd.value:
                if cmd.type_id == 45:
                    point.value = bool(int(cmd.value))
                elif cmd.type_id == 46:
                    point.value = int(cmd.value)
                elif cmd.type_id == 48:
                    point.value = int(cmd.value)
                elif cmd.type_id == 49:
                    point.value = float(cmd.value)

            return point.transmit(cause=cot)

        except Exception:
            return False

    def get_latest_measurements(self):
        with self._measurements_lock:
            return self.received_measurements.copy()

    def is_connected(self):
        return self.connection_state["connected"]


if __name__ == "__main__":
    import sys

    client = IEC104Client(server_ip="127.0.0.1", server_port=2404, common_address=1)

    try:
        client.start()

        if client.wait_for_connection(timeout=10):
            print("\nCliente conectado y operativo. Presiona Ctrl+C para detener.\n")
            while True:
                time.sleep(5)
        else:
            print("No se pudo establecer la conexión")

    except KeyboardInterrupt:
        print("\n\nDeteniendo cliente...")
    finally:
        client.stop()
