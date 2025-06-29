# HTML Template Refactoring Plan

## Overview

This document outlines the refactoring strategy for the large HTML templates in the Xanthus web application. The current templates are too large and contain embedded JavaScript, making them difficult to maintain and violating the 500-line limit guideline.

## Current State Analysis

### Template Sizes
- **vps-manage.html**: 1,548 lines
- **applications.html**: 1,094 lines  
- **vps-create.html**: 850 lines
- **dns-config.html**: 423 lines

### Key Issues
1. **Massive JavaScript blocks** embedded in templates (~900 lines in vps-manage.html)
2. **Repeated UI patterns** without reusable components
3. **Complex utility functions** mixed with presentation logic
4. **Monolithic template structure** making maintenance difficult

## Refactoring Strategy

### Phase 1: JavaScript Extraction

#### 1.1 Create External JavaScript Modules
```
web/static/js/
├── modules/
│   ├── vps-management.js       # Extract from vps-manage.html
│   ├── applications-management.js  # Extract from applications.html
│   ├── vps-creation-wizard.js  # Extract from vps-create.html
│   └── common/
│       ├── alpine-components.js    # Shared Alpine.js components
│       ├── api-client.js          # API interaction utilities
│       ├── formatting-utils.js    # Memory/disk table formatters
│       └── sweet-alert-helpers.js # Reusable SweetAlert2 configs
```

#### 1.2 JavaScript Module Structure
Each module should export Alpine.js data functions:
```javascript
// Example: vps-management.js
export function vpsManagement() {
    return {
        // Alpine.js component data and methods
        servers: [],
        loading: false,
        // ... rest of component logic
    }
}
```

### Phase 2: Partial Templates Creation

#### 2.1 Common Components
```
web/templates/partials/
├── common/
│   ├── loading-modal.html      # Reusable loading overlay
│   ├── action-buttons.html     # Common button patterns
│   ├── page-header.html        # Standard page headers
│   └── empty-state.html        # No data states
├── vps/
│   ├── server-card.html        # Individual server display
│   ├── server-actions.html     # Server action buttons
│   └── server-details.html     # Server information display
├── applications/
│   ├── app-card.html           # Application card component
│   ├── app-catalog.html        # Available apps section
│   ├── app-actions.html        # Application action buttons
│   └── confirmation-modals.html # Delete/change confirmations
└── wizard/
    ├── progress-steps.html     # Multi-step wizard progress
    ├── domain-selection.html  # VPS creation step 1
    ├── api-key-setup.html     # VPS creation step 2
    ├── location-selection.html # VPS creation step 3
    ├── server-type-selection.html # VPS creation step 4
    └── review-create.html     # VPS creation step 5
```

#### 2.2 Template Composition Pattern
```html
<!-- Example: Refactored vps-manage.html -->
<!DOCTYPE html>
<html lang="en">
{{template "partials/common/head.html" .}}
<body class="bg-gray-100 min-h-screen">
    {{template "navbar.html" .}}
    
    <div x-data="vpsManagement()" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {{template "partials/common/loading-modal.html" .}}
        {{template "partials/common/page-header.html" .}}
        {{template "partials/vps/action-buttons.html" .}}
        {{template "partials/common/empty-state.html" .}}
        
        <div x-show="servers.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <template x-for="server in servers" :key="server.id">
                {{template "partials/vps/server-card.html" .}}
            </template>
        </div>
    </div>
    
    <script type="module">
        import { vpsManagement } from '/static/js/modules/vps-management.js';
        window.vpsManagement = vpsManagement;
    </script>
</body>
</html>
```

### Phase 3: CSS Optimization

#### 3.1 Utility Classes
Create utility classes for common patterns:
```css
/* web/static/css/components.css */
.card-base {
    @apply bg-white rounded-lg shadow-md border hover:shadow-lg transition-shadow;
}

.button-primary {
    @apply inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500;
}

.button-secondary {
    @apply inline-flex items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500;
}

.status-badge {
    @apply inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium;
}

.status-running { @apply bg-green-100 text-green-800; }
.status-deploying { @apply bg-blue-100 text-blue-800; }
.status-pending { @apply bg-yellow-100 text-yellow-800; }
.status-failed { @apply bg-red-100 text-red-800; }
.status-not-deployed { @apply bg-gray-100 text-gray-800; }
```

