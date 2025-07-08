import jester

proc corsMiddleware*(): void =
  headers["Access-Control-Allow-Origin"] = "*"
  headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, DELETE, OPTIONS"
  headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization"
  headers["Access-Control-Max-Age"] = "86400"
  
  if request.httpMethod == HttpOptions:
    halt Http200, "OK"