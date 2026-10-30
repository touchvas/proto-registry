# How to generate buf files

1. Every micro service should have its own folder and proto file inside the folder
2. Every micro service should have the auto generated files in its own folder under `gen/go`
3. Whenever you change a .proto file: Run `make generate` (`buf generate proto`, then the OpenAPI files)
4. Commit the changes and the newly generated .pb.go files. 
5. Check current tag number `git describe --tags`
5. Tag the release: `git tag $new_tag_number && git push origin $new_tag_number`

## Standardize Your Package Options
- To avoid import conflicts, every .proto file must have a `go_package option` that points back to your registry repository. Update your files as follows:
  - **Identity:** `option go_package = "github.com/your-org/proto-registry/gen/go/identity";`
  - **Notification:** `option go_package = "github.com/your-org/proto-registry/gen/go/notification";`
  - **Payment:** `option go_package = "github.com/your-org/proto-registry/gen/go/payment";`
  - **Wallet:** `option go_package = "github.com/your-org/proto-registry/gen/go/wallet";`

## APIs served through api-gateway-service
The UI calls one gateway, which translates REST/JSON to gRPC. Endpoints the UI uses are annotated
with `google.api.http` (from the `buf.build/googleapis/googleapis` dependency in `proto/buf.yaml`):

```protobuf
import "google/api/annotations.proto";

service AuthService {
  rpc Login(LoginRequest) returns (SessionResult) {
    option (google.api.http) = { post: "/v1/login" body: "*" };
  }
}
```

- `buf generate proto` then also writes `*.pb.gw.go` next to the Go code, and `make openapi` (part of
  `make generate`) writes one OpenAPI file per gateway profile: `openapi/player.swagger.json` and
  `openapi/admin.swagger.json`, which the gateway serves at `/docs`. Each is generated from the protos
  listed in the Makefile's `PLAYER_PROTOS` / `ADMIN_PROTOS`: add a module's new public API there.
  `cmd/openapi` then documents amounts as numbers, each module's error body and the auth headers.
- Versioned packages live in versioned folders: `identity.v1` in `proto/identity/v1/`, with
  `option go_package = "github.com/touchvas/proto-registry/gen/go/identity/v1;identityv1";`.
- Never annotate service-to-service RPCs (e.g. `identity.v1.IdentityService`): an annotated method
  is reachable from the internet through the gateway. Keep public and internal RPCs in separate services.
- Amounts are `string` fields holding the exact decimal, marked `[(touchvas.api.v1.json_number) = true]`
  (`proto/touchvas/api/v1/json.proto`): the gateway writes and reads them as JSON numbers. Mark an
  `int64` the same way to keep it a JSON number.
- Public messages avoid unmarked `int64` (JSON renders it as a string) and `oneof` in responses; the gateway
  keeps proto field names (`snake_case`) in JSON.
- Run `make install` once: the gateway plugins are pinned to the grpc-gateway runtime version the
  gateway uses.

# How to use in other services
## Import in Microservices
Now, your microservices don't need the .proto files at all. They simply import the generated Go package via go get:

```shell
go get github.com/touchvas/proto-registry/proto-registry/gen/go/identity@v1.0.1
```

```go
import "github.com/touchvas/proto-registry/gen/go/identity"

func main() {
    profile := &identity.Profile{Id: 123}
}
```