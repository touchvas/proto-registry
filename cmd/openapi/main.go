// Command openapi finishes the OpenAPI file protoc-gen-openapiv2 writes for one
// api-gateway-service profile, so it documents what the gateway actually answers:
//
//   - fields marked (touchvas.api.v1.json_number) are JSON numbers, not strings;
//   - errors are the module's body ({"error"} from identity, {code, number, message} from
//     wallet and casino), not google.rpc.Status;
//   - each operation is tagged and named by its full service name, so two modules' services of
//     the same name (identity's and wallet's AuditService) stay apart;
//   - the headers a client authenticates and names its brand with are declared, for "Try it
//     out".
//
// make openapi runs it; see the Makefile.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	apiv1 "github.com/touchvas/proto-registry/gen/go/touchvas/api/v1"
)

// profiles describes each gateway's file.
var profiles = map[string]struct {
	title, description string
	// brandHeader is how a client of this gateway names a brand.
	brandHeader, brandDescription string
}{
	"player": {
		title: "Player API",
		description: "What the player apps call, through api-gateway-service's player profile. " +
			"Send `X-Brand-Code` (the brand's code, e.g. betxyz-ke) on every request, and `Authorization: Bearer <token>` once logged in.\n\n" + common,
		brandHeader:      "X-Brand-Code",
		brandDescription: "The brand's code, e.g. betxyz-ke. Sent on every request.",
	},
	"admin": {
		title: "Back-office API",
		description: "What the back office calls, through api-gateway-service's admin profile. " +
			"Send `Authorization: Bearer <token>` from the admin login; staff on several brands (and SUPER_ADMIN, always) name the brand they work on in `X-Act-As-Brand`.\n\n" + common,
		brandHeader:      "X-Act-As-Brand",
		brandDescription: "The ID of the brand to work on, for staff on several brands. SUPER_ADMIN must always send it.",
	},
}

const common = "Amounts are JSON numbers in the currency's units (940.5 is KES 940.50), with at most 4 decimals and no exponent; " +
	"a numeric string (\"940.5\") is accepted too. Errors from identity are `{\"error\": \"...\"}`; from wallet and casino, " +
	"`{\"code\": \"...\", \"number\": 0, \"message\": \"...\"}`."

// codedErrorPackages answer errors as {code, number, message}; the others as {"error"}.
var codedErrorPackages = []string{"wallet.", "casino."}

func main() {
	image := flag.String("image", "", "file descriptor set of the protos (buf build --as-file-descriptor-set)")
	profile := flag.String("profile", "", "player or admin")
	protos := flag.String("protos", "", "the profile's proto directories under proto/, space-separated (identity/v1 wallet/player/v1)")
	in := flag.String("in", "", "the file protoc-gen-openapiv2 wrote")
	out := flag.String("out", "", "where to write the finished file")
	flag.Parse()

	if err := run(*image, *profile, strings.Fields(*protos), *in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "openapi:", err)
		os.Exit(1)
	}
}

func run(imagePath, profile string, protos []string, in, out string) error {
	p, ok := profiles[profile]
	if !ok {
		return fmt.Errorf("unknown profile %q", profile)
	}
	files, err := loadFiles(imagePath)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}

	if err := finish(doc, files, profile, protos); err != nil {
		return err
	}
	doc["info"] = map[string]any{"title": p.title, "description": p.description, "version": "v1"}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out, append(data, '\n'), 0o644)
}

// loadFiles reads the descriptors, with the options this command reads resolved.
func loadFiles(path string) (*protoregistry.Files, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	types := new(protoregistry.Types)
	for _, ext := range []protoreflect.ExtensionType{apiv1.E_JsonNumber, annotations.E_Http} {
		if err := types.RegisterExtension(ext); err != nil {
			return nil, err
		}
	}
	var set descriptorpb.FileDescriptorSet
	if err := (proto.UnmarshalOptions{Resolver: types}).Unmarshal(raw, &set); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return protodesc.NewFiles(&set)
}

func finish(doc map[string]any, files *protoregistry.Files, profile string, protos []string) error {
	defs, _ := doc["definitions"].(map[string]any)
	if defs == nil {
		defs = map[string]any{}
		doc["definitions"] = defs
	}

	// Number fields, in message definitions and in request bodies grpc-gateway names
	// <Service>.<Method>Body or <package>.<Message without Request>Body.
	for name, def := range defs {
		if md := messageFor(files, name); md != nil {
			numbers(def, md)
		}
	}

	defs["Error"] = object(map[string]any{"error": str("Why the request failed.")})
	defs["CodedError"] = object(map[string]any{
		"code":    str("The result code, e.g. INSUFFICIENT_FUNDS."),
		"number":  map[string]any{"type": "integer", "format": "int32", "description": "The result code's number."},
		"message": str("Why the request failed."),
	})

	methods := httpMethods(files, protos)
	paths, _ := doc["paths"].(map[string]any)
	tags := map[string]bool{}
	for path, item := range paths {
		ops, _ := item.(map[string]any)
		for verb, op := range ops {
			o, _ := op.(map[string]any)
			md, ok := methods[strings.ToUpper(verb)+" "+path]
			if !ok {
				return fmt.Errorf("%s %s: no annotated method serves it", strings.ToUpper(verb), path)
			}
			service := string(md.Parent().FullName())
			o["tags"] = []any{service}
			o["operationId"] = service + "." + string(md.Name())
			tags[service] = true

			errorBody := "Error"
			for _, prefix := range codedErrorPackages {
				if strings.HasPrefix(service, prefix) {
					errorBody = "CodedError"
				}
			}
			responses, _ := o["responses"].(map[string]any)
			responses["default"] = map[string]any{
				"description": "The request failed; the HTTP status says how.",
				"schema":      map[string]any{"$ref": "#/definitions/" + errorBody},
			}
		}
	}
	doc["tags"] = tagList(tags)

	p := profiles[profile]
	doc["securityDefinitions"] = map[string]any{
		"bearer": map[string]any{"type": "apiKey", "in": "header", "name": "Authorization",
			"description": "`Bearer <token>`, the token login returns."},
		"brand": map[string]any{"type": "apiKey", "in": "header", "name": p.brandHeader, "description": p.brandDescription},
	}
	doc["security"] = []any{map[string]any{"bearer": []any{}, "brand": []any{}}}

	prune(doc, defs)
	return nil
}

