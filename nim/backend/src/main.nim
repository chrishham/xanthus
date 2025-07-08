import mummy, mummy/routers, json
import handlers/health
import middleware/cors_middleware

proc indexHandler(request: Request) =
  var headers: HttpHeaders
  headers["Content-Type"] = "text/plain"
  addCorsHeaders(headers)
  request.respond(200, headers, "Xanthus Nim Backend is running")

proc healthHandlerWrapper(request: Request) =
  healthHandler(request)

proc loginHandler(request: Request) =
  # Placeholder for login - will implement JWT validation
  # This is a fake JWT token with header.payload.signature format
  let fakeJwt = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjk5OTk5OTk5OTl9.fake-signature"
  let response = %* {
    "status": "success",
    "data": {
      "jwt_token": fakeJwt
    }
  }
  var headers: HttpHeaders
  headers["Content-Type"] = "application/json"
  addCorsHeaders(headers)
  request.respond(200, headers, $response)

proc optionsHandler(request: Request) =
  var headers: HttpHeaders
  addCorsHeaders(headers)
  request.respond(200, headers, "")

when isMainModule:
  var router: Router
  router.get("/health", healthHandlerWrapper)
  router.get("/", indexHandler)
  router.post("/api/auth/login", loginHandler)
  router.options("/api/auth/login", optionsHandler)
  
  let server = newServer(router)
  echo "Server starting on port 8080"
  server.serve(Port(8080))