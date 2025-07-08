# Puppeteer Testing Commands for Xanthus

## Prerequisites
- Development server running on `http://localhost:8081`
- Cloudflare API token available in `.env` file

## Basic Navigation and Login Flow

### 1. Navigate to Application
```javascript
mcp__puppeteer__puppeteer_navigate("http://localhost:8081")
```

### 2. Take Screenshot of Login Page
```javascript
mcp__puppeteer__puppeteer_screenshot("homepage")
```

### 3. Fill Login Form
```javascript
// Fill the Cloudflare API token input
mcp__puppeteer__puppeteer_fill("input[placeholder=\"Enter your Cloudflare API token\"]", "YOUR_API_TOKEN_HERE")
```

### 4. Submit Login Form
```javascript
// Click the login button
mcp__puppeteer__puppeteer_click("button[type=\"submit\"]")
```

### 5. Verify Login Success and Navigate to Dashboard
```javascript
// Take screenshot to verify login notification
mcp__puppeteer__puppeteer_screenshot("post-login")

// Navigate to dashboard
mcp__puppeteer__puppeteer_navigate("http://localhost:8081/dashboard")
```

## Common Selector Patterns

### Working Selectors
- `input[placeholder="Enter your Cloudflare API token"]` - Login input field
- `button[type="submit"]` - Login button
- `nav a[href="/dashboard"]` - Navigation links

### Selectors to Avoid
- `button:contains('Login')` - CSS :contains() pseudo-selector not supported in querySelector

## Navigation Routes to Test

### Main Application Routes
- `/` - Login page
- `/dashboard` - Main dashboard
- `/dns` - DNS configuration
- `/vps` - VPS management
- `/applications` - Application catalog
- `/version` - Version information
- `/about` - About page

## Complete Test Session Example

```javascript
// 1. Start at login page
mcp__puppeteer__puppeteer_navigate("http://localhost:8081")
mcp__puppeteer__puppeteer_screenshot("01-login-page")

// 2. Perform login
mcp__puppeteer__puppeteer_fill("input[placeholder=\"Enter your Cloudflare API token\"]", "oqt2byqtVfOL2LhWvEyRaOsOV8cXE6QwUGTIJiHb")
mcp__puppeteer__puppeteer_click("button[type=\"submit\"]")
mcp__puppeteer__puppeteer_screenshot("02-login-success")

// 3. Navigate to dashboard
mcp__puppeteer__puppeteer_navigate("http://localhost:8081/dashboard")
mcp__puppeteer__puppeteer_screenshot("03-dashboard")

// 4. Test other routes
mcp__puppeteer__puppeteer_navigate("http://localhost:8081/applications")
mcp__puppeteer__puppeteer_screenshot("04-applications")

mcp__puppeteer__puppeteer_navigate("http://localhost:8081/vps")
mcp__puppeteer__puppeteer_screenshot("05-vps")

mcp__puppeteer__puppeteer_navigate("http://localhost:8081/dns")
mcp__puppeteer__puppeteer_screenshot("06-dns")
```

## Error Handling Notes

### Common Issues
1. **Invalid Selectors**: Use standard CSS selectors, avoid pseudo-selectors like :contains()
2. **Timing Issues**: Wait for page loads before taking screenshots
3. **Authentication**: Ensure valid Cloudflare API token is used

### Debugging Tips
- Take screenshots at each step to verify page state
- Use browser DevTools to inspect elements for correct selectors
- Check network tab for failed requests
- Verify console for JavaScript errors

## Environment Setup

### Required Files
- `.env` file with `CLOUDFARE_API_TOKEN=your_token_here`
- Development server running (`make dev-full`)

### Dependencies
- Puppeteer MCP server configured
- Claude Code with Puppeteer tools enabled