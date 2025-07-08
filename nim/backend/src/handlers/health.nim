import json, times, mummy

proc healthHandler*(request: Request) =
  let response = %* {
    "status": "healthy",
    "service": "xanthus-nim-backend",
    "timestamp": now().format("yyyy-MM-dd'T'HH:mm:ss'Z'"),
    "version": "0.1.0"
  }
  
  var headers: HttpHeaders
  headers["Content-Type"] = "application/json"
  request.respond(200, headers, $response)