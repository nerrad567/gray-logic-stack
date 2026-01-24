"""Periodic scenario engine for sensor simulation.

Runs background threads that update sensor values on a schedule and
push unsolicited L_DATA.ind telegrams to connected clients.

Supported patterns:
  - sine_wave: Oscillates around a center value
  - random_walk: Drifts randomly within bounds
"""

import logging
import math
import random
import threading
import time
from typing import Callable

from devices.sensor import Sensor

logger = logging.getLogger("knxsim.scenarios")


class ScenarioRunner:
    """Manages all periodic scenarios in background threads."""

    def __init__(self, send_telegram: Callable[[bytes], None]):
        """
        Args:
            send_telegram: Function to broadcast a cEMI frame to all connected clients.
        """
        self._send_telegram = send_telegram
        self._threads: list[threading.Thread] = []
        self._running = False

    def add_scenario(
        self, device: Sensor, field: str, scenario_type: str, params: dict
    ):
        """Register a scenario for a sensor field."""
        if scenario_type == "sine_wave":
            target = self._run_sine_wave
        elif scenario_type == "random_walk":
            target = self._run_random_walk
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
