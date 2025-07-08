import jester, json, times

proc healthHandler*(): string =
  let response = %* {
    "status": "healthy",
    "service": "xanthus-nim-backend",
    "timestamp": now().format("yyyy-MM-dd'T'HH:mm:ss'Z'"),
    "version": "0.1.0"
  }
  
  result = $response