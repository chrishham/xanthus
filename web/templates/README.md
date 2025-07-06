# Web Templates Architecture

## 📋 Purpose
HTMX-based dynamic UI templates with Alpine.js reactivity for server-side rendering without complex JavaScript.

## 🏗️ Architecture

### Template Organization
```
web/templates/
├── main.html           # Main layout template
├── navbar.html         # Navigation component
├── applications.html   # Application management UI
├── vps-manage.html     # VPS management UI
├── terminal.html       # Web terminal interface
├── login.html          # Authentication UI
└── partials/           # Reusable components
    ├── applications/   # App-specific partials
    ├── vps/           # VPS-specific partials
    ├── common/        # Shared UI components
    └── wizard/        # Multi-step workflows
```

### Technology Stack
```
HTML Templates + HTMX + Alpine.js + Tailwind CSS
```

## 🔧 Key Templates

### Main Layout (`main.html`)
```html
<!DOCTYPE html>
<html lang="en">
<head>
    {{template "partials/common/head.html" .}}
    <title>{{.Title}} - Xanthus</title>
</head>
<body class="bg-gray-100">
    {{template "navbar.html" .}}
    
    <main class="container mx-auto px-4 py-8">
        {{template "content" .}}
    </main>
    
    {{template "partials/common/script-loader.html" .}}
    {{template "partials/common/loading-modal.html" .}}
</body>
</html>
```

### Applications Page (`applications.html`)
```html
{{define "content"}}
<div class="space-y-6" x-data="applicationManager">
    <!-- Page Header -->
    {{template "partials/common/page-header.html" 
        dict "title" "Applications" 
             "subtitle" "Manage your deployed applications"}}
    
    <!-- Auto-refresh Controls -->
    <div class="flex items-center justify-between">
        <button @click="toggleAutoRefresh()" 
                class="btn btn-secondary">
            <span x-show="autoRefreshEnabled">Auto-refresh ON</span>
            <span x-show="!autoRefreshEnabled">Auto-refresh OFF</span>
        </button>
        
        <div x-show="autoRefreshEnabled" class="flex items-center space-x-2">
            <div class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
            <span class="text-sm text-gray-600">
                Next: <span x-text="countdown"></span>s
            </span>
        </div>
    </div>
    
    <!-- Application Grid -->
    <div id="applications-grid" 
         class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {{range .Applications}}
            {{template "partials/applications/deployed-app-card.html" .}}
        {{end}}
    </div>
    
    <!-- Empty State -->
    {{if not .Applications}}
        {{template "partials/common/empty-state.html" 
            dict "title" "No applications deployed"
                 "description" "Create your first application to get started"
                 "actionText" "Deploy Application"
                 "actionURL" "/applications/catalog"}}
    {{end}}
</div>
{{end}}
```

### VPS Management (`vps-manage.html`)
```html
{{define "content"}}
<div class="space-y-6" x-data="vpsManager">
    <!-- VPS Statistics -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div class="bg-white rounded-lg shadow p-6">
            <h3 class="text-lg font-semibold text-gray-900">Total VPS</h3>
            <p class="text-3xl font-bold text-blue-600">{{.Stats.Total}}</p>
        </div>
        <div class="bg-white rounded-lg shadow p-6">
            <h3 class="text-lg font-semibold text-gray-900">Running</h3>
            <p class="text-3xl font-bold text-green-600">{{.Stats.Running}}</p>
        </div>
        <div class="bg-white rounded-lg shadow p-6">
            <h3 class="text-lg font-semibold text-gray-900">Stopped</h3>
            <p class="text-3xl font-bold text-red-600">{{.Stats.Stopped}}</p>
        </div>
        <div class="bg-white rounded-lg shadow p-6">
            <h3 class="text-lg font-semibold text-gray-900">Applications</h3>
            <p class="text-3xl font-bold text-purple-600">{{.Stats.Applications}}</p>
        </div>
    </div>
    
    <!-- VPS List -->
    <div class="bg-white rounded-lg shadow">
        <div class="p-6 border-b">
            <h2 class="text-xl font-semibold">VPS Instances</h2>
        </div>
        
        <div class="divide-y divide-gray-200">
            {{range .VPSList}}
                {{template "partials/vps/server-card.html" .}}
            {{end}}
        </div>
    </div>
</div>
{{end}}
```

## 📊 Partial Templates

