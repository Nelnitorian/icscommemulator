#!/usr/bin/env python3

"""
Servidor IEC 104 - RTU (Remote Terminal Unit)
Este servidor simula una unidad terminal remota con soporte completo de tipos de datos.
"""

import c104
import time
import threading
import yaml
import random
import datetime


class IEC104Server:
    def __init__(self, config_file="server_config.yaml"):
        """
        Inicializa el servidor IEC 104 desde archivo de configuración

        Args:
            config_file: Ruta al archivo YAML de configuración
        """
        # Cargar configuración
        with open(config_file, "r") as f:
            self.config = yaml.safe_load(f)

        print(
            "[SERVER] Parámetros APCI configurados (valores por defecto de lib60870-C):"
        )
        print(f"  t1={self.config.get('t1', 15)}s (timeout de confirmación)")
        print(f"  t2={self.config.get('t2', 10)}s (timeout de ACK)")
        print(f"  t3={self.config.get('t3', 20)}s (timeout de test)")
        print(f"  k={self.config.get('k', 12)} (ventana de envío)")
        print(f"  w={self.config.get('w', 8)} (ventana de recepción)")

        # Crear servidor
        self.server = c104.Server(
            ip=self.config["ip"],
            port=self.config["port"],
            tick_rate_ms=100,
            max_connections=5,
        )

        # Agregar estación
        self.station = self.server.add_station(
            common_address=self.config["common_address"]
        )

        # Almacenar puntos por categoría
        self.points = {
            "single": [],
            "double": [],
            "measured_scaled": [],
            "measured_normalized": [],
            "commands": [],  # NUEVO: Puntos de comando
        }

        # Diccionario para valores de mediciones
        self._point_values = {}

        # Agregar puntos desde configuración
        self._setup_points()

        # Estado del servidor
        self.running = False
        self.simulation_thread = None

        # Configurar callbacks
        self._setup_callbacks()

    def _setup_points(self):
        """Configura los puntos desde la configuración YAML"""

        # Single Points (M_SP_NA_1) - Tipo 1
        if "single_points" in self.config:
            sp_config = self.config["single_points"]
            values = sp_config["values"]
            start_ioa = sp_config["start_ioa"]

            for i, value in enumerate(values):
                ioa = start_ioa + i
                point = self.station.add_point(
                    io_address=ioa,
                    type=c104.Type.M_SP_NA_1,
                    report_ms=5000,
                )
                point.value = bool(int(value))
                self.points["single"].append(point)
                self._point_values[ioa] = bool(int(value))
                print(f"[SERVER] Punto M_SP_NA_1 creado: IOA {ioa} = {value}")

        # Double Points (M_DP_NA_1) - Tipo 3
        if "double_points" in self.config:
            dp_config = self.config["double_points"]
            values = dp_config["values"]
            start_ioa = dp_config["start_ioa"]

            for i, value in enumerate(values):
                ioa = start_ioa + i
                point = self.station.add_point(
                    io_address=ioa,
                    type=c104.Type.M_DP_NA_1,
                    report_ms=5000,
                )

                double_map = {
                    0: c104.Double.INTERMEDIATE,
                    1: c104.Double.OFF,
                    2: c104.Double.ON,
                    3: c104.Double.INTERMEDIATE,
                }
                double_val = double_map.get(int(value), c104.Double.INTERMEDIATE)
                point.value = double_val
                self.points["double"].append(point)
                self._point_values[ioa] = double_val
                print(f"[SERVER] Punto M_DP_NA_1 creado: IOA {ioa} = {value}")

        # Measured Scaled (M_ME_NB_1) - Tipo 11
        if "measured_scaled" in self.config:
            ms_config = self.config["measured_scaled"]
            values = ms_config["values"]
            start_ioa = ms_config["start_ioa"]

            for i, value in enumerate(values):
                ioa = start_ioa + i
                point = self.station.add_point(
                    io_address=ioa,
                    type=c104.Type.M_ME_NB_1,
                    report_ms=5000,
                )

                # Crear objeto Int16 para valor inicial
                int_val = c104.Int16(int(value))
                point.value = int_val
                self.points["measured_scaled"].append(point)
                self._point_values[ioa] = int(value)
                print(
                    f"[SERVER] Punto M_ME_NB_1 creado: IOA {ioa} (valor inicial: {value})"
                )

        # Measured Normalized (M_ME_NA_1) - Tipo 9
        if "measured_normalized" in self.config:
            mn_config = self.config["measured_normalized"]
            values = mn_config["values"]
            start_ioa = mn_config["start_ioa"]

            for i, value in enumerate(values):
                ioa = start_ioa + i
                point = self.station.add_point(
                    io_address=ioa, type=c104.Type.M_ME_NA_1, report_ms=5000
                )

                # Normalizar el valor
                normalized = (int(value) - 32768) / 32768.0
                norm_val = c104.NormalizedFloat(normalized)
                point.value = norm_val
                self.points["measured_normalized"].append(point)
                self._point_values[ioa] = normalized
                print(
                    f"[SERVER] Punto M_ME_NA_1 creado: IOA {ioa} (valor inicial: {value}, norm: {normalized:.4f})"
                )

        # NUEVO: Command Points - Para soportar comandos del cliente
        if "command_points" in self.config:
            cmd_config = self.config["command_points"]

            # C_SC_NA_1 - Single Command (Tipo 45)
            if "single_commands" in cmd_config:
                for ioa in cmd_config["single_commands"]:
                    point = self.station.add_point(
                        io_address=ioa,
                        type=c104.Type.C_SC_NA_1,
                        report_ms=0,  # Los comandos no reportan automáticamente
                    )

                    # Registrar callback para procesar comandos
                    def on_command_handler(
                        point: c104.Point,
                        previous_info: c104.Information,
                        message: c104.IncomingMessage,
                    ) -> c104.ResponseState:
                        print(
                            f"[SERVER] Comando C_SC_NA_1 recibido en IOA {point.io_address}: {point.value}"
                        )
                        # Actualizar punto de monitoreo correspondiente si existe
                        monitor_ioa = point.io_address
                        if monitor_ioa in self._point_values:
                            self._point_values[monitor_ioa] = point.value
                        return c104.ResponseState.SUCCESS

                    point.on_before_auto_transmit(callable=on_command_handler)
                    self.points["commands"].append(point)
                    print(f"[SERVER] Punto comando C_SC_NA_1 creado: IOA {ioa}")

            # C_SE_NC_1 - Set-point Short Float (Tipo 49)
            if "setpoint_commands" in cmd_config:
                for ioa in cmd_config["setpoint_commands"]:
                    point = self.station.add_point(
                        io_address=ioa,
                        type=c104.Type.C_SE_NC_1,
                        report_ms=0,
                    )

                    def on_setpoint_handler(
                        point: c104.Point,
                        previous_info: c104.Information,
                        message: c104.IncomingMessage,
                    ) -> c104.ResponseState:
                        print(
                            f"[SERVER] Set-point C_SE_NC_1 recibido en IOA {point.io_address}: {point.value}"
                        )
                        if point.io_address in self._point_values:
                            self._point_values[point.io_address] = float(point.value)
                        return c104.ResponseState.SUCCESS

                    point.on_before_auto_transmit(callable=on_setpoint_handler)
                    self.points["commands"].append(point)
                    print(f"[SERVER] Punto comando C_SE_NC_1 creado: IOA {ioa}")

    def _setup_callbacks(self):
        """Configura los callbacks del servidor"""

        def on_clock_sync(
            server: c104.Server, ip: str, date_time: datetime.datetime
        ) -> c104.ResponseState:
            print(f"[SERVER] Sincronización de reloj desde {ip}: {date_time}")
            return c104.ResponseState.SUCCESS

        def on_unexpected_message(
            server: c104.Server, message: c104.IncomingMessage, cause: c104.Umc
        ) -> None:
            print(f"[SERVER] Mensaje inesperado: {message.type}, causa: {cause}")

        def on_receive_raw(server: c104.Server, data: bytes) -> None:
            print(f"[SERVER] RAW recibido: {len(data)} bytes")

        def on_send_raw(server: c104.Server, data: bytes) -> None:
            print(f"[SERVER] RAW enviado: {len(data)} bytes")

        # Asignar callbacks del servidor
        self.server.on_clock_sync(callable=on_clock_sync)
        self.server.on_unexpected_message(callable=on_unexpected_message)
        self.server.on_receive_raw(callable=on_receive_raw)
        self.server.on_send_raw(callable=on_send_raw)

    def _simulate_measurements(self):
        """Hilo de simulación para actualizar mediciones periódicamente"""
        print("[SERVER] Iniciando simulación de mediciones...")

        while self.running:
            try:
                # Para mediciones escaladas (M_ME_NB_1)
                for point in self.points["measured_scaled"]:
                    ioa = point.io_address
                    if ioa in self._point_values:
                        current_value = self._point_values[ioa]
                        variation = random.uniform(-10, 10)
                        new_value = int(current_value + variation)
                        new_value = max(-32768, min(32767, new_value))

                        if new_value != current_value:
                            self._point_values[ioa] = new_value
                            point.value = c104.Int16(new_value)
                            point.transmit(cause=c104.Cot.SPONTANEOUS)

                # Para mediciones normalizadas (M_ME_NA_1)
                for point in self.points["measured_normalized"]:
                    ioa = point.io_address
                    if ioa in self._point_values:
                        current_value = self._point_values[ioa]
                        variation = random.uniform(-0.05, 0.05)
                        new_value = max(-1.0, min(1.0, current_value + variation))

                        if abs(new_value - current_value) > 0.001:
                            self._point_values[ioa] = new_value
                            point.value = c104.NormalizedFloat(new_value)
                            point.transmit(cause=c104.Cot.SPONTANEOUS)

                time.sleep(2)

            except Exception as e:
                print(f"[SERVER] Error en simulación: {e}")
                import traceback

                traceback.print_exc()
                break

    def start(self):
        """Inicia el servidor"""
        identity = self.config.get("identity", {})
        print("[SERVER] Iniciando servidor IEC 104...")
        print(f"[SERVER] Estación: {identity.get('station_name', 'N/A')}")
        print(f"[SERVER] Ubicación: {identity.get('location', 'N/A')}")
        print(f"[SERVER] Versión: {identity.get('version', 'N/A')}")
        print(f"[SERVER] Escuchando en {self.config['ip']}:{self.config['port']}")

        self.server.start()
        self.running = True

        # Iniciar hilo de simulación
        self.simulation_thread = threading.Thread(
            target=self._simulate_measurements, daemon=True
        )
        self.simulation_thread.start()
        print("[SERVER] Servidor iniciado y listo")

    def stop(self):
        """Detiene el servidor"""
        print("[SERVER] Deteniendo servidor...")
        self.running = False
        if self.simulation_thread:
            self.simulation_thread.join(timeout=2)
        self.server.stop()
        print("[SERVER] Servidor detenido")

    def get_point_value(self, io_address):
        """Obtiene el valor actual de un punto desde el diccionario interno"""
        return self._point_values.get(io_address)

    def get_all_points_info(self):
        """Obtiene información de todos los puntos"""
        info_list = []
        for category, points in self.points.items():
            for point in points:
                ioa = point.io_address
                value = self._point_values.get(ioa)
                if value is not None:
                    info_list.append(
                        {
                            "category": category,
                            "io_address": ioa,
                            "type": point.type,
                            "value": value,
                            "quality": "Good",
                        }
                    )
        return info_list


if __name__ == "__main__":
    server = IEC104Server(config_file="server_config.yaml")
    try:
        server.start()
        print("\n[SERVER] Servidor en ejecución. Presiona Ctrl+C para detener.\n")

        while True:
            time.sleep(5)
            if int(time.time()) % 30 == 0:
                print("\n[SERVER] Estado de puntos:")
                for info in server.get_all_points_info()[:5]:
                    print(
                        f"  IOA {info['io_address']} ({info['category']}): {info['value']}"
                    )
                print()

    except KeyboardInterrupt:
        print("\n\nInterrupción detectada...")
    finally:
        server.stop()
