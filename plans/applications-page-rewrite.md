  ⎿ Applications Page Rewrite Plan

    Overview

    Transform the current Helm-chart-based applications page into a predefined applications manager 
    where users select from curated apps (starting with Code Server) and deploy them to managed VPS 
    servers with automatic ingress configuration.

    Current State Analysis

    - Current Handler: Complex wizard-based Helm chart selection with repository management
    - Current Model: Generic Application model supporting any Helm chart
    - Current Templates: Sophisticated wizard UI for chart discovery and configuration
    - Existing Infrastructure: K3s clusters with ArgoCD, Traefik ingress, SSL certificates via domain 
    SSL configs

    Proposed Changes

    1. Create Predefined Application Catalog

    - New file: internal/models/applications.go - Define predefined applications
    - Structure: Static catalog with Code Server as first app
    - Configuration: Helm chart details, default values, required resources
    - Extensibility: Easy to add more applications later

    2. Redesign Application Model

    - Modify: internal/models/types.go - Update Application struct
    - Add fields: App type, predefined configuration, installation status
    - Remove fields: Chart name/version (will be handled internally)

    3. Rewrite Applications Handler

    - File: internal/handlers/applications.go
    - Simplify: Remove complex Helm repository management
    - Add: VPS selection and app type selection logic
    - Integrate: Code Server Helm installation using documented configuration
    - Enhance: Automatic ingress creation with Traefik
    - Maintain: Upgrade, refresh, and delete functionality

    4. Update Templates

    - File: web/templates/applications.html
    - Simplify: Replace complex wizard with app selection grid
    - UI Flow: 
      a. Show predefined apps catalog
      b. Select VPS server
      c. Configure subdomain/domain
      d. Deploy with automatic ingress

    5. Implement Code Server Integration

    - Chart: Use https://github.com/coder/code-server Helm chart
    - Configuration: Apply Code Server documentation settings
    - Ingress: Create Traefik ingress rules automatically
    - SSL: Use existing domain SSL certificate management
    - Namespace: Deploy to separate code-server namespace

    6. Enhance Ingress Management

    - Pattern: Follow existing ArgoCD ingress configuration approach
    - Traefik: Create ingress rules pointing to code-server.domain.com
    - SSL: Automatically apply domain certificates
    - Security: Implement proper ingress annotations

    7. Service Layer Updates

    - File: internal/services/helm.go
    - Add: Predefined application deployment methods
    - Enhance: Ingress creation functionality
    - Maintain: Existing Helm operations compatibility

    Implementation Steps

    Phase 1: Core Infrastructure

    1. Create predefined applications catalog
    2. Update Application model
    3. Add Code Server app definition

    Phase 2: Handler Rewrite

    1. Simplify applications handler methods
    2. Remove repository management complexity
    3. Add VPS + app selection logic
    4. Integrate Code Server deployment

    Phase 3: UI Transformation

    1. Replace wizard with app catalog grid
    2. Simplify deployment flow
    3. Add predefined app cards
    4. Maintain existing functionality (upgrade, delete)

    Phase 4: Ingress Integration

    1. Implement automatic Traefik ingress creation
    2. Apply SSL certificates from domain configuration
    3. Ensure proper namespace isolation
    4. Test code-server.domain.com accessibility

    Phase 5: Testing & Polish

    1. Test Code Server deployment end-to-end
    2. Verify ingress and SSL functionality
    3. Ensure upgrade/delete operations work
    4. Add proper error handling

    Benefits

    - Simplified UX: No more complex Helm repository management
    - Curated Experience: Pre-tested, working applications
    - Automatic Ingress: No manual ingress configuration required
    - Extensible: Easy to add new predefined applications
    - Consistent: Follows existing Xanthus patterns and infrastructure
