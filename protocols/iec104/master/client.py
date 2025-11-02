"""
Cliente IEC 60870-5-104 (SCADA Master)
ImplementaciÃ³n completa con interrogaciÃ³n, comandos, sincronizaciÃ³n de reloj
y procesamiento de eventos espontÃ¡neos con timestamps precisos
"""

import c104
import csv
import logging
import datetime
import time
import threading
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass
from enum import IntEnum


logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s.%(msecs)03d [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


class CommandType(IntEnum):
    """Tipos de comandos soportados"""

    INTERROGATION = 100  # C_IC_NA_1
    COUNTER_INTERROGATION = 101  # C_CI_NA_1
    READ = 102  # C_RD_NA_1
    CLOCK_SYNC = 103  # C_CS_NA_1
    TEST = 104  # C_TS_NA_1
    SINGLE_COMMAND = 45  # C_SC_NA_1
    SINGLE_COMMAND_TIME = 58  # C_SC_TA_1
    SETPOINT_NORMALIZED = 48  # C_SE_NA_1
    SETPOINT_NORMALIZED_TIME = 61  # C_SE_TA_1
    SETPOINT_SCALED = 49  # C_SE_NB_1
    SETPOINT_SCALED_TIME = 62  # C_SE_TB_1
    SETPOINT_SHORT = 50  # C_SE_NC_1
    SETPOINT_SHORT_TIME = 63  # C_SE_TC_1


@dataclass
class CommandSchedule:
    """ProgramaciÃ³n de comando"""

    timestamp: float
    ip: str
    port: int
    type_id: int
    common_address: int
    recurrent: bool
    interval: float
    ioa: int
    cot: int
    value: Optional[str]
    executed: bool = False
    last_execution: float = 0.0


class IEC104Client:
    """
    Cliente IEC 60870-5-104 con capacidades completas de interrogaciÃ³n,
    comandos, sincronizaciÃ³n y procesamiento de eventos con timestamps
    """

    def __init__(self, tick_rate_ms: int = 100, command_timeout_ms: int = 10000):
        """
        Inicializa el cliente SCADA

        Args:
            tick_rate_ms: Intervalo de actualizaciÃ³n del thread del cliente
            command_timeout_ms: Timeout para respuestas de comandos
        """
        self.client = c104.Client(
            tick_rate_ms=tick_rate_ms, command_timeout_ms=command_timeout_ms
        )
        self.connections: Dict[Tuple[str, int], c104.Connection] = {}
        self.stations: Dict[Tuple[str, int, int], c104.Station] = {}
        self.points: Dict[Tuple[str, int, int, int], c104.Point] = {}
        self.received_data: List[dict] = []
        self.data_lock = threading.Lock()
        self.running = False

        logger.info(
            f"IEC104 Client initialized (tick_rate={tick_rate_ms}ms, "
            f"command_timeout={command_timeout_ms}ms)"
        )

    def read_command(
        self,
        ip: str,
        port: int,
        common_address: int,
        io_address: int,
        wait: bool = True,
    ) -> bool:
        """
        Envía comando de lectura (C_RD_NA_1) para solicitar el valor actual de un punto

        Args:
            ip: IP del servidor RTU
            port: Puerto del servidor
            common_address: CA de la estación
            io_address: Dirección del objeto de información a leer
            wait: Esperar respuesta

        Returns:
            True si el comando fue enviado exitosamente
        """
        conn_key = (ip, port)
        if conn_key not in self.connections:
            logger.error(f"Connection {ip}:{port} not found")
            return False

        connection = self.connections[conn_key]

        # Obtener o crear la estación
        station = connection.get_station(common_address=common_address)
        if not station:
            logger.warning(f"Station CA={common_address} not found, creating it")
            station = connection.add_station(common_address=common_address)
            if not station:
                logger.error(f"Failed to create station CA={common_address}")
                return False

        # Obtener o crear el punto
        point = station.get_point(io_address=io_address)
        if not point:
            logger.warning(
                f"Point IOA={io_address} not found in station CA={common_address}, "
                f"creating temporary point as M_SP_NA_1"
            )
            # Crear un punto temporal - el tipo será actualizado cuando llegue la respuesta
            point = station.add_point(
                io_address=io_address, type=c104.Type.M_SP_NA_1  # Tipo por defecto
            )
            if not point:
                logger.error(f"Failed to create point IOA={io_address}")
                return False

        logger.info(
            f"Sending read command (C_RD_NA_1) to {ip}:{port} CA={common_address} IOA={io_address}"
        )

        try:
            # El método read() está en el objeto Point, no en Connection
            success = point.read()

            if success:
                logger.info(f"Read command sent successfully for IOA={io_address}")
            else:
                logger.error(f"Failed to send read command for IOA={io_address}")

            return success

        except Exception as e:
            logger.error(f"Exception sending read command: {e}")
            return False

    def add_connection(
        self, ip: str, port: int = 2404, init_mode: c104.Init = c104.Init.INTERROGATION
    ) -> c104.Connection:
        """
        AÃ±ade conexiÃ³n a RTU
        """
        key = (ip, port)
        if key in self.connections:
            logger.warning(f"Connection {ip}:{port} already exists, returning existing")
            return self.connections[key]

        connection = self.client.add_connection(ip=ip, port=port, init=init_mode)
        if not connection:
            logger.error(f"Failed to create connection to {ip}:{port}")
            return None

        self.connections[key] = connection

        # Configurar callbacks
        connection.on_state_change(callable=self._on_connection_state_change)
        connection.on_receive_raw(callable=self._on_receive_raw)
        connection.on_send_raw(callable=self._on_send_raw)
        connection.on_unexpected_message(callable=self._on_unexpected_message)

        logger.info(f"Connection added: {ip}:{port} (init={init_mode})")

        # IMPORTANTE: Conectar explÃ­citamente
        connection.connect()

        return connection

    def add_station(self, ip: str, port: int, common_address: int) -> c104.Station:
        """
        AÃ±ade estaciÃ³n a una conexiÃ³n
        """
        conn_key = (ip, port)
        if conn_key not in self.connections:
            logger.error(f"Connection {ip}:{port} not found")
            return None

        # Verificar si la estaciÃ³n ya existe
        station_key = (ip, port, common_address)
        if station_key in self.stations:
            logger.info(
                f"Station {ip}:{port} CA={common_address} already exists, returning existing"
            )
            return self.stations[station_key]

        connection = self.connections[conn_key]

        # IMPORTANTE: Usar get_station primero para verificar si ya existe en la conexiÃ³n
        station = connection.get_station(common_address=common_address)

        if not station:
            # Solo crear si no existe
            station = connection.add_station(common_address=common_address)

        if not station:
            logger.error(f"Failed to create station CA={common_address}")
            return None

        self.stations[station_key] = station

        logger.info(f"Station added: {ip}:{port} CA={common_address}")
        return station

    def _on_connection_state_change(
        self, connection: c104.Connection, state: c104.ConnectionState
    ) -> None:
        """Callback: cambio de estado de conexiÃ³n"""
        logger.info(
            f"Connection {connection.ip}:{connection.port} state changed to {state}"
        )

    def _on_receive_raw(self, connection: c104.Connection, data: bytes) -> None:
        """Callback: mensaje entrante (raw)"""
        logger.debug(f"RX [{connection.ip}:{connection.port}]: {data.hex()}")

    def _on_send_raw(self, connection: c104.Connection, data: bytes) -> None:
        """Callback: mensaje saliente (raw)"""
        logger.debug(f"TX [{connection.ip}:{connection.port}]: {data.hex()}")

    def _on_unexpected_message(
        self,
        connection: c104.Connection,
        message: c104.IncomingMessage,
        cause: c104.Umc,
    ) -> None:
        """Callback: mensaje inesperado"""
        logger.warning(
            f"Unexpected message from {connection.ip}:{connection.port}: "
            f"{cause}, CA={message.common_address}, IOA={message.io_address}"
        )

    def _on_point_receive(
        self,
        point: c104.Point,
        previous_info: c104.Information,
        message: c104.IncomingMessage,
    ) -> c104.ResponseState:
        """Callback: recepciÃ³n de valor de punto"""
        # Asegurar timestamps con timezone
        if point.recorded_at:
            timestamp_dt = point.recorded_at
            if timestamp_dt.tzinfo is None:
                timestamp_dt = timestamp_dt.replace(tzinfo=datetime.timezone.utc)
            timestamp_str = timestamp_dt.isoformat()
        else:
            timestamp_str = datetime.datetime.now(datetime.timezone.utc).isoformat()

        # processed_at siempre tiene timezone
        processed_at_str = point.processed_at.isoformat()

        # Almacenar datos recibidos
        data_entry = {
            "timestamp": timestamp_str,
            "processed_at": processed_at_str,
            "io_address": point.io_address,
            "type": str(point.type),
            "value": str(point.value),
            "quality": str(point.quality) if point.quality else "N/A",
            "cot": str(message.cot),
            "common_address": message.common_address,
            "test": message.is_test,
            "negative": message.is_negative,
        }

        with self.data_lock:
            self.received_data.append(data_entry)

        # Log basado en COT
        if message.cot == c104.Cot.SPONTANEOUS:
            logger.info(
                f"SPONTANEOUS: CA={message.common_address}, IOA={point.io_address}, "
                f"Type={point.type}, Value={point.value}, Quality={point.quality}, "
                f"Time={timestamp_str}"
            )
        elif message.cot in [c104.Cot.INTERROGATED_BY_STATION] + [
            getattr(c104.Cot, f"INTERROGATED_BY_GROUP_{i}", None)
            for i in range(1, 17)
            if hasattr(c104.Cot, f"INTERROGATED_BY_GROUP_{i}")
        ]:
            logger.info(
                f"INTERROGATION: CA={message.common_address}, IOA={point.io_address}, "
                f"Type={point.type}, Value={point.value}, Time={timestamp_str}"
            )
        elif message.cot == c104.Cot.ACTIVATION_CON:
            logger.info(
                f"COMMAND ACK: CA={message.common_address}, IOA={point.io_address}, "
                f"Type={point.type}, Value={point.value}"
            )

        return c104.ResponseState.NONE

    def interrogation(
        self,
        ip: str,
        port: int,
        common_address: int = 65535,
        qualifier: c104.Qoi = c104.Qoi.STATION,
        wait: bool = True,
    ) -> bool:
        """
        EnvÃ­a comando de interrogaciÃ³n general

        Args:
            ip: IP del servidor RTU
            port: Puerto del servidor
            common_address: CA de la estaciÃ³n (65535 = broadcast)
            qualifier: Calificador de interrogaciÃ³n
            wait: Esperar respuesta

        Returns:
            True si el comando fue enviado exitosamente
        """
        conn_key = (ip, port)
        if conn_key not in self.connections:
            logger.error(f"Connection {ip}:{port} not found")
            return False

        connection = self.connections[conn_key]

        logger.info(
            f"Sending interrogation to {ip}:{port} CA={common_address}, QOI={qualifier}"
        )
        success = connection.interrogation(
            common_address=common_address,
            cause=c104.Cot.ACTIVATION,
            qualifier=qualifier,
            wait_for_response=wait,
        )

        if success:
            logger.info("Interrogation command sent successfully")
        else:
            logger.error("Failed to send interrogation command")

        return success

    def counter_interrogation(
        self,
        ip: str,
        port: int,
        common_address: int = 65535,
        qualifier: c104.Rqt = c104.Rqt.GENERAL,
        freeze: c104.Frz = c104.Frz.READ,
        wait: bool = True,
    ) -> bool:
        """
        EnvÃ­a comando de interrogaciÃ³n de contadores

        Args:
            ip: IP del servidor RTU
            port: Puerto del servidor
            common_address: CA de la estaciÃ³n
            qualifier: Calificador de interrogaciÃ³n de contadores
            freeze: Comportamiento de congelaciÃ³n
            wait: Esperar respuesta

        Returns:
            True si el comando fue enviado exitosamente
        """
        conn_key = (ip, port)
        if conn_key not in self.connections:
            logger.error(f"Connection {ip}:{port} not found")
            return False

        connection = self.connections[conn_key]

        logger.info(f"Sending counter interrogation to {ip}:{port} CA={common_address}")
        success = connection.counter_interrogation(
            common_address=common_address,
            cause=c104.Cot.ACTIVATION,
            qualifier=qualifier,
            freeze=freeze,
            wait_for_response=wait,
        )

        if success:
            logger.info("Counter interrogation command sent successfully")
        else:
            logger.error("Failed to send counter interrogation command")

        return success

    def clock_sync(
        self, ip: str, port: int, common_address: int = 65535, wait: bool = True
    ) -> bool:
        """
        EnvÃ­a comando de sincronizaciÃ³n de reloj (C_CS_NA_1)

        Args:
            ip: IP del servidor RTU
            port: Puerto del servidor
            common_address: CA de la estaciÃ³n
            wait: Esperar respuesta

        Returns:
            True si el comando fue enviado exitosamente
        """
        conn_key = (ip, port)
        if conn_key not in self.connections:
            logger.error(f"Connection {ip}:{port} not found")
            return False

        connection = self.connections[conn_key]

        current_time = datetime.datetime.now(datetime.timezone.utc)
        logger.info(
            f"Sending clock sync to {ip}:{port} CA={common_address}, Time={current_time.isoformat()}"
        )

        success = connection.clock_sync(
            common_address=common_address, wait_for_response=wait
        )

        if success:
            logger.info("Clock sync command sent successfully")
        else:
            logger.error("Failed to send clock sync command")

        return success

    def add_point(
        self,
        ip: str,
        port: int,
        common_address: int,
        io_address: int,
        point_type: c104.Type,
    ) -> c104.Point:
        """
        AÃ±ade punto a una estaciÃ³n
        """
        station_key = (ip, port, common_address)
        if station_key not in self.stations:
            logger.error(f"Station {ip}:{port} CA={common_address} not found")
            return None

        # Verificar si el punto ya existe
        point_key = (ip, port, common_address, io_address)
        if point_key in self.points:
            logger.info(
                f"Point {ip}:{port} CA={common_address} IOA={io_address} already exists"
            )
            return self.points[point_key]

        station = self.stations[station_key]

        # IMPORTANTE: Usar get_point primero para verificar si ya existe en la estaciÃ³n
        point = station.get_point(io_address=io_address)

        if not point:
            # Solo crear si no existe
            point = station.add_point(io_address=io_address, type=point_type)

        if not point:
            logger.error(f"Failed to create point IOA={io_address}")
            return None

        # Registrar callback para recepciÃ³n de datos
        point.on_receive(callable=self._on_point_receive)

        self.points[point_key] = point

        logger.info(
            f"Point added: {ip}:{port} CA={common_address} IOA={io_address} Type={point_type}"
        )
        return point

    def send_command(
        self,
        ip: str,
        port: int,
        common_address: int,
        io_address: int,
        value: any,
        command_type: c104.Type = c104.Type.C_SC_NA_1,
    ) -> bool:
        """
        EnvÃ­a comando de control
        """
        point_key = (ip, port, common_address, io_address)
        if point_key not in self.points:
            # Crear punto si no existe
            self.add_point(ip, port, common_address, io_address, command_type)

        point = self.points.get(point_key)
        if not point:
            logger.error(f"Failed to get/create point IOA={io_address}")
            return False

        # Asignar valor segÃºn tipo
        try:
            if command_type in [c104.Type.C_SC_NA_1]:
                point.value = bool(value)
            elif command_type in [c104.Type.C_SE_NA_1]:
                point.value = c104.NormalizedFloat(float(value))
            elif command_type in [c104.Type.C_SE_NB_1]:
                point.value = c104.Int16(int(value))
            elif command_type in [c104.Type.C_SE_NC_1]:
                point.value = float(value)
            else:
                point.value = value
        except Exception as e:
            logger.error(f"Failed to set point value: {e}")
            return False

        logger.info(
            f"Sending command to {ip}:{port} CA={common_address} IOA={io_address}, "
            f"Type={command_type}, Value={value}"
        )

        success = point.transmit(cause=c104.Cot.ACTIVATION)

        if success:
            logger.info("Command sent successfully")
        else:
            logger.error("Failed to send command")

        return success

    def load_schedule_from_csv(self, csv_path: str) -> List[CommandSchedule]:
        """
        Carga programaciÃ³n de comandos desde CSV

        Formato CSV:
        timestamp,ip,port,typeid,common_address,recurrent,interval,ioa,cot,value

        Args:
            csv_path: Ruta al archivo CSV

        Returns:
            Lista de CommandSchedule
        """
        schedules = []

        try:
            with open(csv_path, "r") as f:
                reader = csv.DictReader(f)
                for row in reader:
                    schedule = CommandSchedule(
                        timestamp=float(row["timestamp"]),
                        ip=row["ip"],
                        port=int(row["port"]),
                        type_id=int(row["typeid"]),
                        common_address=int(row["common_address"]),
                        recurrent=row["recurrent"].lower() == "true",
                        interval=float(row["interval"]),
                        ioa=int(row["ioa"]),
                        cot=int(row["cot"]),
                        value=row["value"] if row["value"] else None,
                    )
                    schedules.append(schedule)

            logger.info(f"Loaded {len(schedules)} scheduled commands from {csv_path}")

        except Exception as e:
            logger.error(f"Failed to load schedule: {e}", exc_info=True)

        return schedules

    def execute_schedule(self, schedules: List[CommandSchedule]):
        """
        Ejecuta programaciÃ³n de comandos

        Args:
            schedules: Lista de CommandSchedule
        """
        start_time = time.time()

        while self.running:
            current_time = time.time()
            elapsed = current_time - start_time

            for schedule in schedules:
                # Verificar si debe ejecutarse
                should_execute = False

                if not schedule.executed and elapsed >= schedule.timestamp:
                    should_execute = True
                elif (
                    schedule.recurrent
                    and (elapsed - schedule.last_execution) >= schedule.interval
                ):
                    should_execute = True

                if not should_execute:
                    continue

                # Ejecutar comando segÃºn type_id
                try:
                    if schedule.type_id == CommandType.INTERROGATION:
                        self.interrogation(
                            schedule.ip, schedule.port, schedule.common_address
                        )

                    elif schedule.type_id == CommandType.COUNTER_INTERROGATION:
                        self.counter_interrogation(
                            schedule.ip, schedule.port, schedule.common_address
                        )

                    elif schedule.type_id == CommandType.CLOCK_SYNC:
                        self.clock_sync(
                            schedule.ip, schedule.port, schedule.common_address
                        )

                    elif schedule.type_id in [
                        CommandType.SINGLE_COMMAND,
                        CommandType.SINGLE_COMMAND_TIME,
                    ]:
                        cmd_type = (
                            c104.Type.C_SC_TA_1
                            if schedule.type_id == CommandType.SINGLE_COMMAND_TIME
                            else c104.Type.C_SC_NA_1
                        )
                        self.send_command(
                            schedule.ip,
                            schedule.port,
                            schedule.common_address,
                            schedule.ioa,
                            schedule.value,
                            cmd_type,
                        )

                    elif schedule.type_id in [CommandType.SETPOINT_NORMALIZED_TIME]:
                        self.send_command(
                            schedule.ip,
                            schedule.port,
                            schedule.common_address,
                            schedule.ioa,
                            schedule.value,
                            c104.Type.C_SE_TA_1,
                        )

                    elif schedule.type_id in [CommandType.SETPOINT_SCALED_TIME]:
                        self.send_command(
                            schedule.ip,
                            schedule.port,
                            schedule.common_address,
                            schedule.ioa,
                            schedule.value,
                            c104.Type.C_SE_TB_1,
                        )

                    elif schedule.type_id in [CommandType.SETPOINT_SHORT_TIME]:
                        self.send_command(
                            schedule.ip,
                            schedule.port,
                            schedule.common_address,
                            schedule.ioa,
                            schedule.value,
                            c104.Type.C_SE_TC_1,
                        )

                    elif schedule.type_id == CommandType.READ:
                        self.read_command(
                            schedule.ip,
                            schedule.port,
                            schedule.common_address,
                            schedule.ioa,
                        )

                    schedule.executed = True
                    schedule.last_execution = elapsed

                except Exception as e:
                    logger.error(f"Failed to execute schedule: {e}", exc_info=True)

            time.sleep(0.1)

    def save_data_to_csv(self, csv_path: str):
        """
        Guarda datos recibidos a CSV

        Args:
            csv_path: Ruta al archivo CSV de salida
        """
        with self.data_lock:
            if not self.received_data:
                logger.warning("No data to save")
                return

            try:
                with open(csv_path, "w", newline="") as f:
                    fieldnames = [
                        "timestamp",
                        "processed_at",
                        "io_address",
                        "type",
                        "value",
                        "quality",
                        "cot",
                        "common_address",
                        "test",
                        "negative",
                    ]
                    writer = csv.DictWriter(f, fieldnames=fieldnames)
                    writer.writeheader()
                    writer.writerows(self.received_data)

                logger.info(
                    f"Saved {len(self.received_data)} data entries to {csv_path}"
                )

            except Exception as e:
                logger.error(f"Failed to save data: {e}", exc_info=True)

    def start(self):
        """Inicia el cliente"""
        if self.running:
            logger.warning("Client already running")
            return

        logger.info("Starting IEC104 Client...")
        self.client.start()
        self.running = True
        logger.info("Client started successfully")

    def stop(self):
        """Detiene el cliente"""
        if not self.running:
            return

        logger.info("Stopping IEC104 Client...")
        self.running = False
        self.client.stop()
        logger.info("Client stopped")


def main():
    """FunciÃ³n principal"""
    import argparse

    parser = argparse.ArgumentParser(description="IEC 60870-5-104 Client (SCADA)")
    parser.add_argument("--ip", type=str, default="127.0.0.1", help="RTU IP address")
    parser.add_argument("--port", type=int, default=2404, help="RTU port")
    parser.add_argument("--ca", type=int, default=1, help="Common address")
    parser.add_argument("--schedule", type=str, help="CSV file with command schedule")
    parser.add_argument(
        "--output",
        type=str,
        default="client_data.csv",
        help="Output CSV file for received data",
    )
    parser.add_argument("--debug", action="store_true", help="Enable debug logging")

    args = parser.parse_args()

    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)
        c104.set_debug_mode(
            c104.Debug.Client | c104.Debug.Connection | c104.Debug.Point
        )

    client = IEC104Client()

    # AÃ±adir conexiÃ³n y estaciÃ³n
    client.add_connection(args.ip, args.port, c104.Init.INTERROGATION)
    client.add_station(args.ip, args.port, args.ca)

    # AÃ±adir puntos tÃ­picos
    client.add_point(args.ip, args.port, args.ca, 1001, c104.Type.M_SP_TA_1)
    client.add_point(args.ip, args.port, args.ca, 5001, c104.Type.M_ME_TE_1)
    client.add_point(args.ip, args.port, args.ca, 6001, c104.Type.M_ME_TF_1)

    client.start()

    try:
        if args.schedule:
            schedules = client.load_schedule_from_csv(args.schedule)
            client.execute_schedule(schedules)
        else:
            # Modo interactivo simple
            time.sleep(2)
            client.interrogation(args.ip, args.port, args.ca)

            time.sleep(30)

    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received")

    finally:
        client.save_data_to_csv(args.output)
        client.stop()


if __name__ == "__main__":
    main()
