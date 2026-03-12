from __future__ import annotations

import os

__all__ = ["__version__"]

__version__ = os.environ.get("MOLTBOX_BUILD_VERSION") or os.environ.get("REMRAM_BUILD_VERSION", "0.3.0")
