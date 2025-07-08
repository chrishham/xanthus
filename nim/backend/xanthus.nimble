# Package

version       = "0.1.0"
author        = "Xanthus Team"
description   = "Xanthus Infrastructure Management Platform - Nim Backend"
license       = "MIT"
srcDir        = "src"
bin           = @["xanthus"]

# Dependencies

requires "nim >= 1.6.0"
requires "jester >= 0.5.0"
requires "jwt >= 0.2.0"
requires "asynctools >= 0.1.0"
requires "httpx >= 0.3.0"
requires "jsony >= 1.1.0"
requires "redis >= 0.3.0"
requires "sqlite3 >= 0.1.0"
requires "argparse >= 4.0.0"