# Xanthus Self-Deployment Investigation

**Date**: 2025-07-07  
**Status**: In Progress  
**Goal**: Enable xanthus to deploy itself on managed VPS instances

## 🎯 Objective

Enable the xanthus platform to deploy itself as an application on managed VPS instances, allowing for self-hosting and testing scenarios.

## 📋 Current Status

### ✅ Completed Tasks

1. **Fixed Repository Configuration**
   - Changed from `repository: local` to `repository: https://github.com/chrishham/xanthus.git`
   - Updated chart path to `charts/xanthus-platform`
   - Fixed namespace from `xanthus-platform` to `xanthus`

2. **Fixed Template Processing**
   - Updated `generateFromTemplate()` in `internal/services/application_service_templates.go`
   - Added proper Go template resolution for placeholders like `{{.Version}}` → `"v1.0.7"`
   - Fixed domain/subdomain placeholder substitution

3. **Enhanced Version Lookup System**
   - Added `TypeXanthus` to application types in `config.go`
   - Added `VersionSource` field to `PredefinedApplication` model
   - Updated conversion function to include version source configuration
   - Made version handler generic to support any app with `version_source.type: github`

4. **Version Lookup Working**
   - Successfully fetches releases from GitHub: v1.0.7, v1.0.6, v1.0.5
   - API endpoint `/applications/versions/xanthus` returns proper release data

### ❌ Current Issues

1. **Version Parameter Not Respected**
   - API call with `"version": "v1.0.7"` still results in `"app_version": "latest"`
   - Deployment defaults to `latest` regardless of specified version

2. **Image Pull Still Failing**
   - Still attempting to pull `ghcr.io/chrishham/xanthus:latest`
   - Should be pulling `ghcr.io/chrishham/xanthus:v1.0.7`
   - Same 401 Unauthorized error persists

## 🔍 Investigation Results

### Deployment Flow Analysis

The xanthus deployment follows this flow:

1. **API Call**: `POST /applications/create` with version parameter
2. **Handler**: `HandleApplicationsCreate()` in `internal/handlers/applications/http.go:111`
3. **Service**: `CreateApplication()` in `internal/services/application_service_core.go:172`
4. **Deployment**: `deployApplication()` in `internal/services/application_service_deployment.go:16`
5. **Template**: `generateFromTemplate()` in `internal/services/application_service_templates.go:24`
6. **Helm**: Chart deployment with GitHub repository clone

### Current Deployment Attempt Results

**Command Used**:
```bash
curl -X POST "http://localhost:8081/applications/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "xanthus",
    "description": "Self-hosted infrastructure management platform", 
    "app_type": "xanthus",
    "subdomain": "xanthus",
    "domain": "myclasses.gr",
    "vps": "1751833610",
    "version": "v1.0.7"
  }'
```

**Observed Behavior**:
- ✅ Application created successfully
- ❌ Shows `"app_version": "latest"` instead of `"v1.0.7"`
- ❌ Pod status: `ImagePullBackOff`
- ❌ Image: `ghcr.io/chrishham/xanthus:latest` (should be `:v1.0.7`)

**Kubernetes Pod Details**:
```
NAME: xanthus-xanthus-xanthus-platform-0
STATUS: ImagePullBackOff
IMAGE: ghcr.io/chrishham/xanthus:latest
ERROR: failed to authorize: 401 Unauthorized
```

## 🔧 Root Cause Analysis

### Issue 1: Version Parameter Handling

**Problem**: The `"version"` parameter in the API request is not being processed correctly.

**Evidence**:
- API request includes `"version": "v1.0.7"`
- Database/KV storage shows `"app_version": "latest"`
- Template processing uses incorrect version

**Investigation Needed**:
1. Check how `HandleApplicationsCreate()` processes the version field
2. Verify version is passed to `CreateApplication()`
3. Confirm version is stored correctly in application model
4. Trace version flow through to template generation

### Issue 2: Template Placeholder Resolution

**Problem**: Helm template still resolves to `:latest` tag instead of specified version.

**Evidence**:
- Template processing was fixed for domain placeholders
- Version placeholder still not working correctly
- Pod attempts to pull `:latest` tag

**Investigation Needed**:
1. Check xanthus template file: `internal/templates/applications/xanthus.yaml`
2. Verify image tag placeholder: should be `{{APPLICATION_VERSION}}`
3. Confirm placeholder mapping in `configs/applications/xanthus.yaml`
4. Test template resolution with debug output

### Issue 3: Docker Image Availability

**Problem**: Even if version resolution works, need to verify image exists.

**Evidence**:
- 401 Unauthorized suggests authentication or missing image
- Image should be at `ghcr.io/chrishham/xanthus:v1.0.7`

**Investigation Needed**:
1. Verify image exists: `docker pull ghcr.io/chrishham/xanthus:v1.0.7`
2. Check image visibility (public vs private)
3. Confirm image tags match GitHub releases
4. Test manual pull from VPS

## 🚀 Next Steps

### Priority 1: Version Parameter Flow
- [ ] Debug `HandleApplicationsCreate()` - check if version field is parsed
- [ ] Trace version through `CreateApplication()` service call
- [ ] Verify version storage in application model
- [ ] Check if `predefinedApp.Version` is updated with user selection

### Priority 2: Template Investigation  
- [ ] Examine `internal/templates/applications/xanthus.yaml` image tag
- [ ] Verify placeholder configuration in `configs/applications/xanthus.yaml`
- [ ] Add debug logging to template processing
- [ ] Test template resolution with mock data

### Priority 3: Image Verification
- [ ] Check if `ghcr.io/chrishham/xanthus:v1.0.7` exists and is public
- [ ] Test manual Docker pull from VPS
- [ ] Verify release process creates correct image tags
- [ ] Confirm image registry authentication requirements

### Priority 4: End-to-End Testing
- [ ] Fix version parameter handling
- [ ] Redeploy with corrected version flow
- [ ] Monitor Kubernetes deployment progress
- [ ] Verify successful xanthus self-deployment

## 📁 Key Files to Investigate

### API/Handler Layer
- `internal/handlers/applications/http.go:111` - `HandleApplicationsCreate()`
- `internal/handlers/applications/config.go` - Application type definitions

### Service Layer  
- `internal/services/application_service_core.go:172` - `CreateApplication()`
- `internal/services/application_service_deployment.go:16` - `deployApplication()`
- `internal/services/application_service_templates.go:24` - `generateFromTemplate()`

### Configuration
- `configs/applications/xanthus.yaml` - Application configuration
- `internal/templates/applications/xanthus.yaml` - Helm values template
- `charts/xanthus-platform/values.yaml` - Default Helm values

### Models
- `internal/models/application.go` - Application data structures
- `internal/models/config.go` - Configuration loading

## 🎯 Success Criteria

1. **Version Selection Works**: API respects `"version": "v1.0.7"` parameter
2. **Template Resolution**: Helm template uses correct image tag `ghcr.io/chrishham/xanthus:v1.0.7`
3. **Image Pull Success**: Kubernetes successfully pulls versioned image
4. **Self-Deployment**: Xanthus successfully deploys itself on managed VPS
5. **Functional Deployment**: Deployed xanthus instance is accessible and working

## 📝 Notes

- Version lookup system works perfectly - can fetch GitHub releases
- Template processing for domain/subdomain works correctly  
- GitHub repository cloning and chart deployment logic is functional
- Issue is specifically in version parameter flow and image tag resolution
- Docker image availability needs verification independent of deployment testing

---

**Last Updated**: 2025-07-07 14:10 UTC  
**Next Review**: After Priority 1 investigation completion