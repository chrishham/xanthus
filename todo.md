## VPS
argocd admin initial-password -n argocd
download approriate binaries for argocd depending on vps architecture

streamline @internal/services/cloudinit.yaml with @internal/handlers/vps.go, is there anything that can be moved to the cloudinit?

give the user the initial password for argocd :

kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo

@web/templates/vps-manage.html
- show total days and hours since vps creation
- at resources add the storage capacity

- is there a way to determine the degree of utilisation of the vps ? so that the user will know if he needs to provision another vps


- Uptime: 1d 0h no uptime, time since creation

- systemd-resolved: ✅ Active => remove it 
- Fix spacing , label and status should be closer together
SSH Status: ✅ Connected
K3s Service: ✅ Running


- terminal: user can add multiple ssh sessions as tabs

@web/templates/vps-create.html
- at vps creation ensure that login without ssh is completely disabled
- at create a vps, list and the dedicated instances and add an appropriate filter
- choose server type: add option to filter out unavailable
- install filebrowser https://github.com/gtsteffaniak/filebrowser
- let the user choose a domain from managed by xanthus domains  to link to the vps. Add check that
if there are not any managed domains dont let the user initiate the create vps wizard but display an informative message and redirect him to dns page. Then after vps obtains an ip address, delete all the A records at the cloudfare at the domain and add the A records necessary for resolving domain and *.domain to the new ip.

- How to enable argocd web app?

## Applications
@web/templates/applications.html

- add new repo doesn't show an input box for user to type
- when typing into input box in helm chart creation, nothing happens.

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