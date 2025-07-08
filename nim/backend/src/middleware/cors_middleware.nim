import mummy

proc corsMiddleware*(request: Request): bool =
  # This would be used as a middleware check
  # For now, we'll return true to continue processing
  # CORS headers will be added in individual handlers
  return true

proc addCorsHeaders*(headers: var HttpHeaders) =
  headers["Access-Control-Allow-Origin"] = "*"
  headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, DELETE, OPTIONS"
  headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization"
  headers["Access-Control-Max-Age"] = "86400"

proc handleOptionsRequest*(request: Request) =
  var headers: HttpHeaders
  addCorsHeaders(headers)
  request.respond(200, headers, "OK")