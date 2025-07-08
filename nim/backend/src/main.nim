import jester, asyncdispatch, json, os, strutils
import handlers/[health]
import middleware/[cors_middleware]

proc setupRoutes(app: Jester) =
  # CORS middleware
  app.all "*", corsMiddleware
  
  # Health check endpoint
  app.get "/health", healthHandler

when isMainModule:
  let port = if existsEnv("PORT"): parseInt(getEnv("PORT")) else: 8080
  let settings = newSettings(port=Port(port))
  
  echo "Starting Xanthus Nim backend on port ", port
  
  var jester = initJester(settings)
  jester.setupRoutes()
  runForever()