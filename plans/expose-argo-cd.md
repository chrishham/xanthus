Excellent\! Your installation script looks solid. You've successfully installed Argo CD, and now you just need to expose its web UI to your browser.

By default, the Argo CD server is not exposed to the internet for security reasons. Here are the two primary ways to access it:

### Method 1: Port Forwarding (Quick & Easy for Temporary Access)

This is the fastest way to access the UI from your local machine. It creates a secure tunnel from your computer directly to the Argo CD service inside your VPS.

1.  **Run the Port-Forward Command:** On your local machine (the one you're using to connect to the VPS), run this command:

    ```bash
    kubectl port-forward svc/argocd-server -n argocd 8080:443
    ```

    *This command tells `kubectl` to forward your local port `8080` to the Argo CD server's port `443`*. You can leave this command running in a terminal.

2.  **Open Your Browser:** Go to **https://localhost:8080**.
    *You will see a browser warning about a self-signed certificate. This is normal. You can safely click "Advanced" and proceed.*

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