### Application Card (`partials/applications/deployed-app-card.html`)
```html
<div class="bg-white rounded-lg shadow-md hover:shadow-lg transition-shadow">
    <div class="p-6">
        <!-- App Header -->
        <div class="flex items-start justify-between">
            <div>
                <h3 class="text-xl font-semibold text-gray-900">{{.Name}}</h3>
                <p class="text-sm text-gray-600">{{.Type}}</p>
            </div>
            
            <!-- Status Badge -->
            <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
                        {{if eq .Status "deployed"}}bg-green-100 text-green-800
                        {{else if eq .Status "deploying"}}bg-blue-100 text-blue-800
                        {{else if eq .Status "failed"}}bg-red-100 text-red-800
                        {{else}}bg-gray-100 text-gray-800{{end}}">
                {{.Status}}
            </span>
        </div>
        
        <!-- App Details -->
        <div class="mt-4 space-y-2">
            <div class="flex items-center text-sm text-gray-600">
                <span class="font-medium">URL:</span>
                <a href="{{.URL}}" target="_blank" 
                   class="ml-2 text-blue-600 hover:text-blue-800">
                    {{.URL}}
                </a>
            </div>
            
            <div class="flex items-center text-sm text-gray-600">
                <span class="font-medium">VPS:</span>
                <span class="ml-2">{{.VPSName}}</span>
            </div>
            
            <div class="flex items-center text-sm text-gray-600">
                <span class="font-medium">Created:</span>
                <span class="ml-2">{{.CreatedAt.Format "Jan 2, 2006"}}</span>
            </div>
        </div>
        
        <!-- Actions -->
        <div class="mt-6 flex space-x-3">
            <a href="{{.URL}}" target="_blank" 
               class="btn btn-primary btn-sm">
                Open
            </a>
            
            <button hx-get="/applications/{{.ID}}/password" 
                    hx-target="#password-modal" 
                    hx-swap="innerHTML"
                    class="btn btn-secondary btn-sm">
                Get Password
            </button>
            
            <button hx-post="/applications/{{.ID}}/restart" 
                    hx-confirm="Are you sure you want to restart this application?"
                    class="btn btn-warning btn-sm">
                Restart
            </button>
            
            <button hx-delete="/applications/{{.ID}}" 
                    hx-confirm="Are you sure you want to delete this application?"
                    hx-target="closest .bg-white"
                    hx-swap="outerHTML"
                    class="btn btn-danger btn-sm">
                Delete
            </button>
        </div>
    </div>
</div>
```

### VPS Server Card (`partials/vps/server-card.html`)
```html
<div class="p-6 hover:bg-gray-50 transition-colors">
    <div class="flex items-center justify-between">
        <div class="flex items-center space-x-4">
            <!-- Provider Icon -->
            <div class="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center">
                {{if eq .Provider "hetzner"}}
                    <span class="text-blue-600 font-bold">H</span>
                {{else if eq .Provider "oracle"}}
                    <span class="text-red-600 font-bold">O</span>
                {{else}}
                    <span class="text-gray-600 font-bold">?</span>
                {{end}}
            </div>
            
            <!-- Server Details -->
            <div>
                <h3 class="text-lg font-semibold text-gray-900">{{.Name}}</h3>
                <p class="text-sm text-gray-600">{{.IPAddress}} • {{.Location}}</p>
                <p class="text-sm text-gray-600">{{.Size}} • {{.Provider}}</p>
            </div>
        </div>
        
        <!-- Status and Actions -->
        <div class="flex items-center space-x-4">
            <!-- Status -->
            <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
                        {{if eq .Status "running"}}bg-green-100 text-green-800
                        {{else if eq .Status "stopped"}}bg-red-100 text-red-800
                        {{else}}bg-yellow-100 text-yellow-800{{end}}">
                {{.Status}}
            </span>
            
            <!-- Actions -->
            <div class="flex space-x-2">
                {{if eq .Status "running"}}
                    <button hx-post="/vps/{{.ID}}/stop" 
                            class="btn btn-sm btn-warning">
                        Stop
                    </button>
                {{else}}
                    <button hx-post="/vps/{{.ID}}/start" 
                            class="btn btn-sm btn-success">
                        Start
                    </button>
                {{end}}
                
                <button hx-get="/vps/{{.ID}}/details" 
                        hx-target="#vps-details-modal" 
                        hx-swap="innerHTML"
                        class="btn btn-sm btn-secondary">
                    Details
                </button>
                
                <button hx-delete="/vps/{{.ID}}" 
                        hx-confirm="Are you sure you want to delete this VPS?"
                        class="btn btn-sm btn-danger">
                    Delete
                </button>
            </div>
        </div>
    </div>
</div>
```

