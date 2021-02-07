docker build -t danilarff86/multi-client:latest -t danilarff86/multi-client:$SHA -f ./client/Dockerfile ./client
docker build -t danilarff86/multi-server:latest -t danilarff86/multi-server:$SHA -f ./server/Dockerfile ./server
docker build -t danilarff86/multi-worker:latest -t danilarff86/multi-worker:$SHA -f ./worker/Dockerfile ./worker

docker push danilarff86/multi-client:latest
docker push danilarff86/multi-server:latest
docker push danilarff86/multi-worker:latest
docker push danilarff86/multi-client:$SHA
docker push danilarff86/multi-server:$SHA
docker push danilarff86/multi-worker:$SHA

kubectl apply -f k8s
kubectl set image deployments/client-deployment client=danilarff86/multi-client:$SHA
kubectl set image deployments/server-deployment server=danilarff86/multi-server:$SHA
kubectl set image deployments/worker-deployment worker=danilarff86/multi-worker:$SHA