// messageFor returns the message a definition describes: the message of that name, or the
// request whose body it is.
func messageFor(files *protoregistry.Files, name string) protoreflect.MessageDescriptor {
	if d, err := files.FindDescriptorByName(protoreflect.FullName(name)); err == nil {
		if md, ok := d.(protoreflect.MessageDescriptor); ok {
			return md
		}
	}
	base, ok := strings.CutSuffix(name, "Body")
	if !ok {
		return nil
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		if d, err := files.FindDescriptorByName(protoreflect.FullName(base[:i])); err == nil {
			if sd, ok := d.(protoreflect.ServiceDescriptor); ok {
				if m := sd.Methods().ByName(protoreflect.Name(base[i+1:])); m != nil {
					return m.Input()
				}
			}
		}
	}
	if d, err := files.FindDescriptorByName(protoreflect.FullName(base + "Request")); err == nil {
		if md, ok := d.(protoreflect.MessageDescriptor); ok {
			return md
		}
	}
	return nil
}

// numbers documents md's number fields in def as the gateway writes them.
func numbers(def any, md protoreflect.MessageDescriptor) {
	d, _ := def.(map[string]any)
	props, _ := d["properties"].(map[string]any)
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if on, _ := proto.GetExtension(fd.Options(), apiv1.E_JsonNumber).(bool); !on {
			continue
		}
		prop, ok := props[string(fd.Name())].(map[string]any)
		if !ok {
			continue
		}
		target := prop
		if fd.IsList() {
			target, _ = prop["items"].(map[string]any)
		}
		if fd.Kind() == protoreflect.StringKind {
			delete(target, "format")
			target["type"] = "number"
			note := "Up to 4 decimals; a numeric string is accepted too."
			if desc, _ := prop["description"].(string); desc != "" {
				note = desc + " " + note
			}
			prop["description"] = note
		} else {
			target["type"] = "integer"
		}
	}
}

// httpMethods maps "VERB /path" to the method annotated with it in the profile's protos (the
// other profile may serve the same path: both have POST /v1/challenge).
func httpMethods(files *protoregistry.Files, protos []string) map[string]protoreflect.MethodDescriptor {
	out := map[string]protoreflect.MethodDescriptor{}
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if !slices.ContainsFunc(protos, func(dir string) bool { return strings.HasPrefix(fd.Path(), dir+"/") }) {
			return true
		}
		for i := 0; i < fd.Services().Len(); i++ {
			methods := fd.Services().Get(i).Methods()
			for j := 0; j < methods.Len(); j++ {
				md := methods.Get(j)
				rule, _ := proto.GetExtension(md.Options(), annotations.E_Http).(*annotations.HttpRule)
				for _, r := range append([]*annotations.HttpRule{rule}, rule.GetAdditionalBindings()...) {
					if verb, path := route(r); verb != "" {
						out[verb+" "+path] = md
					}
				}
			}
		}
		return true
	})
	return out
}

func route(r *annotations.HttpRule) (verb, path string) {
	switch p := r.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return "GET", p.Get
	case *annotations.HttpRule_Post:
		return "POST", p.Post
	case *annotations.HttpRule_Put:
		return "PUT", p.Put
	case *annotations.HttpRule_Patch:
		return "PATCH", p.Patch
	case *annotations.HttpRule_Delete:
		return "DELETE", p.Delete
	}
	return "", ""
}

// prune drops definitions nothing refers to any more (google.rpc.Status and the like).
func prune(doc, defs map[string]any) {
	for {
		used := map[string]bool{}
		for name, def := range defs {
			refs(def, func(ref string) {
				if ref != name {
					used[ref] = true
				}
			})
		}
		for key, v := range doc {
			if key != "definitions" {
				refs(v, func(ref string) { used[ref] = true })
			}
		}
		removed := false
		for name := range defs {
			if !used[name] {
				delete(defs, name)
				removed = true
			}
		}
		if !removed {
			return
		}
	}
}

func refs(v any, found func(string)) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if s, ok := val.(string); ok && k == "$ref" {
				found(strings.TrimPrefix(s, "#/definitions/"))
			} else {
				refs(val, found)
			}
		}
	case []any:
		for _, val := range x {
			refs(val, found)
		}
	}
}

func tagList(tags map[string]bool) []any {
	names := make([]string, 0, len(tags))
	for t := range tags {
		names = append(names, t)
	}
	slices.Sort(names)
	out := make([]any, 0, len(names))
	for _, n := range names {
		out = append(out, map[string]any{"name": n})
	}
	return out
}

func object(props map[string]any) map[string]any {
	return map[string]any{"type": "object", "properties": props}
}

func str(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}
