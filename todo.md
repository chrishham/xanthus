## VPS
@web/templates/vps-manage.html
ram usage output is problematic
ArgoCD: View Credentials  delete

- when a vps is deleted , then delete all associated entries from applications.

- when user presses create vps then for a split second some error message appears at the ui before the wizard takes over.
- when a vps is created then show a loading modal , show that user cant press anything else
- When a vps is created then dont check for available dns records created by Xanthus and dont touch A records.
- Dont touch A records when creating a vps. when user creates an application then for the domain associated (or blank or asterisk for the bare domain) only then create an A record. 
- When user creates a new app check if the subdomain is arleady taken show relevant error.



 
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



 The current codebase is well-structured with good separation of concerns. The
  applications-management.js file could be refactored into smaller modules, but that's a
  refactoring issue, not an architecture problem.

basically, assuming 24/7 operation for the full month, you can have a single 4-core / 24GB RAM system