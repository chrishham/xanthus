### Instructions for `@web/templates/vps-create.html`

#### 1. Domain Check Before VPS Creation

* Before initiating the "Create VPS" wizard, **check whether there are any managed domains available via Xanthus**.
* If **no managed domains** are found:

  * Display an informative message to the user (e.g., *"You need to have at least one managed domain before creating a VPS."*).
  * Redirect the user to the **DNS management page**.

#### 2. Domain Selection and DNS Configuration

* Allow the user to **select a domain** from the list of **domains managed by Xanthus** to link to the new VPS.
* Once the VPS has been created and **assigned a public IP address**:

  1. **Delete all existing A records** for that domain from **Cloudflare**.
  2. **Add the following A records** to point to the new VPS IP (example shown using `188.245.145.211`):

  ```dns
  ;; A Records
  *.myclasses.gr.        1   IN  A   188.245.145.211  ; cf_tags=cf-proxied:true
  myclasses.gr.          1   IN  A   188.245.145.211  ; cf_tags=cf-proxied:true
  www.myclasses.gr.      1   IN  A   188.245.145.211  ; cf_tags=cf-proxied:true
  ```

#### 3. TLS Certificate Secret for ArgoCD

* After `k3s` and `ArgoCD` are up and running, **create a Kubernetes TLS secret** using the certificate and private key issued by Cloudflare (Origin Certificate):

  ```bash
  kubectl create secret tls kantoliana-gr-cloudflare-tls \
    --cert=kantoliana-gr-cert.crt \
    --key=kantoliana-gr-private.key \
    -n default
  ```

  > 📝 Replace the certificate/key file paths and secret name according to the selected domain. Certificates and keys should be securely retrieved from your KV store.

#### 4. Expose ArgoCD via Ingress

* Create and apply an `Ingress` resource to expose the ArgoCD service:

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: kantoliana-client-ingress
    annotations:
      kubernetes.io/tls-acme: "true"
  spec:
    ingressClassName: traefik
    rules:
    - host: "argocd.kantoliana.gr"
      http:
        paths:
        - path: /
          pathType: Prefix
          backend:
            service:
              name: argocd
              port:
                number: 80  # Change this if ArgoCD listens on a different port
    tls:
    - hosts:
      - "argocd.kantoliana.gr"
      secretName: kantoliana-gr-cloudflare-tls
  ```

  > 📝 Make sure the `port.number` matches the actual port on which the ArgoCD service is listening.

---

### ✅ Result

After completing the steps above, the user will be able to access and manage ArgoCD at:

```
https://argocd.kantoliana.gr
```

Let me know if you'd like these instructions in Markdown, HTML, or inlined in a Go template.
