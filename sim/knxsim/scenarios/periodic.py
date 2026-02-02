"""Periodic scenario engine for sensor simulation.

Runs background threads that update sensor values on a schedule and
push unsolicited L_DATA.ind telegrams to connected clients.

Supported patterns:
  - sine_wave: Oscillates around a center value
  - random_walk: Drifts randomly within bounds
  - presence_pattern: Simulates realistic occupancy transitions
  - thermal_simulation: Realistic room temperature based on heating state
"""

import logging
import math
import random
import threading
import time
from collections.abc import Callable

from devices.base import BaseDevice
from devices.presence import PresenceSensor
from devices.sensor import Sensor
from devices.thermostat import Thermostat

logger = logging.getLogger("knxsim.scenarios")


class ScenarioRunner:
    """Manages all periodic scenarios in background threads."""

    def __init__(
        self,
        send_telegram: Callable[[bytes], None],
        dispatch_local: Callable[[int, bytes], None] | None = None,
        on_state_change: Callable[[str, dict], None] | None = None,
        premise_id: str = "default",
    ):
        """
        Args:
            send_telegram: Function to broadcast a cEMI frame to all connected clients.
            dispatch_local: Optional function to dispatch payload to local devices on a GA.
            on_state_change: Optional callback(device_id, state) for WebSocket pushes.
            premise_id: Premise ID for state change callbacks.
        """
        self._send_telegram = send_telegram
        self._dispatch_local = dispatch_local
        self._on_state_change = on_state_change
        self._premise_id = premise_id
        self._threads: list[threading.Thread] = []
        self._running = False

    def add_scenario(
        self, device: BaseDevice, field: str, scenario_type: str, params: dict
    ):
        """Register a scenario for a sensor field."""
        if scenario_type == "sine_wave":
            target = self._run_sine_wave
        elif scenario_type == "random_walk":
            target = self._run_random_walk
        elif scenario_type == "presence_pattern":
            target = self._run_presence_pattern
        elif scenario_type == "thermal_simulation":
            target = self._run_thermal_simulation
        else:
            logger.warning("Unknown scenario type: %s", scenario_type)
            return

        t = threading.Thread(
            target=target,
            args=(device, field, params),
            name=f"scenario-{device.device_id}-{field}",
            daemon=True,
        )
        self._threads.append(t)

    def start(self):
        """Start all scenario threads."""
        self._running = True
        for t in self._threads:
            t.start()
        if self._threads:
            logger.info("Started %d scenario(s)", len(self._threads))

    def stop(self):
        """Signal all threads to stop."""
        self._running = False

    def _run_sine_wave(self, device: Sensor, field: str, params: dict):
        """Sine wave: value = center + amplitude * sin(2π * t / period).

        Params:
            center: Base value (e.g., 21.5 for temperature)
            amplitude: Swing amount (e.g., 2.0 = ±2°C)
            period_minutes: Full cycle duration
            interval_seconds: How often to send updates
        """
        center = params.get("center", 20.0)
        amplitude = params.get("amplitude", 2.0)
        period = params.get("period_minutes", 30) * 60  # Convert to seconds
        interval = params.get("interval_seconds", 10)

        start_time = time.time()

        while self._running:
            elapsed = time.time() - start_time
            value = center + amplitude * math.sin(2 * math.pi * elapsed / period)

            # Update device state
            device.state[field] = round(value, 2)

            # Send unsolicited indication
            cemi = device.get_indication(field)
            if cemi:
                self._send_telegram(cemi)
                logger.info("%s → %s = %.2f", device.device_id, field, value)

            # Sleep in small increments so we can check _running
            deadline = time.time() + interval
            while self._running and time.time() < deadline:
                time.sleep(0.5)

    def _run_random_walk(self, device: Sensor, field: str, params: dict):
        """Random walk: value drifts by small random steps within bounds.

        Params:
            center: Starting value
            step: Max change per update (e.g., 0.5)
            min_value: Lower bound
            max_value: Upper bound
            interval_seconds: How often to send updates
        """
        center = params.get("center", 20.0)
        step = params.get("step", 0.3)
        min_val = params.get("min_value", center - 5)
        max_val = params.get("max_value", center + 5)
        interval = params.get("interval_seconds", 10)

        value = device.state.get(field, center)

        while self._running:
            # Random step
            delta = random.uniform(-step, step)
            value = max(min_val, min(max_val, value + delta))

            # Update device state
            device.state[field] = round(value, 2)

            # Send unsolicited indication
            cemi = device.get_indication(field)
            if cemi:
                self._send_telegram(cemi)
                logger.debug("%s → %s = %.2f", device.device_id, field, value)

            # Sleep in small increments
            deadline = time.time() + interval
            while self._running and time.time() < deadline:
                time.sleep(0.5)

    def _run_presence_pattern(self, device: PresenceSensor, field: str, params: dict):
        """Presence pattern: simulates realistic occupancy with movement.

        Models a person moving through a room — occupied for a while (sitting,
        working, watching TV), then brief empty periods (leaving to kitchen,
        bathroom), then occupied again. Occasionally longer absences.

        Params:
            occupied_min_seconds: Minimum time occupied (default: 30)
            occupied_max_seconds: Maximum time occupied (default: 180)
            empty_min_seconds: Minimum time empty (default: 5)
            empty_max_seconds: Maximum time empty (default: 45)
            long_absence_chance: Probability of a longer absence (default: 0.15)
            long_absence_max_seconds: Max duration of long absence (default: 120)
            lux_occupied: Lux level when room is occupied (optional, for lux GA)
            lux_empty: Lux level when room is empty (optional)
        """
        occ_min = params.get("occupied_min_seconds", 30)
        occ_max = params.get("occupied_max_seconds", 180)
        empty_min = params.get("empty_min_seconds", 5)
        empty_max = params.get("empty_max_seconds", 45)
        long_absence_chance = params.get("long_absence_chance", 0.15)
        long_absence_max = params.get("long_absence_max_seconds", 120)
        lux_occupied = params.get("lux_occupied")
        lux_empty = params.get("lux_empty")

        # Start as occupied
        occupied = True
        device.state[field] = True

        while self._running:
            # Set presence state
            device.state[field] = occupied

            # Send presence telegram
            cemi = device.get_indication_bool(field)
            if cemi:
                self._send_telegram(cemi)
                logger.info(
                    "%s → %s = %s",
                    device.device_id,
                    field,
                    "OCCUPIED" if occupied else "EMPTY",
                )

            # Optionally update lux to correlate with presence
            if lux_occupied is not None and lux_empty is not None:
                lux_val = lux_occupied if occupied else lux_empty
                # Add slight randomness to lux (±10%)
                lux_val *= random.uniform(0.9, 1.1)
                device.state["lux"] = round(lux_val, 1)
                lux_cemi = device.get_indication_float("lux")
                if lux_cemi:
                    self._send_telegram(lux_cemi)
                    logger.debug("%s → lux = %.1f", device.device_id, lux_val)

            # Determine how long to hold this state
            if occupied:
                hold_time = random.uniform(occ_min, occ_max)
            else:
                # Chance of a longer absence (person left the room properly)
                if random.random() < long_absence_chance:
                    hold_time = random.uniform(empty_max, long_absence_max)
                else:
                    hold_time = random.uniform(empty_min, empty_max)

            # Sleep in small increments so we can check _running
            deadline = time.time() + hold_time
            while self._running and time.time() < deadline:
                time.sleep(0.5)

            # Toggle state
            occupied = not occupied

    def _run_thermal_simulation(self, device: Thermostat, field: str, params: dict):
        """Thermal simulation: realistic room temperature based on heating state.

        Models real-world thermal behavior:
        - When heating (valve open), temperature rises toward setpoint
        - When not heating, temperature falls toward ambient (heat loss)
        - Rate of change proportional to valve opening percentage
        - Includes thermal mass (temperature changes gradually)

        Params:
            ambient_temp: External/ambient temperature (default: 10°C)
            heating_power: Max temp rise per minute at 100% (default: 0.15°C/min)
            heat_loss_rate: Temp drop per minute toward ambient (default: 0.03°C/min)
            interval_seconds: Update frequency (default: 10)
        """
        ambient = params.get("ambient_temp", 10.0)
        heating_power = params.get("heating_power", 0.15)  # °C/min at 100%
        heat_loss = params.get("heat_loss_rate", 0.03)  # °C/min toward ambient
        interval = params.get("interval_seconds", 10)

        logger.info(
            "%s: thermal simulation started (ambient=%.1f°C, power=%.2f, loss=%.3f, interval=%ds)",
            device.device_id, ambient, heating_power, heat_loss, interval
        )

        last_time = time.time()

        while self._running:
            now = time.time()
            elapsed_minutes = (now - last_time) / 60.0
            last_time = now

            # Get current state
            current_temp = device.state.get("current_temperature", 20.0)
            heating_output = device.state.get("heating_output", 0)  # 0-100%

            # Calculate temperature change
            # 1. Heat gain from heating system (proportional to valve %)
            heat_gain = (heating_output / 100.0) * heating_power * elapsed_minutes

            # 2. Heat loss to ambient (proportional to temp difference)
            temp_diff = current_temp - ambient
            heat_loss_amount = heat_loss * temp_diff * elapsed_minutes

            # Net change
            delta = heat_gain - heat_loss_amount
            new_temp = current_temp + delta

            # Clamp to reasonable bounds
            new_temp = max(ambient - 2, min(40, new_temp))

            # Only update if changed significantly (avoid noise)
            if abs(new_temp - current_temp) >= 0.01:
                device.state["current_temperature"] = round(new_temp, 2)

                # Recalculate heating output (thermostat PID)
                old_output = device.state.get("heating_output", 0)
                device.recalculate_output()
                new_output = device.state.get("heating_output", 0)

                # Send temperature telegram
                cemi = device.get_indication("current_temperature")
                if cemi:
                    self._send_telegram(cemi)

                # Push state change to WebSocket
                if self._on_state_change:
                    self._on_state_change(
                        self._premise_id, device.device_id, dict(device.state)
                    )

                # If heating output changed, send that too and dispatch to local devices
                if new_output != old_output:
                    output_cemi = device.get_indication("heating_output")
                    if output_cemi:
                        self._send_telegram(output_cemi)
                        # Also dispatch to local devices (e.g., valve actuator)
                        if self._dispatch_local:
                            from devices.base import encode_dpt5
                            heating_ga = device.group_addresses.get("heating_output")
                            if heating_ga:
                                self._dispatch_local(heating_ga, encode_dpt5(new_output))

                logger.info(
                    "%s: temp=%.2f°C (Δ%.3f) heating=%d%% (ambient=%.1f°C)",
                    device.device_id,
                    new_temp,
                    delta,
                    new_output,
                    ambient,
                )

            # Sleep in small increments
            deadline = time.time() + interval
            while self._running and time.time() < deadline:
                time.sleep(0.5)
