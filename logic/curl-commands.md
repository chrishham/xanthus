# Xanthus API Curl Commands

This document contains useful curl commands for interacting with the Xanthus application API.

## Authentication

The application uses Cloudflare API token for authentication. You can get the Cloudfare_Api_Token from the .env file.

### Login to the Application
Use `CLOUDFARE_API_TOKEN` from `.env`

```bash
curl -X POST "http://localhost:8081/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "cf_token=$CLOUDFARE_API_TOKEN" \
  -c cookies.txt
```

### Use Session Cookie for Subsequent Requests
```bash
curl -X GET "http://localhost:8081/applications" \
  -b cookies.txt
```

## Base URL
```bash
BASE_URL="http://localhost:8081"
```

## Common Commands

### 1. Check Application Status
```bash
curl -X GET "$BASE_URL/" \
  -H "Authorization: Bearer $CLOUDFARE_API_TOKEN"
```

### 2. List Applications
```bash
curl -X GET "$BASE_URL/applications" \
  -H "Authorization: Bearer $CLOUDFARE_API_TOKEN"
```

### 3. Deploy New Application
```bash
curl -X POST "$BASE_URL/applications/create" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -b cookies.txt \
  -d "subdomain=test-app" \
  -d "domain=myclasses.gr" \
  -d "vps_id=66731639" \
  -d "app_type=code-server" \
  -d "description=Test deployment"
```

### 4. Deploy ArgoCD Application (JSON)
```bash
curl -X POST "$BASE_URL/applications/create" \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "name": "argocd2",
    "subdomain": "argocd2", 
    "domain": "myclasses.gr",
    "vps": "66731639",
    "app_type": "argocd",
    "description": "ArgoCD GitOps deployment"
  }'
```

### 4. List VPS Instances
```bash
curl -X GET "$BASE_URL/vps" \
  -b cookies.txt
```

### 5. Get Application Details
```bash
curl -X GET "$BASE_URL/applications/APPLICATION_ID" \
  -b cookies.txt
```

## VPS Investigation Commands

### Get VPS Details for SSH Connection
```bash
# Get all VPS instances with IP addresses and SSH details
curl -X GET "$BASE_URL/vps" \
  -b cookies.txt \
  | jq '.[] | {id, name, ip_address, provider, ssh_user}'

# Get specific VPS details
curl -X GET "$BASE_URL/vps/VPS_ID" \
  -b cookies.txt \
  | jq '{id, name, ip_address, provider, ssh_user}'
```

### SSH Connection Based on VPS Type
After getting VPS details from API:

**For Hetzner VPS (SSH user: root):**
```bash
ssh -i xanthus-key.pem root@{ip_address}
```

**For Oracle VPS (SSH user: ubuntu):**
```bash
ssh -i xanthus-key.pem ubuntu@{ip_address}
```

### Example Investigation Workflow
```bash
# 1. Login and get session cookie
curl -X POST "$BASE_URL/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "cf_token=$CLOUDFARE_API_TOKEN" \
  -c cookies.txt

# 2. Get VPS list to find target VPS
curl -X GET "$BASE_URL/vps" -b cookies.txt

# 3. Get specific VPS details (replace VPS_ID with actual ID)
VPS_INFO=$(curl -s -X GET "$BASE_URL/vps/VPS_ID" -b cookies.txt)
IP_ADDRESS=$(echo $VPS_INFO | jq -r '.ip_address')
PROVIDER=$(echo $VPS_INFO | jq -r '.provider')

# 4. SSH with correct user based on provider
if [ "$PROVIDER" = "oracle" ]; then
  ssh -i xanthus-key.pem ubuntu@$IP_ADDRESS
else
  ssh -i xanthus-key.pem root@$IP_ADDRESS
fi
```

## Current Deployments

### ArgoCD Deployment (Successfully deployed)
- **URL**: https://argocd3.myclasses.gr
- **Username**: admin  
- **Password**: Ne2EaJ7tTYQH9GJk
- **Application ID**: app-1751723395
- **Status**: Running
- **VPS**: xanthus-k3s-1751719134798 (66731639)

## Notes
- Replace `YOUR_VPS_ID` with actual VPS ID from the VPS list (currently: 66731639)
- Replace `APPLICATION_ID` with actual application ID
- All requests require session cookie (use `-b cookies.txt`)
- POST requests to `/applications/create` expect JSON content type
- Form data should be URL-encoded for other POST requests