## 🔧 HTMX Integration

### Dynamic Content Loading
```html
<!-- Load partial content -->
<div hx-get="/applications/list" 
     hx-trigger="load, every 30s" 
     hx-swap="innerHTML">
    Loading applications...
</div>

<!-- Form submission -->
<form hx-post="/applications/create" 
      hx-target="#applications-grid" 
      hx-swap="afterbegin">
    <!-- Form fields -->
</form>

<!-- Confirm dialogs -->
<button hx-delete="/applications/{{.ID}}" 
        hx-confirm="Are you sure you want to delete this application?"
        hx-target="closest .application-card"
        hx-swap="outerHTML">
    Delete
</button>
```

### Auto-refresh System
```html
<!-- Auto-refresh container -->
<div id="auto-refresh-container" 
     x-data="autoRefreshManager"
     x-init="initAutoRefresh()">
    
    <!-- Refresh controls -->
    <div class="flex items-center space-x-4">
        <button @click="toggleAutoRefresh()" 
                class="btn btn-secondary">
            <span x-show="enabled">Auto-refresh ON</span>
            <span x-show="!enabled">Auto-refresh OFF</span>
        </button>
        
        <div x-show="enabled" class="flex items-center space-x-2">
            <div class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
            <span class="text-sm text-gray-600">
                Next: <span x-text="countdown"></span>s
            </span>
        </div>
    </div>
    
    <!-- Content that gets refreshed -->
    <div hx-get="/applications/refresh" 
         hx-trigger="refreshApplications from:body"
         hx-swap="innerHTML">
        {{template "application-list" .}}
    </div>
</div>
```

## 🎯 Alpine.js Components

### Application Manager
```javascript
// Defined in web/static/js/modules/applications-management.js
Alpine.data('applicationManager', () => ({
    autoRefreshEnabled: true,
    countdown: 30,
    refreshTimer: null,
    
    init() {
        this.startAutoRefresh();
    },
    
    toggleAutoRefresh() {
        this.autoRefreshEnabled = !this.autoRefreshEnabled;
        
        if (this.autoRefreshEnabled) {
            this.startAutoRefresh();
        } else {
            this.stopAutoRefresh();
        }
    },
    
    startAutoRefresh() {
        this.refreshTimer = setInterval(() => {
            if (this.countdown > 0) {
                this.countdown--;
            } else {
                this.refreshApplications();
                this.countdown = 30;
            }
        }, 1000);
    },
    
    refreshApplications() {
        htmx.trigger(document.body, 'refreshApplications');
    }
}));
```

### VPS Manager
```javascript
// VPS management component
Alpine.data('vpsManager', () => ({
    selectedVPS: null,
    showDetails: false,
    
    selectVPS(vpsId) {
        this.selectedVPS = vpsId;
        this.showDetails = true;
    },
    
    closeDetails() {
        this.selectedVPS = null;
        this.showDetails = false;
    }
}));
```

## 📊 Common Patterns

### Form Handling
```html
<!-- Application creation form -->
<form hx-post="/applications/create" 
      hx-target="#applications-grid" 
      hx-swap="afterbegin"
      hx-on::after-request="this.reset()">
    
    <div class="space-y-4">
        <div>
            <label class="block text-sm font-medium text-gray-700">Name</label>
            <input type="text" name="name" required 
                   class="mt-1 block w-full rounded-md border-gray-300">
        </div>
        
        <div>
            <label class="block text-sm font-medium text-gray-700">Type</label>
            <select name="app_type" required 
                    class="mt-1 block w-full rounded-md border-gray-300">
                <option value="">Select application type</option>
                {{range .AppTypes}}
                    <option value="{{.ID}}">{{.Name}}</option>
                {{end}}
            </select>
        </div>
        
        <button type="submit" class="btn btn-primary">
            Deploy Application
        </button>
    </div>
</form>
```

### Modal Integration
```html
<!-- Modal trigger -->
<button hx-get="/applications/{{.ID}}/password" 
        hx-target="#password-modal" 
        hx-swap="innerHTML"
        class="btn btn-secondary">
    Get Password
</button>

<!-- Modal container -->
<div id="password-modal" class="modal"></div>

<!-- Modal content (returned by server) -->
<div class="modal-content">
    <div class="modal-header">
        <h3>Application Password</h3>
        <button class="modal-close">&times;</button>
    </div>
    <div class="modal-body">
        <p>Your application password is:</p>
        <code class="bg-gray-100 px-2 py-1 rounded">{{.Password}}</code>
    </div>
</div>
```

