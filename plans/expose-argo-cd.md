3.  **Log In:**

      * **Username:** `admin`
      * **Password:** To get the initial password, run the following command on your VPS:
        ```bash
        argocd admin initial-password -n argocd
        ```
        *Alternatively, you can use `kubectl`:*
        ```bash
        kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo
        ```
        Copy the password from the output and paste it into the login screen.

### Method 2: Ingress (Permanent & Recommended)

For permanent, production-like access, you should expose Argo CD through an Ingress, just like we discussed for other web applications in the document. This will allow you to access it at a subdomain like `argocd.yourdomain.com`.

This involves:

1.  Ensuring you have an Ingress controller (like Nginx or Traefik) running in your cluster.
2.  Creating an `Ingress` Kubernetes resource that points a hostname to the `argocd-server` service on port `443`.
3.  Configuring TLS for a secure connection, likely using `cert-manager`.

The port-forwarding method is perfect for getting started right now.