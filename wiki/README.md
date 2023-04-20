# wiki

# K8s token & ca.crt

## api server host
```
export KUBERNETES_SERVICE_HOST=127.0.0.1
export KUBERNETES_SERVICE_PORT=6443
```

## 绑定 namespace token 
token
```
eyJhbGciOiJSUzI1NiIsImtpZCI6Ik42cGRZYU1tbW1PRGZfRUROTjY0QmVndUtVVnFWekJ3YUVCOHZMdzhSanMifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiLCJrM3MiXSwiZXhwIjoxNjY4OTI4NDkzLCJpYXQiOjE2MzczOTI0OTMsImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJ6aHVpeWkiLCJwb2QiOnsibmFtZSI6ImdvbGFuZy1za2VsZXRvbi01NDY3OGM2YmY3LTZ2djdoIiwidWlkIjoiNjY2NmQ3OWUtMmY5NS00OTc1LThiOGEtYmUyYjBiNDg3OGRmIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJkZWZhdWx0IiwidWlkIjoiODQxMWVhYjUtY2EzNy00ZWExLWJlNjEtMTdkYzE2ZTc3NmQzIn0sIndhcm5hZnRlciI6MTYzNzM5NjEwMH0sIm5iZiI6MTYzNzM5MjQ5Mywic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OnpodWl5aTpkZWZhdWx0In0.usJgD9hBuDQ_GvKD3t54bgoH1GES7dp9I2XYFNTxbP8pqC-T0N7TIMSWiuHXQyvw0Pf2_MfJWOhIS9AmhCmT9qGv_M3PtD92npWj9tCKRq9aal6B-Rmr12Aj0K4J_fO-h3LyUaK6_QtBzDtlnpkmV_LC_TAo41xOq4gmEKKqDde9aw9XQiiAekYwuubonjMYS9XKdLekK57rNtFaFtIyQOlMolHcND07RsVVso-YwZsPSMnSdUgwCk--grnfgHoF8Mlm0EcN0w9cT7dlyO2DoB8CmUE7n3INtaFsNS4QEgV79L-uWkMfNPCMgdxBtf2SG4KycDITmWjYDfAeVZBjug
```

ca.crt
```
-----BEGIN CERTIFICATE-----
MIIBdzCCAR2gAwIBAgIBADAKBggqhkjOPQQDAjAjMSEwHwYDVQQDDBhrM3Mtc2Vy
dmVyLWNhQDE2MzQwMDM3ODkwHhcNMjExMDEyMDE1NjI5WhcNMzExMDEwMDE1NjI5
WjAjMSEwHwYDVQQDDBhrM3Mtc2VydmVyLWNhQDE2MzQwMDM3ODkwWTATBgcqhkjO
PQIBBggqhkjOPQMBBwNCAAQ9eBT7Pvc6MPUBXFPqGnt09vA9RAXbNWbryEC+KivI
VXDLkao5irHKoEQTqFwfJKOKBXUXmkJpXMMIeZzMbLipo0IwQDAOBgNVHQ8BAf8E
BAMCAqQwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUZeaQqFzP2REte4vjK8ya
o7k8uNcwCgYIKoZIzj0EAwIDSAAwRQIgC6yBXvLSDfesW5fBwxjEIeoYRGf3lZW9
feNDCCEJOR8CIQC7ycjl+xhU23hVjsfuu9M1Cecqzx6nApSqU+/wshYV5A==
-----END CERTIFICATE-----
```


## 声明 K8s clientset

```go
// creates the in-cluster config
config, errConfig := rest.InClusterConfig()
if errConfig != nil {
    panic(errConfig.Error())
}
// creates the clientset
clientset, errClientSet := kubernetes.NewForConfig(config)
if errClientSet != nil {
    panic(errClientSet.Error())
}

```