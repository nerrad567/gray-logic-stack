"""Pytest configuration for KNXSim tests."""

from __future__ import annotations

import os
import sys

# Ensure sim/knxsim is on sys.path for direct module imports.
_TESTS_DIR = os.path.dirname(__file__)
_PROJECT_ROOT = os.path.abspath(os.path.join(_TESTS_DIR, ".."))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)