### Phase 4: Template Refactoring Implementation

#### 4.1 vps-manage.html Refactoring
**Target size**: ~200 lines (87% reduction)

**Structure**:
```html
<!-- Main template with includes -->
{{template "partials/common/head.html" .}}
{{template "partials/vps/page-header.html" .}}
{{template "partials/vps/action-section.html" .}}
{{template "partials/vps/servers-grid.html" .}}
{{template "partials/common/script-loader.html" "vps-management"}}
```

#### 4.2 applications.html Refactoring  
**Target size**: ~150 lines (86% reduction)

**Structure**:
```html
<!-- Main template with includes -->
{{template "partials/common/head.html" .}}
{{template "partials/applications/page-header.html" .}}
{{template "partials/applications/catalog-section.html" .}}
{{template "partials/applications/deployed-section.html" .}}
{{template "partials/common/script-loader.html" "applications-management"}}
```

#### 4.3 vps-create.html Refactoring
**Target size**: ~100 lines (88% reduction)

**Structure**:
```html
<!-- Main template with wizard steps -->
{{template "partials/common/head.html" .}}
{{template "partials/wizard/page-header.html" .}}
{{template "partials/wizard/progress-steps.html" .}}
{{template "partials/wizard/step-content.html" .}}
{{template "partials/common/script-loader.html" "vps-creation-wizard"}}
```

## Implementation Steps

### Step 1: Prepare Infrastructure
1. Create directory structure for modules and partials
2. Set up build process for JavaScript modules
3. Update Makefile to include new asset compilation

### Step 2: Extract JavaScript (Week 1)
1. Extract vps-management.js from vps-manage.html
2. Extract applications-management.js from applications.html  
3. Extract vps-creation-wizard.js from vps-create.html
4. Create common utility modules
5. Update templates to use external scripts

### Step 3: Create Partial Templates (Week 2)
1. Identify and extract repeated HTML patterns
2. Create partial templates for each component
3. Test partial template rendering
4. Create utility CSS classes

### Step 4: Refactor Main Templates (Week 3)
1. Refactor vps-manage.html to use partials
2. Refactor applications.html to use partials
3. Refactor vps-create.html to use partials
4. Update any remaining templates

### Step 5: Testing and Optimization (Week 4)
1. Test all templates for functionality
2. Optimize JavaScript module loading
3. Validate CSS utility classes
4. Performance testing and optimization

## Expected Outcomes

### Template Size Reduction
- **vps-manage.html**: 1,548 → ~200 lines (87% reduction)
- **applications.html**: 1,094 → ~150 lines (86% reduction)
- **vps-create.html**: 850 → ~100 lines (88% reduction)

### Maintainability Improvements
- **Reusable components** reduce code duplication
- **External JavaScript** enables proper IDE support and debugging
- **Modular structure** makes updates and bug fixes easier
- **Consistent styling** through utility classes

### Performance Benefits
- **Cacheable JavaScript modules** improve loading times
- **Smaller HTML payloads** reduce initial page load
- **Better browser caching** for static assets

## Risk Mitigation

### Backward Compatibility
- Maintain existing API endpoints and data structures
- Ensure Alpine.js component interfaces remain unchanged
- Test all user interactions thoroughly

### Rollback Strategy
- Keep original templates as `.bak` files until refactoring is stable
- Use feature flags to switch between old and new templates
- Implement gradual rollout starting with least critical pages

## Success Metrics

1. **All templates under 500 lines** (strict requirement)
2. **JavaScript successfully externalized** (measurable via file sizes)
3. **No functionality regression** (validated through testing)
4. **Improved page load performance** (measured via browser dev tools)
5. **Reduced code duplication** (measured via duplicate code analysis)

## Next Steps

1. **Approval**: Review and approve this refactoring plan
2. **Timeline**: Assign timeline and resources for 4-week implementation
3. **Backup**: Create backup branch before starting refactoring
4. **Implementation**: Begin with Step 1 infrastructure preparation

---

*This refactoring will significantly improve code maintainability while adhering to the 500-line limit guideline and modern web development best practices.*