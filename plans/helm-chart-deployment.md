# How to present values to user to fill for the helm chart
How to Automatically Obtain Values from a Helm Chart

You can get all the user-configurable parameters directly from the chart's files. There are two key files you should use:

    values.yaml: This is the most important file. It contains all the default values for a chart. Every key in this file represents a parameter that a user can override during installation. You can programmatically fetch and parse this file to build a basic input form for the user.

        How to get it: You can use the helm command-line tool to display the contents of this file:

        helm show values <chart-name>

        For example, helm show values bitnami/mysql. Your web UI's backend can execute this command and send the resulting YAML to the frontend.

    values.schema.json (Optional but Recommended): This file is a JSON Schema that defines the structure and validation rules for the values.yaml file. If a chart includes a schema, you can use it to create a much more user-friendly and validated form. The schema provides:

        Type information: (e.g., string, integer, boolean, array). This allows you to render the correct UI element (e.g., a text box, a slider, a checkbox, a list).

        Descriptions: You can use these to show tooltips or help text next to each field, explaining what it does.

        Validation rules: (e.g., minimum/maximum values, required fields, regular expression patterns). This allows for client-side validation before the user even submits the form.

Key Values to Present to the User

While every chart is different, here are the most common and important values you should always aim to present in your UI. These are typically found in the values.yaml file.
1. General Application Configuration

These are high-level settings that control the deployment.

    replicaCount: The number of application pods to run. This is a fundamental scaling parameter.

    namespace: The Kubernetes namespace to install the chart into. While this is an installation-time parameter, it's often useful to expose it.

2. Image Configuration

This determines the actual software that will be run.

    image.repository: The name of the Docker image (e.g., nginx, bitnami/mysql).

    image.tag: The version or tag of the image (e.g., 1.21.6, latest). This is one of the most frequently changed values.

    image.pullPolicy: When to pull the image (e.g., Always, IfNotPresent).

3. Network and Service Exposure

How the application is exposed to the network. The ingress section is the most reliable indicator that a chart exposes a web server.

    service.type: The type of Kubernetes service (e.g., ClusterIP, NodePort, LoadBalancer).

    service.port: The port that the service exposes.

    ingress.enabled: A boolean to enable or disable the creation of an Ingress resource.

    ingress.className: The ingressClassName to use (e.g., nginx, traefik).

    ingress.annotations: A key-value map for adding annotations, often used for cert-manager (kubernetes.io/tls-acme: "true"), authentication, or rewrite rules.

    ingress.hosts: A list of host objects, where each object defines the host (subdomain) and the paths.

    ingress.tls: A list of TLS configuration blocks. Each block typically specifies the hosts it applies to and the secretName containing the TLS certificate.

A typical ingress section in values.yaml might look like this:

ingress:
  enabled: true
  className: "traefik"
  annotations:
    kubernetes.io/tls-acme: "true"
  hosts:
    - host: "myapp.example.com"
      paths:
        - path: /
          pathType: Prefix
  tls:
    - hosts:
        - "myapp.example.com"
      secretName: "myapp-tls-secret"

4. Resource Management

Controls the CPU and memory allocated to the application pods.

    resources.limits.cpu: The maximum CPU that can be used.

    resources.limits.memory: The maximum memory that can be used.

    resources.requests.cpu: The amount of CPU to request at startup.

    resources.requests.memory: The amount of memory to request at startup.

5. Persistence

For stateful applications that need to store data.

    persistence.enabled: A boolean to enable or disable persistent storage.

    persistence.size: The size of the persistent volume (e.g., 10Gi).

    persistence.storageClass: The name of the StorageClass to use.

6. Application-Specific Configuration

This is the largest and most varied category. You should display all other values from the values.yaml file, as they control the application's internal behavior. Examples include:

    Database connection strings and credentials.

    Configuration for environment variables.

    Settings for enabling/disabling specific application features.

    Passwords and secrets (which should be handled securely in your UI).

By parsing the values.yaml and, when available, the values.schema.json, you can create a robust and user-friendly UI for managing any Helm installation without hardcoding fields for each chart.

## Check if the helm chart exposes a web server for linkage with a subdomain

While there isn't a universal flag that says isWebServer: true, you can look for common conventions that strongly indicate a web service is being exposed.

Here are the key things to look for in the values.yaml, in order of reliability:

    The ingress Section (Most Reliable): This is the most direct indicator. The entire purpose of a Kubernetes Ingress is to manage external access to HTTP/HTTPS services, which is exactly what you need for a subdomain.

        Check for ingress.enabled: If this key exists, the chart is explicitly designed to be exposed via an Ingress controller.

        Check for ingress.hosts: This is the specific field where the user would enter their subdomain. If your UI finds this section, you should definitely present the user with the option to enable the ingress and provide a hostname.

    The service Section (Strong Indicator): If there's no ingress section, you can look at how the service is configured.

        Check service.type: If the type is set to LoadBalancer, it's meant to be exposed directly to the internet with its own IP address. This is very common for web services.

        Check service.port or port names: Look for common web ports like 80, 443, or 8080. Often, the ports will have names like http, https, or web.

Your Strategy in the UI should be:

Programmatically parse the values.yaml and check:

    Does an ingress object exist?

        If yes, present an option like "Expose via a domain?" When the user checks it, show them the input fields for ingress.hosts and any other ingress-related settings. This is the ideal scenario.

    If not, does the service.type default to LoadBalancer?

        If yes, this is still likely a web service. You could inform the user that the service will be exposed via a public IP address. Linking a subdomain would then be a DNS task for the user to perform after they get the IP address.

By checking for the presence of the ingress section, you can reliably automate the process of offering subdomain configuration to the user at the time of installation.