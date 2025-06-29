## VPS

@web/templates/vps-manage.html
ram usage output is problematic
ArgoCD: View Credentials  delete

- when a vps is deleted , then delete all associated entries from applications.
- when a vps is created...
 
VPS Status & Health
k3s status doesnt update succesfully, even after Setup completed! All components are ready.
 K3s Service: ❌ unknown 
@web/templates/vps-create.html
- at vps creation ensure that login without ssh is completely disabled
- at create a vps, list and the dedicated instances and add an appropriate filter
- choose server type: add option to filter out unavailable
- install filebrowser https://github.com/gtsteffaniak/filebrowser
headlmamp, openwebui, code-server

## Applications
@web/templates/applications.html

- fix Not Deployed at applications list, when the app is surely deployed.

# Settings

- separate page to update the Hetzner api key ?

## Port forward from vps to local machine


# Tests

no e2e all the others

# Initial setup for the user

buy domain, point nameservers to cloudfare, wait for the domain to be active, create account at Hetzner and create api ket
 and then use xanthus

# Others

- Clarify the minimum permissions required for cloudfare api key to work.
- What happens if the user forgets or revokes the cloudfare api key?

## Essential apps

- 

 find internal -type f -name '*.go' -exec wc -l {} + | sort -nr | sed -n '2,11p'
  1389 internal/handlers/applications.go
  1282 internal/handlers/vps.go
   781 internal/services/ssh.go
   647 internal/services/cloudflare.go
   472 internal/services/hetzner.go
   427 internal/services/kv.go
   317 internal/utils/cloudflare.go
   291 internal/utils/hetzner.go
   274 internal/handlers/dns.go
   257 internal/services/helm.go

   GetPredefinedApplications