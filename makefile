proto/xray.proto:
	mkdir -p proto
	curl -L -o proto/xray.proto https://raw.githubusercontent.com/infanasotku/netku/master/proto/xray.proto

generate: proto/xray.proto
	protoc --go_out=infra/grpc/ \
	--go_opt=Mproto/xray.proto=./gen \
	--go-grpc_out=infra/grpc/ \
	--go-grpc_opt=Mproto/xray.proto=./gen \
	proto/xray.proto