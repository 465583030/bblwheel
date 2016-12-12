protoc --go_out=plugins=grpc:. *.proto
protoc --plugin=/usr/local/protoc/bin/protoc-gen-grpc-java --grpc-java_out=java --java_out=java *.proto
