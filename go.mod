module github.com/blackwell-systems/gcp-iam-emulator

go 1.24.0

require (
	google.golang.org/genproto v0.0.0-20260126211449-d11affda4bed
	google.golang.org/grpc v1.78.0
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/blackwell-systems/gcp-emulator-auth v0.2.0

require (
	cloud.google.com/go/iam v1.5.3 // indirect
	github.com/fsnotify/fsnotify v1.9.0
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260120174246-409b4a993575 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120174246-409b4a993575 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
