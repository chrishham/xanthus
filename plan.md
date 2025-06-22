I want to create a go web app with the name "Xanthus" (gin/htmx/tailwind/alpinejs) that helps developers/vibe coders to deploy their apps to a k3s cluster at a Hetzner VPS Ubuntu instance. 

The app is to be used at a desktop pc by downloading the go build executable and run at localhost at first available port which will be informed to the user.



Features: 

- Cloudfare DNS management, the app will connect to cloudfare and take all the actions necessary as per the sample.go file indicates to configure ssl for a specific domain and get the certificate needed for enabling tls at k3s ingress with traefik

- Given a Hetzner api key the app will provision an ubuntu VPS that has the essential software istalled and preconfigured .

- Every setting of the app(ie the hetzner api key, the hetzner ip and others) will be stored at kv at cloudfare so the user the only thing that needs to remember for login is his Cloudfare api key.