## 🎨 Styling System

### Tailwind CSS Classes
```html
<!-- Common button styles -->
<button class="btn btn-primary">Primary Action</button>
<button class="btn btn-secondary">Secondary Action</button>
<button class="btn btn-danger">Danger Action</button>

<!-- Status badges -->
<span class="badge badge-success">Running</span>
<span class="badge badge-warning">Pending</span>
<span class="badge badge-error">Failed</span>

<!-- Cards and containers -->
<div class="card">
    <div class="card-header">
        <h3 class="card-title">Title</h3>
    </div>
    <div class="card-body">
        Content
    </div>
</div>
```

### Custom CSS Components
```css
/* Defined in web/static/css/components.css */
.btn {
    @apply inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm;
}

.btn-primary {
    @apply text-white bg-blue-600 hover:bg-blue-700 focus:ring-blue-500;
}

.card {
    @apply bg-white overflow-hidden shadow rounded-lg;
}

.modal {
    @apply fixed inset-0 z-50 overflow-y-auto;
}
```

## 🔧 Template Functions

### Custom Template Functions
```go
// Defined in handler initialization
funcMap := template.FuncMap{
    "formatDate": func(t time.Time) string {
        return t.Format("Jan 2, 2006 15:04")
    },
    "statusColor": func(status string) string {
        switch status {
        case "deployed", "running":
            return "green"
        case "deploying", "creating":
            return "blue"
        case "failed":
            return "red"
        default:
            return "gray"
        }
    },
    "dict": func(values ...interface{}) map[string]interface{} {
        dict := make(map[string]interface{})
        for i := 0; i < len(values); i += 2 {
            dict[values[i].(string)] = values[i+1]
        }
        return dict
    },
}
```

### Template Usage
```html
<!-- Format dates -->
<span>Created: {{formatDate .CreatedAt}}</span>

<!-- Dynamic styling -->
<span class="badge badge-{{statusColor .Status}}">{{.Status}}</span>

<!-- Pass data to partials -->
{{template "partials/common/empty-state.html" 
    dict "title" "No applications" 
         "description" "Get started by creating your first application"
         "actionText" "Create Application"
         "actionURL" "/applications/create"}}
```

## 🚀 Performance Optimizations

### Template Caching
```go
// Templates parsed once at startup
templates := template.Must(template.ParseGlob("web/templates/*.html"))
templates = template.Must(templates.ParseGlob("web/templates/partials/*/*.html"))

// Cached template rendering
func renderTemplate(c *gin.Context, name string, data interface{}) {
    c.HTML(http.StatusOK, name, data)
}
```

### Lazy Loading
```html
<!-- Load content on demand -->
<div hx-get="/applications/details/{{.ID}}" 
     hx-trigger="intersect once" 
     hx-swap="innerHTML">
    <div class="animate-pulse">Loading...</div>
</div>
```

### Progressive Enhancement
```html
<!-- Works without JavaScript -->
<form action="/applications/create" method="POST">
    <!-- Form fields -->
    <button type="submit" class="btn btn-primary">Create</button>
</form>

<!-- Enhanced with HTMX -->
<form hx-post="/applications/create" 
      hx-target="#applications-grid" 
      hx-swap="afterbegin">
    <!-- Same form fields -->
    <button type="submit" class="btn btn-primary">Create</button>
</form>
```

## 🔒 Security Considerations

### CSRF Protection
```html
<!-- CSRF token in forms -->
<form hx-post="/applications/create">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
    <!-- Form fields -->
</form>
```

### Input Validation
```html
<!-- Client-side validation -->
<input type="text" name="subdomain" 
       required 
       pattern="[a-z0-9-]+" 
       title="Only lowercase letters, numbers, and hyphens allowed">

<!-- Server-side validation handled by handlers -->
```

### Content Security Policy
```html
<!-- CSP headers set by server -->
<meta http-equiv="Content-Security-Policy" 
      content="default-src 'self'; script-src 'self' 'unsafe-inline';">
```

## 🛠️ Development Workflow

### Template Development
1. **Edit templates** in `web/templates/`
2. **Restart server** to reload templates (development mode)
3. **Test in browser** with live reload
4. **Add new partials** as needed

### Adding New Pages
1. Create main template file
2. Add route in `internal/router/routes.go`
3. Add handler in appropriate domain
4. Create necessary partials
5. Update navigation if needed

### Component Development
1. Create partial template
2. Add Alpine.js component if needed
3. Style with Tailwind classes
4. Add HTMX interactions
5. Test responsive design