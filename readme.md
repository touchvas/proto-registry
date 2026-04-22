# How to generate buf files

1. Every micro service should have its own folder and proto file inside the folder
2. Every miccro service should have the auto generated files in its own folder under `gen/go`
3. Whenever you change a .proto file: Run `buf generate proto`
4. Commit the changes and the newly generated .pb.go files. 
5. Checck ccurrent tag number `git describe --tags`
5. Tag the release: `git tag $new_tag_number && git push origin $new_tag_number`

## Standardize Your Package Options
- To avoid import conflicts, every .proto file must have a `go_package option` that points back to your registry repository. Update your files as follows:
  - **Identity:** `option go_package = "github.com/your-org/proto-registry/gen/go/identity";`
  - **Notification:** `option go_package = "github.com/your-org/proto-registry/gen/go/notification";`
  - **Payment:** `option go_package = "github.com/your-org/proto-registry/gen/go/payment";`
  - **Wallet:** `option go_package = "github.com/your-org/proto-registry/gen/go/wallet";`

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