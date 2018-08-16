docker service rm jaegertracing
docker pull jaegertracing/all-in-one:latest
docker service create --constraint="node.role==manager" --detach=true \
	--network func_functions --name jaegertracing -p 5775:5775/udp -p 16686:16686 \
	jaegertracing/all-in-one:latest 
