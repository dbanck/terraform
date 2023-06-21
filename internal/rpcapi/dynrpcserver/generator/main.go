// This program is the generator for the gRPC service wrapper types in the
// parent directory. It's not suitable for any other use.
//
// This makes various assumptions about how the protobuf compiler and
// gRPC stub generators produce code. If those significantly change in future
// then this will probably break.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

const protobufPkg = "github.com/hashicorp/terraform/internal/rpcapi/terraform1"

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, protobufPkg)
	if err != nil {
		log.Fatalf("can't load the protobuf/gRPC proxy package: %s", err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("wrong number of packages found")
	}
	pkg := pkgs[0]
	if pkg.TypesInfo == nil {
		log.Fatalf("types info not available")
	}
	if len(pkg.GoFiles) < 1 {
		log.Fatalf("no files included in package")
	}

	// We assume that our output directory is sibling to the directory
	// containing the protobuf specification.
	outDir := filepath.Join(filepath.Dir(pkg.GoFiles[0]), "../dynrpcserver")

Types:
	for _, obj := range pkg.TypesInfo.Defs {
		typ, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		underTyp := typ.Type().Underlying()
		iface, ok := underTyp.(*types.Interface)
		if !ok {
			continue
		}
		if !strings.HasSuffix(typ.Name(), "Server") || typ.Name() == "SetupServer" {
			// Doesn't look like a generated gRPC server interface
			continue
		}

		// The interfaces used for streaming requests/responses unfortunately
		// also have a "Server" suffix in the generated Go code, and so
		// we need to detect those more surgically by noticing that they
		// have grpc.ServerStream embedded inside.
		for i := 0; i < iface.NumEmbeddeds(); i++ {
			emb, ok := iface.EmbeddedType(i).(*types.Named)
			if !ok {
				continue
			}
			pkg := emb.Obj().Pkg().Path()
			name := emb.Obj().Name()
			if pkg == "google.golang.org/grpc" && name == "ServerStream" {
				continue Types
			}
		}

		// If we get here then what we're holding _seems_ to be a gRPC
		// server interface, and so we'll generate a dynamic initialization
		// wrapper for it.

		ifaceName := typ.Name()
		baseName := strings.TrimSuffix(ifaceName, "Server")
		filename := toFilenameCase(baseName) + ".go"
		absFilename := filepath.Join(outDir, filename)

		var buf bytes.Buffer

		fmt.Fprintf(&buf, `package dynrpcserver

			import (
				"context"
				"sync"

				tf1 %q
			)

		`, protobufPkg)
		fmt.Fprintf(&buf, "type %s struct {\n", baseName)
		fmt.Fprintf(&buf, "impl tf1.%s\n", ifaceName)
		fmt.Fprintln(&buf, "mu sync.RWMutex")
		buf.WriteString("}\n\n")

		fmt.Fprintf(&buf, "var _ tf1.%s = (*%s)(nil)\n\n", ifaceName, baseName)

		fmt.Fprintf(&buf, "func New%sStub() *%s {\n", baseName, baseName)
		fmt.Fprintf(&buf, "return &%s{}\n", baseName)
		fmt.Fprintf(&buf, "}\n\n")

		for i := 0; i < iface.NumMethods(); i++ {
			method := iface.Method(i)
			sig := method.Type().(*types.Signature)

			fmt.Fprintf(&buf, "func (s *%s) %s(", baseName, method.Name())
			for i := 0; i < sig.Params().Len(); i++ {
				param := sig.Params().At(i)

				// The generated interface types don't include parameter names
				// and so we just use synthetic parameter names here.
				name := fmt.Sprintf("a%d", i)
				genType := typeRef(param.Type().String())

				if i > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(name)
				buf.WriteString(" ")
				buf.WriteString(genType)
			}
			fmt.Fprintf(&buf, ")")
			if sig.Results().Len() > 1 {
				buf.WriteString("(")
			}
			for i := 0; i < sig.Results().Len(); i++ {
				result := sig.Results().At(i)
				genType := typeRef(result.Type().String())
				if i > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(genType)
			}
			if sig.Results().Len() > 1 {
				buf.WriteString(")")
			}
			switch n := sig.Results().Len(); n {
			case 1:
				fmt.Fprintf(&buf, ` {
				impl, err := s.realRPCServer()
				if err != nil {
					return err
				}
			`)
			case 2:
				fmt.Fprintf(&buf, ` {
				impl, err := s.realRPCServer()
				if err != nil {
					return nil, err
				}
			`)
			default:
				log.Fatalf("don't know how to make a stub for method with %d results", n)
			}
			fmt.Fprintf(&buf, "return impl.%s(", method.Name())
			for i := 0; i < sig.Params().Len(); i++ {
				if i > 0 {
					buf.WriteString(", ")
				}
				fmt.Fprintf(&buf, "a%d", i)
			}
			fmt.Fprintf(&buf, ")\n}\n\n")
		}

		fmt.Fprintf(&buf, `
			func (s *%s) ActivateRPCServer(impl tf1.%s) {
				s.mu.Lock()
				s.impl = impl
				s.mu.Unlock()
			}

			func (s *%s) realRPCServer() (tf1.%s, error) {
				s.mu.RLock()
				impl := s.impl
				s.mu.RUnlock()
				if impl == nil {
					return nil, unavailableErr
				}
				return impl, nil
			}
		`, baseName, ifaceName, baseName, ifaceName)

		src, err := format.Source(buf.Bytes())
		if err != nil {
			//log.Fatalf("formatting %s: %s", filename, err)
			src = buf.Bytes()
		}
		f, err := os.Create(absFilename)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write(src)
		if err != nil {
			log.Fatalf("writing %s: %s", filename, err)
		}

	}
}

func typeRef(fullType string) string {
	// The following is specialized to only the parameter types
	// we typically expect to see in a server interface. This
	// might need extra rules if we step outside the design idiom
	// we've used for these services so far.
	switch {
	case fullType == "context.Context" || fullType == "error":
		return fullType
	case fullType == "interface{}" || fullType == "any":
		return "any"
	case strings.HasPrefix(fullType, "*"+protobufPkg+"."):
		return "*tf1." + fullType[len(protobufPkg)+2:]
	case strings.HasPrefix(fullType, protobufPkg+"."):
		return "tf1." + fullType[len(protobufPkg)+1:]
	default:
		log.Fatalf("don't know what to do with parameter type %s", fullType)
		return ""
	}
}

var firstCapPattern = regexp.MustCompile("(.)([A-Z][a-z]+)")
var otherCapPattern = regexp.MustCompile("([a-z0-9])([A-Z])")

func toFilenameCase(typeName string) string {
	ret := firstCapPattern.ReplaceAllString(typeName, "${1}_${2}")
	ret = otherCapPattern.ReplaceAllString(ret, "${1}_${2}")
	return strings.ToLower(ret)
}
