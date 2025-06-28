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
Argocd Installation failed:
Invalid SSL certificate 

cerfificates stopped working!

 deployPredefinedApplication function is too specific for code server or not?

# Settings

- separate page to update the Hetzner api key ?

## Port forward from vps to local machine

- need for both tunelling
ssh -i C:\Users\E40274\Desktop\Test\SSH_ubuntu-ampere-4core-24gbRam.key -D 8089 -N -f ubuntu@158.180.27.32
- and port forwarding one or more ports
ssh -i C:\Users\E40274\Desktop\Test\SSH_ubuntu-ampere-4core-24gbRam.key   -N -L 8080:localhost:8080 ubuntu@158.180.27.32
ssh -i C:\Users\E40274\Desktop\Test\SSH_ubuntu-ampere-4core-24gbRam.key   -N -L 3000:localhost:3000 ubuntu@158.180.27.32

ssh -i C:\Users\E40274\Desktop\Test\SSH_ubuntu-ampere-4core-24gbRam.key   -N -L 8082:localhost:8082 ubuntu@158.180.27.32
ssh -i C:\Users\E40274\Desktop\Test\SSH_ubuntu-ampere-4core-24gbRam.key   -N -L 9000:localhost:9000 ubuntu@158.180.27.32
ssh -i C:\Users\E40274\Desktop\Test\SSH_ubuntu-ampere-4core-24gbRam.key   -N -L 9001:localhost:9001 ubuntu@158.180.27.32


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