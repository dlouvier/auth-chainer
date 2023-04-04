# auth-chainer
This tool is a solution for the lack of multi service authorisation of the module `ngx_http_auth_request_module` of [Nginx](http://nginx.org/en/docs/http/ngx_http_auth_request_module.html).

It was built because of the need of having authentification and device authorisation for a [Zero Trust Architecture lab](https://github.com/dlouvier/zero-trust-architecture-lab).

It depends on [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy) and [Quiq/webauthn_proxy](https://github.com/Quiq/webauthn_proxy).

A detailed demo and resources how to run it, it is available at [dlouvier/zero-trust-architecture-lab](https://github.com/dlouvier/zero-trust-architecture-lab/) repository.

# Docker images
Check latest docker images at [DockerHub](https://hub.docker.com/r/dlouvier/auth-chainer/tags)

# Usage
In Kubernetes, configure and run a deployment of **auth-chainer** like:

```
containers:
- name: container
image: dlouvier/auth-chainer:888f171
env:
    - name: AUTHENTIFICATION_SERVICE_HOST
    value: "oauth2-proxy"
    - name: AUTHORISATION_SERVICE_HOST
    value: "webauthn"
    - name: SESSION_COOKIE_SECRET
    valueFrom:
        secretKeyRef:
        name: auth-chainer
        key: session_cookie_secret
ports:
- containerPort: 8080
    protocol: TCP
    name: http
```

Additionally, you will need to run [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy) and [Quiq/webauthn_proxy](https://github.com/Quiq/webauthn_proxy) too.

Then configure two Ingresses resources like:
```
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: private
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "http://auth-chainer.zta-demo.svc.cluster.local/auth" # this path domain should be resolveable by the cluster
    nginx.ingress.kubernetes.io/auth-signin: "/register" # this domain should be resolvable by client           
    nginx.ingress.kubernetes.io/auth-response-headers: "X-Auth-Request-User"
spec:
  tls:
  - secretName: site-certificates
  rules:
  - host: private-7f000001.nip.io
    http:
      paths:
      - pathType: Prefix
        path: /
        backend:
          service:
            name: echo-server
            port:
              name: http
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: auth-sites
  # This ingress doesn't have any auth annotation,
  # because otherwise we won't be able to log the user.
  # note: the '/' path of the app is declared in private ingress.
spec:
  tls:
  - secretName: site-certificates
  rules:
  - host: private-7f000001.nip.io
    http:
      paths:
      - pathType: Prefix
        path: /oauth2
        backend:
          service:
            name: oauth2-proxy
            port:
              name: http
      - pathType: Prefix
        path: /register
        backend:
          service:
            name: auth-chainer
            port:
              name: http
      - pathType: Prefix
        path: /webauthn
        backend:
          service:
            name: webauthn
            port:
              name: http
```