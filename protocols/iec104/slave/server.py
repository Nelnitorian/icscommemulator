"""
Servidor IEC 60870-5-104 (RTU/Outstation)
ImplementaciÃ³n completa con soporte de timestamps, eventos espontÃ¡neos,
comandos y sincronizaciÃ³n de reloj segÃºn estÃ¡ndar IEC 60870-5-104
"""

import c104
import yaml
import logging
import datetime
import threading
import time
import random
from pathlib import Path
from typing import Dict, List, Optional

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s.%(msecs)03d [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


class IEC104Server:
    """
    Servidor IEC 60870-5-104 con gestiÃ³n completa de timestamps,
    eventos espontÃ¡neos, comandos y sincronizaciÃ³n
    """

    def __init__(self, config_path: str = "server_config.yaml"):
        """
        Inicializa el servidor RTU

        Args:
            config_path: Ruta al archivo de configuraciÃ³n YAML
        """
        self.config = self._load_config(config_path)
        self.server = None
        self.stations: Dict[int, c104.Station] = {}
        self.points: Dict[int, Dict[int, c104.Point]] = {}
        self.simulation_thread = None
        self.running = False
        self._clock_offset = datetime.timedelta(0)

        self._init_server()

    def _load_config(self, config_path: str) -> dict:
        """Carga configuraciÃ³n desde YAML"""
        try:
            with open(config_path, "r") as f:
                return yaml.safe_load(f)
        except FileNotFoundError:
            logger.warning(f"Config file not found: {config_path}, using defaults")
            return self._default_config()

    def _default_config(self) -> dict:
        """ConfiguraciÃ³n por defecto"""
        return {
            "ip": "0.0.0.0",
            "port": 2404,
            "tick_rate_ms": 100,
            "select_timeout_ms": 10000,
            "max_connections": 5,
            "protocol_parameters": {"t1": 15, "t2": 10, "t3": 20, "k": 12, "w": 8},
            "stations": [
                {
                    "common_address": 1,
                    "points": {
                        "single_points": [
                            {"ioa": 1001, "value": True, "report_ms": 10000},
                            {"ioa": 1002, "value": False, "report_ms": 10000},
                        ],
                        "double_points": [
                            {"ioa": 2001, "value": "ON", "report_ms": 10000}
                        ],
                        "step_positions": [
                            {"ioa": 3001, "value": 0, "report_ms": 10000}
                        ],
                        "measured_normalized": [
                            {"ioa": 4001, "value": 0.5, "report_ms": 10000}
                        ],
                        "measured_scaled": [
                            {"ioa": 5001, "value": 100, "report_ms": 10000},
                            {"ioa": 5002, "value": 200, "report_ms": 10000},
                        ],
                        "measured_short": [
                            {"ioa": 6001, "value": 12.34, "report_ms": 10000},
                            {"ioa": 6002, "value": 56.78, "report_ms": 10000},
                        ],
                        "binary_counters": [{"ioa": 7001, "value": 0, "report_ms": 0}],
                        "single_commands": [{"ioa": 8001, "related_ioa": 1001}],
                        "setpoint_normalized": [{"ioa": 9001, "related_ioa": 4001}],
                        "setpoint_scaled": [{"ioa": 10001, "related_ioa": 5001}],
                        "setpoint_short": [{"ioa": 11001, "related_ioa": 6001}],
                    },
                }
            ],
            "authorized_masters": [0, 1, 2, 3],
        }

    def _init_server(self):
        """Inicializa el servidor con todos los parÃ¡metros"""
        logger.info(
            f"Initializing IEC104 Server on {self.config['ip']}:{self.config['port']}"
        )

        self.server = c104.Server(
            ip=self.config["ip"],
            port=self.config["port"],
            tick_rate_ms=self.config.get("tick_rate_ms", 100),
            select_timeout_ms=self.config.get("select_timeout_ms", 10000),
            max_connections=self.config.get("max_connections", 0),
        )

        params = self.config.get("protocol_parameters", {})
        proto = self.server.protocol_parameters
        proto.message_timeout = params.get("t1", 15)
        proto.confirm_interval = params.get("t2", 10)
        proto.keep_alive_interval = params.get("t3", 20)
        proto.send_window_size = params.get("k", 12)
        proto.receive_window_size = params.get("w", 8)

        logger.info(
            f"Protocol params: T1={proto.message_timeout}s, T2={proto.confirm_interval}s, "
            f"T3={proto.keep_alive_interval}s, K={proto.send_window_size}, W={proto.receive_window_size}"
        )

        self.server.on_connect(callable=self._on_connect)
        self.server.on_clock_sync(callable=self._on_clock_sync)
        self.server.on_receive_raw(callable=self._on_receive_raw)
        self.server.on_send_raw(callable=self._on_send_raw)
        self.server.on_unexpected_message(callable=self._on_unexpected_message)

        self._create_stations_and_points()

    def _create_stations_and_points(self):
        """Crea estaciones y puntos segÃºn configuraciÃ³n"""
        for station_cfg in self.config.get("stations", []):
            ca = station_cfg["common_address"]
            station = self.server.add_station(common_address=ca)

            if not station:
                logger.error(f"Failed to create station with CA={ca}")
                continue

            self.stations[ca] = station
            self.points[ca] = {}

            logger.info(f"Created station CA={ca}")

            points_cfg = station_cfg.get("points", {})

            for sp in points_cfg.get("single_points", []):
                point = station.add_point(
                    io_address=sp["ioa"],
                    type=c104.Type.M_SP_NA_1,
                    report_ms=sp.get("report_ms", 0),
                )
                if point:
                    point.value = sp.get("value", False)
                    point.quality = c104.Quality()
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][sp["ioa"]] = point
                    logger.info(
                        f"  Point IOA={sp['ioa']} Type=M_SP_NA_1 report_ms={sp.get('report_ms', 0)}"
                    )

            for dp in points_cfg.get("double_points", []):
                point = station.add_point(
                    io_address=dp["ioa"],
                    type=c104.Type.M_DP_NA_1,
                    report_ms=dp.get("report_ms", 0),
                )
                if point:
                    val_str = dp.get("value", "OFF")
                    point.value = getattr(c104.Double, val_str, c104.Double.OFF)
                    point.quality = c104.Quality()
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][dp["ioa"]] = point
                    logger.info(f"  Point IOA={dp['ioa']} Type=M_DP_NA_1")

            for st in points_cfg.get("step_positions", []):
                point = station.add_point(
                    io_address=st["ioa"],
                    type=c104.Type.M_ST_NA_1,
                    report_ms=st.get("report_ms", 0),
                )
                if point:
                    point.value = c104.Int7(st.get("value", 0))
                    point.quality = c104.Quality()
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][st["ioa"]] = point
                    logger.info(f"  Point IOA={st['ioa']} Type=M_ST_NA_1")

            for mn in points_cfg.get("measured_normalized", []):
                point = station.add_point(
                    io_address=mn["ioa"],
                    type=c104.Type.M_ME_NA_1,
                    report_ms=mn.get("report_ms", 0),
                )
                if point:
                    point.value = c104.NormalizedFloat(mn.get("value", 0.0))
                    point.quality = c104.Quality()
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][mn["ioa"]] = point
                    logger.info(f"  Point IOA={mn['ioa']} Type=M_ME_NA_1")

            for ms in points_cfg.get("measured_scaled", []):
                point = station.add_point(
                    io_address=ms["ioa"],
                    type=c104.Type.M_ME_NB_1,
                    report_ms=ms.get("report_ms", 0),
                )
                if point:
                    point.value = c104.Int16(ms.get("value", 0))
                    point.quality = c104.Quality()
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][ms["ioa"]] = point
                    logger.info(f"  Point IOA={ms['ioa']} Type=M_ME_NB_1")

            for mf in points_cfg.get("measured_short", []):
                point = station.add_point(
                    io_address=mf["ioa"],
                    type=c104.Type.M_ME_NC_1,
                    report_ms=mf.get("report_ms", 0),
                )
                if point:
                    point.value = float(mf.get("value", 0.0))
                    point.quality = c104.Quality()
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][mf["ioa"]] = point
                    logger.info(f"  Point IOA={mf['ioa']} Type=M_ME_NC_1")
            # M_ME_TF_1 - Measured short with time tag CP56Time2a
            for mf_t in points_cfg.get("measured_short_time", []):
                point = station.add_point(
                    io_address=mf_t["ioa"],
                    type=c104.Type.M_ME_TF_1,
                    report_ms=mf_t.get("report_ms", 0),
                )
                if point:
                    point.value = float(mf_t.get("value", 0.0))
                    point.quality = c104.Quality()
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][mf_t["ioa"]] = point
                    logger.info(f"  Point IOA={mf_t['ioa']} Type=M_ME_TF_1")

            # M_SP_TB_1 - Single point with time tag CP56Time2a
            for sp_t in points_cfg.get("single_points_time", []):
                point = station.add_point(
                    io_address=sp_t["ioa"],
                    type=c104.Type.M_SP_TB_1,
                    report_ms=sp_t.get("report_ms", 0),
                )
                if point:
                    point.value = sp_t.get("value", False)
                    point.quality = c104.Quality()
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][sp_t["ioa"]] = point
                    logger.info(f"  Point IOA={sp_t['ioa']} Type=M_SP_TB_1")

            # C_SC_TA_1 - Single command with time tag
            for sc_t in points_cfg.get("single_commands_time", []):
                point = station.add_point(
                    io_address=sc_t["ioa"],
                    type=c104.Type.C_SC_TA_1,
                    related_io_address=sc_t.get("related_ioa"),
                    related_io_autoreturn=True,
                    command_mode=c104.CommandMode.DIRECT,
                )
                if point:
                    point.on_receive(callable=self._on_command_receive)
                    self.points[ca][sc_t["ioa"]] = point
                    logger.info(
                        f"  Point IOA={sc_t['ioa']} Type=C_SC_TA_1 (DIRECT mode)"
                    )

            # C_SE_TC_1 - Setpoint short with time tag
            for sf_t in points_cfg.get("setpoint_short_time", []):
                point = station.add_point(
                    io_address=sf_t["ioa"],
                    type=c104.Type.C_SE_TC_1,
                    related_io_address=sf_t.get("related_ioa"),
                    related_io_autoreturn=True,
                    command_mode=c104.CommandMode.DIRECT,
                )
                if point:
                    point.on_receive(callable=self._on_command_receive)
                    self.points[ca][sf_t["ioa"]] = point
                    logger.info(f"  Point IOA={sf_t['ioa']} Type=C_SE_TC_1")

            for bc in points_cfg.get("binary_counters", []):
                point = station.add_point(
                    io_address=bc["ioa"],
                    type=c104.Type.M_IT_NA_1,
                    report_ms=bc.get("report_ms", 0),
                )
                if point:
                    point.value = bc.get("value", 0)
                    point.on_before_auto_transmit(
                        callable=self._on_before_auto_transmit
                    )
                    point.on_before_read(callable=self._on_before_read)
                    self.points[ca][bc["ioa"]] = point
                    logger.info(f"  Point IOA={bc['ioa']} Type=M_IT_NA_1")

            for sc in points_cfg.get("single_commands", []):
                point = station.add_point(
                    io_address=sc["ioa"],
                    type=c104.Type.C_SC_NA_1,
                    related_io_address=sc.get("related_ioa"),
                    related_io_autoreturn=True,
                    command_mode=c104.CommandMode.DIRECT,
                )
                if point:
                    point.on_receive(callable=self._on_command_receive)
                    self.points[ca][sc["ioa"]] = point
                    logger.info(f"  Point IOA={sc['ioa']} Type=C_SC_NA_1 (DIRECT mode)")

            for sn in points_cfg.get("setpoint_normalized", []):
                point = station.add_point(
                    io_address=sn["ioa"],
                    type=c104.Type.C_SE_NA_1,
                    related_io_address=sn.get("related_ioa"),
                    related_io_autoreturn=True,
                    command_mode=c104.CommandMode.DIRECT,
                )
                if point:
                    point.on_receive(callable=self._on_command_receive)
                    self.points[ca][sn["ioa"]] = point
                    logger.info(f"  Point IOA={sn['ioa']} Type=C_SE_NA_1")

            for ss in points_cfg.get("setpoint_scaled", []):
                point = station.add_point(
                    io_address=ss["ioa"],
                    type=c104.Type.C_SE_NB_1,
                    related_io_address=ss.get("related_ioa"),
                    related_io_autoreturn=True,
                    command_mode=c104.CommandMode.DIRECT,
                )
                if point:
                    point.on_receive(callable=self._on_command_receive)
                    self.points[ca][ss["ioa"]] = point
                    logger.info(f"  Point IOA={ss['ioa']} Type=C_SE_NB_1")

            for sf in points_cfg.get("setpoint_short", []):
                point = station.add_point(
                    io_address=sf["ioa"],
                    type=c104.Type.C_SE_NC_1,
                    related_io_address=sf.get("related_ioa"),
                    related_io_autoreturn=True,
                    command_mode=c104.CommandMode.DIRECT,
                )
                if point:
                    point.on_receive(callable=self._on_command_receive)
                    self.points[ca][sf["ioa"]] = point
                    logger.info(f"  Point IOA={sf['ioa']} Type=C_SE_NC_1")

    def _get_synchronized_time(self) -> datetime.datetime:
        """Obtiene timestamp sincronizado (UTC + offset de sync)"""
        return datetime.datetime.now(datetime.timezone.utc) + self._clock_offset

    def _on_connect(self, server: c104.Server, ip: str) -> bool:
        """Callback: solicitud de conexiÃ³n"""
        logger.info(f"Connection request from {ip}")
        return True

    def _on_clock_sync(
        self, server: c104.Server, ip: str, date_time: datetime.datetime
    ) -> c104.ResponseState:
        """Callback: sincronizaciÃ³n de reloj (C_CS_NA_1)"""
        logger.info(f"Clock sync command from {ip}: {date_time.isoformat()}")

        if date_time.tzinfo is None:
            date_time = date_time.replace(tzinfo=datetime.timezone.utc)

        local_time = datetime.datetime.now(datetime.timezone.utc)
        self._clock_offset = date_time - local_time

        logger.info(
            f"Clock offset adjusted to: {self._clock_offset.total_seconds():.3f} seconds"
        )
        logger.info(f"Synchronized time: {self._get_synchronized_time().isoformat()}")

        return c104.ResponseState.SUCCESS

    def _on_receive_raw(self, server: c104.Server, data: bytes) -> None:
        """Callback: mensaje entrante (raw)"""
        logger.debug(f"RX: {data.hex()} | {c104.explain_bytes(apdu=data)}")

    def _on_send_raw(self, server: c104.Server, data: bytes) -> None:
        """Callback: mensaje saliente (raw)"""
        logger.debug(f"TX: {data.hex()} | {c104.explain_bytes(apdu=data)}")

    def _on_unexpected_message(
        self, server: c104.Server, message: c104.IncomingMessage, cause: c104.Umc
    ) -> None:
        """Callback: mensaje inesperado"""
        logger.warning(
            f"Unexpected message: {cause} | CA={message.common_address}, "
            f"IOA={message.io_address}, Type={message.type}, COT={message.cot}"
        )

    def _on_before_auto_transmit(self, point: c104.Point) -> None:
        """Callback: antes de transmisiÃ³n automÃ¡tica (periÃ³dica)"""
        current_time = self._get_synchronized_time()
        logger.debug(
            f"Auto-transmit point IOA={point.io_address}, Type={point.type}, "
            f"Value={point.value}, Time={current_time.isoformat()}"
        )

    def _on_before_read(self, point: c104.Point) -> None:
        """Callback: antes de lectura (interrogaciÃ³n)"""
        current_time = self._get_synchronized_time()
        logger.debug(
            f"Read point IOA={point.io_address}, Type={point.type}, "
            f"Value={point.value}, Time={current_time.isoformat()}"
        )

    def _on_command_receive(
        self,
        point: c104.Point,
        previous_info: c104.Information,
        message: c104.IncomingMessage,
    ) -> c104.ResponseState:
        """Callback: recepciÃ³n de comando"""
        logger.info(
            f"Command received: IOA={point.io_address}, Type={point.type}, "
            f"COT={message.cot}, Value={point.value}, Quality={point.quality}"
        )

        if point.related_io_address:
            station = point.station
            if station:
                related_point = station.get_point(point.related_io_address)
                if related_point:
                    related_point.value = point.value
                    related_point.quality = c104.Quality()

                    if related_point.transmit(cause=c104.Cot.SPONTANEOUS):
                        logger.info(
                            f"Related point IOA={related_point.io_address} updated spontaneously"
                        )

        return c104.ResponseState.SUCCESS

    def _simulation_loop(self):
        """Thread de simulaciÃ³n de cambios de valores"""
        logger.info("Simulation thread started")

        while self.running:
            try:
                time.sleep(random.uniform(10.0, 20.0))

                if not self.stations:
                    continue

                ca = random.choice(list(self.stations.keys()))
                station = self.stations[ca]

                monitoring_points = [
                    p
                    for p in self.points[ca].values()
                    if p.type
                    in [
                        c104.Type.M_SP_NA_1,
                        c104.Type.M_ME_NA_1,
                        c104.Type.M_ME_NB_1,
                        c104.Type.M_ME_NC_1,
                    ]
                ]

                if not monitoring_points:
                    continue

                point = random.choice(monitoring_points)

                if point.type == c104.Type.M_SP_NA_1:
                    point.value = not point.value
                elif point.type == c104.Type.M_ME_NA_1:
                    point.value = c104.NormalizedFloat(random.uniform(-1.0, 1.0))
                elif point.type == c104.Type.M_ME_NB_1:
                    point.value = c104.Int16(random.randint(-1000, 1000))
                elif point.type == c104.Type.M_ME_NC_1:
                    point.value = random.uniform(-100.0, 100.0)

                point.quality = c104.Quality()

                if point.transmit(cause=c104.Cot.SPONTANEOUS):
                    current_time = self._get_synchronized_time()
                    logger.info(
                        f"Spontaneous event: CA={ca}, IOA={point.io_address}, "
                        f"Type={point.type}, Value={point.value}, Time={current_time.isoformat()}"
                    )

            except Exception as e:
                logger.error(f"Simulation error: {e}", exc_info=True)

    def start(self):
        """Inicia el servidor"""
        if self.running:
            logger.warning("Server already running")
            return

        logger.info("Starting IEC104 Server...")
        self.server.start()
        self.running = True

        self.simulation_thread = threading.Thread(
            target=self._simulation_loop, daemon=True
        )
        self.simulation_thread.start()

        for ca, station in self.stations.items():
            station.signal_initialized(cause=c104.Coi.LOCAL_POWER_ON)
            logger.info(f"Station CA={ca} initialized")

        logger.info("Server started successfully")

    def stop(self):
        """Detiene el servidor"""
        if not self.running:
            return

        logger.info("Stopping IEC104 Server...")
        self.running = False

        if self.simulation_thread:
            self.simulation_thread.join(timeout=2.0)

        self.server.stop()
        logger.info("Server stopped")

    def run_forever(self):
        """Ejecuta el servidor indefinidamente"""
        self.start()
        try:
            while True:
                time.sleep(1)
        except KeyboardInterrupt:
            logger.info("Keyboard interrupt received")
        finally:
            self.stop()


def main():
    """FunciÃ³n principal"""
    import argparse

    parser = argparse.ArgumentParser(description="IEC 60870-5-104 Server (RTU)")
    parser.add_argument(
        "--config",
        type=str,
        default="server_config.yaml",
        help="Path to configuration file",
    )
    parser.add_argument("--debug", action="store_true", help="Enable debug logging")

    args = parser.parse_args()

    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)
        c104.set_debug_mode(
            c104.Debug.Server | c104.Debug.Connection | c104.Debug.Point
        )

    server = IEC104Server(config_path=args.config)
    server.run_forever()


if __name__ == "__main__":
    main()
