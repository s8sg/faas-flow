# faasflow-minio-statemanager
A **[faasflow](https://github.com/s8sg/faasflow)** statemanager implementation that uses minio DB to store state  
which can also be used with s3 bucket

## Getting Stated

### Deploy Minio
* Generate secrets for Minio
```bash
SECRET_KEY=$(head -c 12 /dev/urandom | shasum| cut -d' ' -f1)
ACCESS_KEY=$(head -c 12 /dev/urandom | shasum| cut -d' ' -f1)
```
#### Deploy in Kubernets
* Store the secrets in Kubernetes
```bash
kubectl create secret generic -n openfaas-fn \
 s3-secret-key --from-literal s3-secret-key="$SECRET_KEY"
kubectl create secret generic -n openfaas-fn \
 s3-access-key --from-literal s3-access-key="$ACCESS_KEY"
```
* Install Minio with helm
```bash
helm install --name cloud --namespace openfaas \
   --set accessKey=$ACCESS_KEY,secretKey=$SECRET_KEY,replicas=1,persistence.enabled=false,service.port=9000,service.type=NodePort \
  stable/minio
```
The name value should be `cloud-minio.openfaas.svc.cluster.local`  
   
Enter the value of the DNS above into `s3_url` in `gateway_config.yml` adding the port at the end: `cloud-minio-svc.openfaas.svc.cluster.local:9000`

#### Deploy in Swarm
* Store the secrets
```bash
echo -n "$SECRET_KEY" | docker secret create s3-secret-key -
echo -n "$ACCESS_KEY" | docker secret create s3-access-key -
```
* Deploy Minio
```bash
docker service rm minio

docker service create --constraint="node.role==manager" \
 --name minio \
 --detach=true --network func_functions \
 --secret s3-access-key \
 --secret s3-secret-key \
 --env MINIO_SECRET_KEY_FILE=s3-secret-key \
 --env MINIO_ACCESS_KEY_FILE=s3-access-key \
minio/minio:latest server /export
```
> Note: For debugging and testing. You can expose the port of Minio with docker service update minio --publish-add 9000:9000, but this is not recommended on the public internet.

### Use Minio stateManager in `faasflow`
* Set the `stack.yml` with the necessary environments
```yaml
      s3_url: "minio:9000"
      s3_bucket: "faasflow"
      s3_tls: false
    secrets:
      - s3-secret-key
      - s3-access-key
```
* Use the `faasflowMinioStateManager` as a StateManager
```go
minioStateManager "github.com/s8sg/faasflowMinioStateManager"

func Define(flow *faasflow.Workflow, context *faasflow.Context) (err error) {
 
       // initialize minio StateManager
       miniosm, err := minioStateManager.GetMinioStateManager()
       if err != nil {
               return err
       }
       // Set StateManager
       context.SetStateManager(miniosm)
       
       // Define pipeline
       ...
}
```
